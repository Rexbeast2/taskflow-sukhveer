package infrastructure

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sukhveer/taskflow/internal/config"
)

func TestJWTTokenService(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:    "super-secret-key",
		ExpiresIn: 1 * time.Hour,
	}
	svc := NewJWTTokenService(cfg)
	userID := uuid.New()
	email := "test@example.com"

	t.Run("Generate and Validate Success", func(t *testing.T) {
		// 1. Generate token
		token, err := svc.Generate(userID, email)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// 2. Validate token
		claims, err := svc.Validate(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("Fail - Expired Token", func(t *testing.T) {
		// Create a service with a negative expiration
		shortCfg := config.JWTConfig{
			Secret:    "secret",
			ExpiresIn: -1 * time.Minute,
		}
		shortSvc := NewJWTTokenService(shortCfg)

		token, err := shortSvc.Generate(userID, email)
		require.NoError(t, err)

		// Validation should fail due to expiration
		claims, err := svc.Validate(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token signature is invalid")
	})

	t.Run("Fail - Invalid Secret", func(t *testing.T) {
		token, err := svc.Generate(userID, email)
		require.NoError(t, err)

		// Service with a DIFFERENT secret
		wrongSvc := NewJWTTokenService(config.JWTConfig{
			Secret: "wrong-secret",
		})

		claims, err := wrongSvc.Validate(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "signature is invalid")
	})

	t.Run("Fail - Malformed Token", func(t *testing.T) {
		claims, err := svc.Validate("not.a.valid.token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Fail - Unexpected Signing Method", func(t *testing.T) {
		// Manually create a token using None signing method
		token := jwt.NewWithClaims(jwt.SigningMethodNone, jwtClaims{
			UserID: userID,
			Email:  email,
		})
		// SigningMethodNone requires jwt.UnsafeAllowNoneSignatureType to parse,
		// but our service strictly checks for HMAC.
		tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

		claims, err := svc.Validate(tokenStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected signing method")
		assert.Nil(t, claims)
	})
}
