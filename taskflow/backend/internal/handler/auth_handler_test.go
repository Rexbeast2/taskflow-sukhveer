package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthHandler_Register(t *testing.T) {
	svc := new(mockAuthService)
	h := NewAuthHandler(svc, discardLogger())

	t.Run("success - returns 201 and token", func(t *testing.T) {
		reqBody := registerRequest{
			Name:     "Alice",
			Email:    "alice@example.com",
			Password: "securepassword",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		expectedUser := &schema.User{ID: uuid.New(), Name: "Alice", Email: "alice@example.com"}
		svc.On("Register", mock.Anything, schema.RegisterInput{
			Name:     reqBody.Name,
			Email:    reqBody.Email,
			Password: reqBody.Password,
		}).Return(&schema.AuthOutput{Token: "fake-jwt-token", User: expectedUser}, nil).Once()

		h.Register(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp authResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, "fake-jwt-token", resp.Token)
		assert.Equal(t, expectedUser.ID, resp.User.ID)
	})

}

func TestAuthHandler_Login(t *testing.T) {
	svc := new(mockAuthService)
	h := NewAuthHandler(svc, discardLogger())

	t.Run("success - returns 200", func(t *testing.T) {
		reqBody := loginRequest{Email: "alice@example.com", Password: "password123"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		svc.On("Login", mock.Anything, schema.LoginInput{
			Email:    reqBody.Email,
			Password: reqBody.Password,
		}).Return(&schema.AuthOutput{Token: "login-token", User: &schema.User{}}, nil).Once()

		h.Login(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("fail - invalid credentials", func(t *testing.T) {
		reqBody := loginRequest{Email: "wrong@example.com", Password: "wrong"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		svc.On("Login", mock.Anything, mock.Anything).
			Return(nil, schema.ErrInvalidPassword).Once()

		h.Login(rr, req)

		// schemaErrorToHTTP should map ErrInvalidPassword to 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("fail - malformed json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("{invalid json")))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
