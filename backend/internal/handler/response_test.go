package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestRespond(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]string{"message": "hello"}

	respond(w, http.StatusAccepted, payload)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusAccepted, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	assert.Equal(t, "hello", body["message"])
}

func TestSchemaErrorToHTTP(t *testing.T) {
	tests := []struct {
		name           string
		inputErr       error
		expectedStatus int
		expectedError  string
		expectedFields map[string]string
	}{
		{
			name:           "Unauthorized - Invalid Password",
			inputErr:       schema.ErrInvalidPassword,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
		},
		{
			name:           "Forbidden - Not Owner",
			inputErr:       schema.ErrNotProjectOwner,
			expectedStatus: http.StatusForbidden,
			expectedError:  "forbidden: only the project owner can perform this action",
		},
		{
			name:           "Not Found - Project Not Found",
			inputErr:       schema.ErrProjectNotFound,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
		},
		{
			name:           "Conflict - Email Taken",
			inputErr:       schema.ErrEmailAlreadyTaken,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation failed",
			expectedFields: map[string]string{"email": "already taken"},
		},
		{
			name:           "Validation - Name Required",
			inputErr:       schema.ErrNameRequired,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation failed",
			expectedFields: map[string]string{"name": "is required"},
		},
		{
			name:           "Internal Server Error - Unknown Error",
			inputErr:       errors.New("something went bang"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			schemaErrorToHTTP(w, tt.inputErr)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			var actualBody errorResponse
			json.NewDecoder(res.Body).Decode(&actualBody)

			assert.Equal(t, tt.expectedError, actualBody.Error)
			if tt.expectedFields != nil {
				assert.Equal(t, tt.expectedFields, actualBody.Fields)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	type testInput struct {
		Name string `json:"name"`
	}

	t.Run("valid json", func(t *testing.T) {
		body := `{"name": "taskflow"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		var input testInput
		err := decode(req, &input)

		assert.NoError(t, err)
		assert.Equal(t, "taskflow", input.Name)
	})

	t.Run("unknown fields fail", func(t *testing.T) {
		// Because you use dec.DisallowUnknownFields()
		body := `{"name": "taskflow", "extra": "data"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		var input testInput
		err := decode(req, &input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("invalid json structure", func(t *testing.T) {
		body := `{"name": "taskflow"` // Missing closing brace
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		var input testInput
		err := decode(req, &input)

		assert.Error(t, err)
	})
}
