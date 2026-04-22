package http // Contains HTTP middleware utilities for request authentication.

import (
	"context"  // Uses context.WithValue to store authenticated user ID in request context.
	"net/http" // Uses net/http middleware interfaces and status codes.
	"strings"  // Uses prefix trimming for Bearer token parsing.

	"go-crm-learning-api/internal/auth" // Uses JWT manager to validate bearer tokens.
)

type contextKey string // Defines custom context key type to avoid collisions.

const authUserIDContextKey contextKey = "auth_user_id" // Stores authenticated user ID in request context.

func requireAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler { // Creates middleware that enforces valid JWT bearer token.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization") // Reads Authorization header from incoming request.
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header") // Returns 401 when header is absent.
				return
			}

			const bearerPrefix = "Bearer " // Expected authorization scheme prefix.
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				writeUnauthorized(w, "authorization header must use Bearer token") // Returns 401 for invalid auth scheme.
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix)) // Extracts raw JWT string from header.
			if token == "" {
				writeUnauthorized(w, "missing bearer token") // Returns 401 when token value is empty.
				return
			}

			userID, err := jwtManager.ParseAccessToken(token) // Validates access JWT and extracts authenticated user ID.
			if err != nil {
				writeUnauthorized(w, "invalid or expired token") // Returns 401 for parse/signature/expiry failures.
				return
			}

			ctx := context.WithValue(r.Context(), authUserIDContextKey, userID) // Stores authenticated user ID for downstream handlers.
			next.ServeHTTP(w, r.WithContext(ctx))                               // Continues request chain with enriched context.
		})
	}
}

func AuthenticatedUserID(ctx context.Context) (int64, bool) { // Reads authenticated user ID from request context for handler/service usage.
	userID, ok := ctx.Value(authUserIDContextKey).(int64) // Type-asserts stored context value to int64.
	return userID, ok                                     // Returns user ID and presence flag.
}
