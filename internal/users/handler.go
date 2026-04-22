package users // Exposes HTTP handlers that map requests/responses to service methods.

import (
	"encoding/json" // Uses JSON encoder/decoder for request and response payloads.
	"errors"        // Uses errors.Is for matching domain errors like ErrUserNotFound.
	"net/http"      // Uses HTTP status codes, Request, and ResponseWriter types.
	"strconv"       // Uses ParseInt to convert URL id string into int64.

	"github.com/go-chi/chi/v5" // Uses chi router helpers for route registration and URL params.
)

type Handler struct { // Holds service dependency for all HTTP endpoint methods.
	service *Service // Pointer to users service injected from main wiring.
}

func NewHandler(service *Service) *Handler { // Constructor to build handler with service dependency.
	return &Handler{service: service} // Returns pointer used by router registration.
}

func (h *Handler) RegisterRoutes(r chi.Router) { // Registers all users module routes under parent router group.
	r.Post("/", h.create)                      // Creates new user via POST /users.
	r.Get("/", h.list)                         // Lists users via GET /users.
	r.Get("/{id}", h.get)                      // Gets one user by ID via GET /users/{id}.
	r.Put("/{id}", h.update)                   // Updates user via PUT /users/{id}.
	r.Delete("/{id}", h.delete)                // Deletes user via DELETE /users/{id}.
	r.Post("/{id}/login", h.login)             // Verifies password via POST /users/{id}/login.
	r.Post("/{id}/password/change", h.changePassword) // Changes password via POST /users/{id}/password/change.
	r.Post("/{id}/2fa/send", h.sendTOTP)       // Generates/sends 2FA code via POST /users/{id}/2fa/send.
	r.Post("/{id}/2fa/verify", h.verifyTOTP)   // Verifies 2FA code via POST /users/{id}/2fa/verify.
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) { // Handles create-user endpoint.
	var in CreateUserInput // Declares struct to receive decoded JSON body.
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil { // Decodes request body JSON into input struct.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Returns 400 when JSON is malformed.
		return // Stops handler after writing error response.
	}
	u, err := h.service.CreateUser(r.Context(), in) // Calls service layer with request context and input.
	if err != nil { // Handles validation or repository failures.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Returns 400 with service error message.
		return // Stops handler after error response.
	}
	writeJSON(w, http.StatusCreated, u) // Returns 201 with created user payload.
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) { // Handles list-users endpoint.
	users, err := h.service.ListUsers(r.Context()) // Fetches all users from service layer.
	if err != nil { // Handles unexpected read failure.
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Returns 500 for server-side failure.
		return // Stops handler after error response.
	}
	writeJSON(w, http.StatusOK, users) // Returns 200 with users slice as JSON.
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) { // Handles get-user-by-id endpoint.
	id, ok := parseID(w, r) // Parses `{id}` path parameter into int64.
	if !ok { // Checks whether parseID already wrote error response.
		return // Stops early when ID is invalid.
	}
	u, err := h.service.GetUser(r.Context(), id) // Calls service to load one user.
	if err != nil { // Handles not-found and internal errors.
		if errors.Is(err, ErrUserNotFound) { // Matches shared domain not-found error.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 for missing user.
			return // Stops handler after 404 response.
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Returns 500 for unexpected failures.
		return // Stops handler after error response.
	}
	writeJSON(w, http.StatusOK, u) // Returns 200 with found user payload.
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) { // Handles update-user endpoint.
	id, ok := parseID(w, r) // Parses and validates user ID from URL.
	if !ok { // Checks if parsing failed.
		return // Stops because parseID already wrote 400.
	}
	var in UpdateUserInput // Declares struct for update request body.
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil { // Decodes JSON payload into struct.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Returns 400 when JSON is invalid.
		return // Stops handler after bad request response.
	}
	u, err := h.service.UpdateUser(r.Context(), id, in) // Calls service to validate and persist update.
	if err != nil { // Handles not-found or validation failures.
		if errors.Is(err, ErrUserNotFound) { // Detects missing user record.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 when ID does not exist.
			return // Stops handler after 404 response.
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Returns 400 for validation/business errors.
		return // Stops handler after error response.
	}
	writeJSON(w, http.StatusOK, u) // Returns 200 with updated user payload.
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) { // Handles delete-user endpoint.
	id, ok := parseID(w, r) // Parses user ID from route parameter.
	if !ok { // Checks parse result.
		return // Stops because parseID handled error response.
	}
	if err := h.service.DeleteUser(r.Context(), id); err != nil { // Calls service to delete target user.
		if errors.Is(err, ErrUserNotFound) { // Handles user-not-found case.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 when delete target missing.
			return // Stops handler after 404 response.
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Returns 500 for unexpected delete errors.
		return // Stops handler after error response.
	}
	w.WriteHeader(http.StatusNoContent) // Returns 204 with empty body on successful delete.
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) { // Handles password verification step before 2FA.
	id, ok := parseID(w, r) // Parses user ID from URL.
	if !ok { // Checks parse result.
		return // Stops on invalid ID.
	}
	var body struct { // Declares inline request body type for login endpoint.
		Password string `json:"password"` // Expects `password` field from client JSON.
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // Decodes login request body.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Returns 400 when JSON is invalid.
		return // Stops after error response.
	}
	if err := h.service.Login(r.Context(), id, body.Password); err != nil { // Calls service to verify credentials and mark login.
		if errors.Is(err, ErrUserNotFound) { // Checks if user does not exist.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 for missing user.
			return // Stops after 404 response.
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()}) // Returns 401 for invalid credentials.
		return // Stops after auth failure response.
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password verified, continue with 2FA verify"}) // Returns success guidance for next step.
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) { // Handles password change endpoint.
	id, ok := parseID(w, r) // Parses and validates user ID path parameter.
	if !ok { // Checks parse result.
		return // Stops when ID is invalid.
	}
	var body struct { // Defines expected request body fields for password change.
		OldPassword string `json:"old_password"` // Existing password used for verification.
		NewPassword string `json:"new_password"` // Replacement password to hash and store.
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // Decodes request JSON payload.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Returns 400 for malformed JSON.
		return // Stops after bad request response.
	}
	if err := h.service.ChangePassword(r.Context(), id, body.OldPassword, body.NewPassword); err != nil { // Calls service to validate old password and update hash.
		if errors.Is(err, ErrUserNotFound) { // Handles missing user.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 when user does not exist.
			return // Stops after 404 response.
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Returns 400 for validation/business rule failures.
		return // Stops after error response.
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed"}) // Returns 200 confirmation on success.
}

func (h *Handler) sendTOTP(w http.ResponseWriter, r *http.Request) { // Handles endpoint that generates/sends 2FA code.
	id, ok := parseID(w, r) // Parses user ID from route.
	if !ok { // Checks parse status.
		return // Stops on invalid ID.
	}
	if err := h.service.SendTOTP(r.Context(), id); err != nil { // Calls service to create current TOTP code.
		if errors.Is(err, ErrUserNotFound) { // Handles missing user case.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 when user ID is unknown.
			return // Stops after 404 response.
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Returns 400 for other business errors.
		return // Stops after error response.
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA code sent (check server logs for demo)"}) // Returns 200 success message for demo flow.
}

func (h *Handler) verifyTOTP(w http.ResponseWriter, r *http.Request) { // Handles endpoint to verify submitted 2FA code.
	id, ok := parseID(w, r) // Parses user ID from URL.
	if !ok { // Checks parse result.
		return // Stops on invalid ID.
	}
	var body struct { // Declares expected request body for verification endpoint.
		Code string `json:"code"` // Expects `code` field containing TOTP token.
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { // Decodes JSON verification body.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Returns 400 for malformed JSON.
		return // Stops after bad request response.
	}
	if err := h.service.VerifyTOTP(r.Context(), id, body.Code); err != nil { // Calls service to validate code.
		if errors.Is(err, ErrUserNotFound) { // Handles unknown user.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Returns 404 when user does not exist.
			return // Stops after 404 response.
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()}) // Returns 401 for invalid or expired code.
		return // Stops after auth error response.
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA verified"}) // Returns 200 when code is valid.
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) { // Shared helper to parse and validate URL `{id}` parameter.
	idStr := chi.URLParam(r, "id") // Reads raw `id` path segment using chi helper.
	id, err := strconv.ParseInt(idStr, 10, 64) // Converts base-10 string to int64 for service/repository usage.
	if err != nil || id <= 0 { // Rejects non-numeric, overflow, and non-positive IDs.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"}) // Writes 400 response when ID is invalid.
		return 0, false // Returns false so caller exits early.
	}
	return id, true // Returns parsed ID and success flag.
}

func writeJSON(w http.ResponseWriter, status int, data any) { // Shared helper to write JSON responses consistently.
	w.Header().Set("Content-Type", "application/json") // Sets JSON content type header for clients.
	w.WriteHeader(status) // Writes provided HTTP status code before body.
	_ = json.NewEncoder(w).Encode(data) // Encodes response payload to JSON and writes it to response body.
}
