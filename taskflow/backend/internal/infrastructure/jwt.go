package infrastructure

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/config"
	"github.com/Sukhveer/taskflow/internal/schema"
)

// jwtClaims extends standard JWT claims with custom fields
type jwtClaims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// JWTTokenService implements schema.TokenService using JWT
type JWTTokenService struct {
	cfg config.JWTConfig
}

// NewJWTTokenService creates a new JWTTokenService
func NewJWTTokenService(cfg config.JWTConfig) *JWTTokenService {
	return &JWTTokenService{cfg: cfg}
}

// Generate creates a signed JWT token for the given user
func (s *JWTTokenService) Generate(userID uuid.UUID, email string) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Email:  email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// Validate parses and verifies a JWT token string
func (s *JWTTokenService) Validate(tokenStr string) (*schema.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &schema.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}
