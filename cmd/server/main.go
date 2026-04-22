package main // Defines the executable entrypoint package.

import (
	"context"  // Carries cancellation and deadlines through startup calls.
	"log"      // Writes structured startup and fatal logs.
	"net/http" // Runs the HTTP server.
	"time"     // Defines timeout durations.

	"go-crm-learning-api/internal/auth"
	"go-crm-learning-api/internal/config" // Loads app configuration from environment.
	"go-crm-learning-api/internal/customers"
	httpapi "go-crm-learning-api/internal/http"      // Wires API routes and handlers.
	"go-crm-learning-api/internal/platform/database" // Creates PostgreSQL connection pool.
	"go-crm-learning-api/internal/users"
)

func main() {
	cfg, err := config.Load() // Load and validate environment configuration.
	if err != nil {           // Stop if config is invalid.
		log.Fatalf("load config: %v", err) // Exit with clear message.
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Bound DB init time.
	defer cancel()                                                           // Ensure context resources are released.

	pool, err := database.NewPool(ctx, cfg.DBURL) // Create and ping PostgreSQL pool.
	if err != nil {                               // Stop on DB bootstrap failure.
		log.Fatalf("connect database: %v", err) // Exit with clear message.
	}
	defer pool.Close() // Close all DB connections during shutdown.

	userRepo := users.NewRepository(pool) // Create users repository using shared DB pool.
	if err := userRepo.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, time.Duration(cfg.JWTTTLMinutes)*time.Minute, time.Duration(cfg.JWTRefreshTTLHours)*time.Hour) // Create shared JWT manager for access/refresh token lifecycle.
	userService := users.NewService(userRepo)                                                                                                      // Create users business layer.
	userHandler := users.NewHandler(userService, jwtManager)                                                                                       // Create users HTTP handler.
	customerRepo := customers.NewRepository(pool)                                                                                                  // Create customers repository using shared DB pool.
	if err := customerRepo.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	customerService := customers.NewService(customerRepo)    // Create customers business layer.
	customerHandler := customers.NewHandler(customerService) // Create customers HTTP handler.

	router := httpapi.NewRouter(func() error { // Build router with DB health callback.
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second) // Bound ping time.
		defer pingCancel()                                                              // Release ping context.
		return pool.Ping(pingCtx)                                                       // Health check DB at request time.
	}, userHandler, customerHandler, jwtManager)

	server := &http.Server{ // Configure HTTP server instance.
		Addr:              ":" + cfg.HTTPPort, // Listen on configured port.
		Handler:           router,             // Use our application router.
		ReadHeaderTimeout: 5 * time.Second,    // Protect against slowloris header attacks.
	}

	log.Printf("server starting on :%s (%s)", cfg.HTTPPort, cfg.AppEnv) // Print startup info.

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed { // Start blocking server loop.
		log.Fatalf("http server error: %v", err) // Exit if server crashes unexpectedly.
	}
}
