package http // Keeps health endpoint logic in HTTP layer.

import (
	"encoding/json" // Encodes Go structs/maps into JSON response bodies.
	"net/http"      // Provides request/response objects and status codes.
)

func healthHandler(dbChecker func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := dbChecker(); err != nil { // Check database connectivity at request time.
			w.Header().Set("Content-Type", "application/json") // Return JSON on failure.
			w.WriteHeader(http.StatusServiceUnavailable)       // 503 means dependency is unhealthy.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "down",        // Overall service status.
				"db":     "unreachable", // Dependency status.
				"error":  err.Error(),   // Human-readable failure detail.
				"path":   r.URL.Path,    // Helpful debug field for caller.
				"method": r.Method,      // Helpful debug field for caller.
			})
			return // Stop processing after writing failure response.
		}

		w.Header().Set("Content-Type", "application/json") // Return JSON on success.
		w.WriteHeader(http.StatusOK)                       // 200 indicates healthy service.
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",      // Overall health.
			"db":     "healthy", // Database health.
		})
	}
}
