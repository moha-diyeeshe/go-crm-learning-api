package customers // Exposes HTTP handlers for customer CRUD endpoints.

import (
	"encoding/json" // Uses JSON encoder/decoder for request/response payloads.
	"errors"        // Uses errors.Is for matching domain-level not-found error.
	"net/http"      // Uses HTTP types and status codes.
	"strconv"       // Uses ParseInt for converting URL path IDs.

	"github.com/go-chi/chi/v5" // Uses chi router for route registration and URL params.
)

type Handler struct { // Holds service dependency used by route handlers.
	service *Service // Injected customer service that executes business logic.
}

func NewHandler(service *Service) *Handler { // Constructor for handler dependency injection.
	return &Handler{service: service} // Returns shared handler pointer.
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) { // Registers customer CRUD routes behind JWT auth middleware.
	r.Group(func(protected chi.Router) { // Groups all customer CRUD endpoints as protected resources.
		protected.Use(authMiddleware)       // Applies JWT middleware so only logged-in users can access customer routes.
		protected.Post("/", h.create)       // Maps POST /customers to create handler.
		protected.Get("/", h.list)          // Maps GET /customers to list handler.
		protected.Get("/{id}", h.get)       // Maps GET /customers/{id} to get handler.
		protected.Put("/{id}", h.update)    // Maps PUT /customers/{id} to update handler.
		protected.Delete("/{id}", h.delete) // Maps DELETE /customers/{id} to delete handler.
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) { // Handles customer creation endpoint.
	var in CreateCustomerInput                                  // Declares DTO for decoded create payload.
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil { // Decodes JSON body into struct.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Responds with 400 on malformed JSON.
		return                                                                               // Stops processing after writing error response.
	}
	customer, err := h.service.CreateCustomer(r.Context(), in) // Calls service create flow with request context.
	if err != nil {                                            // Handles validation and repository errors.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Responds with 400 for bad input/conflicts.
		return                                                                       // Stops processing on service error.
	}
	writeJSON(w, http.StatusCreated, customer) // Responds with 201 and created customer body.
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) { // Handles customer list endpoint.
	customers, err := h.service.ListCustomers(r.Context()) // Calls service to fetch all customers.
	if err != nil {                                        // Handles unexpected server errors.
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Responds with 500 for internal failures.
		return                                                                                // Stops processing after error response.
	}
	writeJSON(w, http.StatusOK, customers) // Responds with 200 and customer array.
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) { // Handles fetch-customer-by-ID endpoint.
	id, ok := parseID(w, r) // Parses and validates path ID.
	if !ok {                // Checks parse result.
		return // Stops because parseID already wrote the response.
	}
	customer, err := h.service.GetCustomer(r.Context(), id) // Calls service to fetch customer by ID.
	if err != nil {                                         // Handles not-found and internal errors.
		if errors.Is(err, ErrCustomerNotFound) { // Detects domain not-found error.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Responds with 404 when customer is missing.
			return                                                                     // Stops after 404 response.
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Responds with 500 on unexpected failures.
		return                                                                                // Stops after error response.
	}
	writeJSON(w, http.StatusOK, customer) // Responds with 200 and customer body.
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) { // Handles customer update endpoint.
	id, ok := parseID(w, r) // Parses and validates customer ID from URL.
	if !ok {                // Checks parse result.
		return // Stops on invalid ID.
	}
	var in UpdateCustomerInput                                  // Declares DTO for update payload.
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil { // Decodes JSON body into update struct.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"}) // Responds with 400 for malformed JSON.
		return                                                                               // Stops after bad request response.
	}
	customer, err := h.service.UpdateCustomer(r.Context(), id, in) // Calls service update flow.
	if err != nil {                                                // Handles not-found and validation/conflict errors.
		if errors.Is(err, ErrCustomerNotFound) { // Detects missing customer.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Responds with 404 when target doesn't exist.
			return                                                                     // Stops after 404 response.
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()}) // Responds with 400 for validation/conflict errors.
		return                                                                       // Stops after error response.
	}
	writeJSON(w, http.StatusOK, customer) // Responds with 200 and updated customer.
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) { // Handles customer deletion endpoint.
	id, ok := parseID(w, r) // Parses and validates URL ID.
	if !ok {                // Checks parse result.
		return // Stops on invalid ID.
	}
	if err := h.service.DeleteCustomer(r.Context(), id); err != nil { // Calls service delete flow.
		if errors.Is(err, ErrCustomerNotFound) { // Detects missing customer case.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()}) // Responds with 404 when record not found.
			return                                                                     // Stops after 404 response.
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}) // Responds with 500 for unexpected errors.
		return                                                                                // Stops after error response.
	}
	w.WriteHeader(http.StatusNoContent) // Responds with 204 and empty body on successful delete.
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) { // Shared helper that parses `{id}` route parameter safely.
	idStr := chi.URLParam(r, "id")             // Reads `id` segment from route path.
	id, err := strconv.ParseInt(idStr, 10, 64) // Converts string ID to int64.
	if err != nil || id <= 0 {                 // Rejects non-numeric and non-positive IDs.
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid customer id"}) // Responds with 400 for invalid IDs.
		return 0, false                                                                        // Returns failure flag to caller.
	}
	return id, true // Returns parsed ID and success flag.
}

func writeJSON(w http.ResponseWriter, status int, data any) { // Shared helper for consistent JSON responses.
	w.Header().Set("Content-Type", "application/json") // Sets response content type header.
	w.WriteHeader(status)                              // Writes provided HTTP status code.
	_ = json.NewEncoder(w).Encode(data)                // Encodes data as JSON and writes response body.
}
