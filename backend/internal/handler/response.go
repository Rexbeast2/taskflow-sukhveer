package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// respond writes a JSON response with the given status code
func respond(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			slog.Error("encode response", "error", err)
		}
	}
}

// errorResponse is the structured error body
type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// respondError writes a structured JSON error
func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, errorResponse{Error: message})
}

// respondValidationError writes a 400 with field-level errors
func respondValidationError(w http.ResponseWriter, fields map[string]string) {
	respond(w, http.StatusBadRequest, errorResponse{
		Error:  "validation failed",
		Fields: fields,
	})
}

// schemaErrorToHTTP maps known schema errors to HTTP status codes and messages
func schemaErrorToHTTP(w http.ResponseWriter, err error) {
	switch {
	// 401 — auth errors
	case errors.Is(err, schema.ErrInvalidPassword):
		respondError(w, http.StatusUnauthorized, "invalid credentials")

	// 403 — authorization errors
	case errors.Is(err, schema.ErrNotProjectOwner):
		respondError(w, http.StatusForbidden, "forbidden: only the project owner can perform this action")

	// 404 — not found or access denied (we intentionally conflate these to
	// prevent resource enumeration — a stranger cannot tell whether a project
	// exists but they lack access, or simply doesn't exist)
	case errors.Is(err, schema.ErrUserNotFound),
		errors.Is(err, schema.ErrProjectNotFound),
		errors.Is(err, schema.ErrTaskNotFound),
		errors.Is(err, schema.ErrAccessDenied):
		respondError(w, http.StatusNotFound, "not found")

	// 409 — conflict
	case errors.Is(err, schema.ErrEmailAlreadyTaken):
		respondValidationError(w, map[string]string{"email": "already taken"})

	// 400 — validation errors
	case errors.Is(err, schema.ErrNameRequired):
		respondValidationError(w, map[string]string{"name": "is required"})
	case errors.Is(err, schema.ErrInvalidEmail):
		respondValidationError(w, map[string]string{"email": "is invalid"})
	case errors.Is(err, schema.ErrPasswordTooShort):
		respondValidationError(w, map[string]string{"password": "must be at least 8 characters"})
	case errors.Is(err, schema.ErrProjectNameRequired):
		respondValidationError(w, map[string]string{"name": "is required"})
	case errors.Is(err, schema.ErrTaskTitleRequired):
		respondValidationError(w, map[string]string{"title": "is required"})
	case errors.Is(err, schema.ErrInvalidStatus):
		respondValidationError(w, map[string]string{"status": "must be todo, in_progress, or done"})
	case errors.Is(err, schema.ErrInvalidPriority):
		respondValidationError(w, map[string]string{"priority": "must be low, medium, or high"})

	default:
		slog.Error("unhandled service error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// decode reads and decodes JSON from the request body
func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
