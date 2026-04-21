package http // Contains HTTP layer wiring (router + handlers).

import (
	"net/http" // Provides HTTP primitives and handler interfaces.

	"github.com/go-chi/chi/v5" // Lightweight router for building REST APIs.
	"go-crm-learning-api/internal/users"
)

func NewRouter(dbChecker func() error, usersHandler *users.Handler) http.Handler {
	r := chi.NewRouter() // Create a new chi router instance.

	r.Get("/health", healthHandler(dbChecker)) // Register health endpoint.
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Route("/users", usersHandler.RegisterRoutes)
	})

	return r // Return ready-to-use router.
}
