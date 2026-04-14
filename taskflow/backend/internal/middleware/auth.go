package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// contextKey is an unexported type to prevent collisions in context values
type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyEmail  contextKey = "email"
)

// Authenticate is a middleware that validates the Bearer JWT token
// and injects the user's claims into the request context.
func Authenticate(tokenService schema.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w, "invalid authorization header format")
				return
			}

			claims, err := tokenService.Validate(parts[1])
			if err != nil {
				slog.Debug("token validation failed", "error", err)
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's UUID from the context.
// Panics if the middleware was not applied — which is a programmer error.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	v, ok := ctx.Value(contextKeyUserID).(uuid.UUID)
	if !ok {
		panic("middleware.UserIDFromContext: user_id not found in context — ensure Authenticate middleware is applied")
	}
	return v
}

// used only for mocking in tests, not exported
func SetUserContext(ctx context.Context, userID uuid.UUID, email string) context.Context {
	ctx = context.WithValue(ctx, contextKeyUserID, userID)
	ctx = context.WithValue(ctx, contextKeyEmail, email)
	return ctx
}

// EmailFromContext extracts the authenticated user's email from the context.
func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyEmail).(string)
	return v
}

// writeUnauthorized sends a 401 JSON response
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

// RequestLogger logs each incoming request with method, path, and status
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// loggingResponseWriter captures the status code for logging
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Recoverer catches panics in handlers and returns a 500
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"internal server error"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
