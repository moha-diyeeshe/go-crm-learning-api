package http // Contains HTTP layer wiring (router + handlers).

import (
	"net/http" // Provides HTTP primitives and handler interfaces.

	"github.com/go-chi/chi/v5" // Lightweight router for building REST APIs.
	"go-crm-learning-api/internal/auth"
	"go-crm-learning-api/internal/customers"
	"go-crm-learning-api/internal/users"
)

func NewRouter(dbChecker func() error, usersHandler *users.Handler, customersHandler *customers.Handler, jwtManager *auth.JWTManager) http.Handler {
	r := chi.NewRouter()                      // Create a new chi router instance.
	r.Use(requestLogger)                      // Applies request logging middleware to every route on this router.
	authMiddleware := requireAuth(jwtManager) // Builds reusable JWT auth middleware for protected routes.

	r.Get("/health", healthHandler(dbChecker)) // Register health endpoint.
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Route("/users", func(usersRoutes chi.Router) {
			usersHandler.RegisterRoutes(usersRoutes, authMiddleware) // Register users routes with public auth flow and protected CRUD.
		})
		v1.Route("/customers", func(customersRoutes chi.Router) {
			customersHandler.RegisterRoutes(customersRoutes, authMiddleware) // Register customers CRUD routes as protected endpoints.
		})
	})

	return r // Return ready-to-use router.
}
