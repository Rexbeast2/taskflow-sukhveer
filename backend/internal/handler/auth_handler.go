package handler

import (
	"log/slog"
	"net/http"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService schema.AuthService
	logger      *slog.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService schema.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{authService: authService, logger: logger}
}

// registerRequest is the incoming body for POST /auth/register
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the incoming body for POST /auth/login
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse is the outgoing body on successful auth
type authResponse struct {
	Token string       `json:"token"`
	User  *schema.User `json:"user"`
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if req.Name == "" {
		fields["name"] = "is required"
	}
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if req.Password == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		respondValidationError(w, fields)
		return
	}

	out, err := h.authService.Register(r.Context(), schema.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusCreated, authResponse{Token: out.Token, User: out.User})
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if req.Password == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		respondValidationError(w, fields)
		return
	}

	out, err := h.authService.Login(r.Context(), schema.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusOK, authResponse{Token: out.Token, User: out.User})
}
