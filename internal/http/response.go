package http // Contains shared HTTP response helpers used by middleware and handlers.

import (
	"encoding/json" // Uses JSON encoder for error response bodies.
	"net/http"      // Uses status codes and ResponseWriter interface.
)

func writeUnauthorized(w http.ResponseWriter, message string) { // Writes a standardized 401 JSON response.
	w.Header().Set("Content-Type", "application/json") // Marks payload content type as JSON.
	w.WriteHeader(http.StatusUnauthorized)             // Sends HTTP 401 unauthorized status code.
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message, // Includes readable authentication failure reason.
	})
}
