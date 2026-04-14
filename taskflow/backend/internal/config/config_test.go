package config

import (
	"os"
	"testing"
	"time"
)

func setEnv(key, value string) func() {
	old := os.Getenv(key)
	os.Setenv(key, value)

	return func() {
		os.Setenv(key, old)
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	db := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable"

	if db.DSN() != expected {
		t.Errorf("expected %s, got %s", expected, db.DSN())
	}
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("valid int", func(t *testing.T) {
		cleanup := setEnv("TEST_INT", "10")
		defer cleanup()

		val := getEnvAsInt("TEST_INT", 5)
		if val != 10 {
			t.Errorf("expected 10, got %d", val)
		}
	})

	t.Run("invalid int", func(t *testing.T) {
		cleanup := setEnv("TEST_INT", "abc")
		defer cleanup()

		val := getEnvAsInt("TEST_INT", 5)
		if val != 5 {
			t.Errorf("expected default 5, got %d", val)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		os.Unsetenv("TEST_INT")

		val := getEnvAsInt("TEST_INT", 7)
		if val != 7 {
			t.Errorf("expected default 7, got %d", val)
		}
	})
}

func TestLoad_Success(t *testing.T) {
	defer setEnv("JWT_SECRET", "mysecret")()
	defer setEnv("JWT_EXPIRY_HOURS", "48")()
	defer setEnv("SHUTDOWN_TIMEOUT_SECONDS", "60")()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.JWT.Secret != "mysecret" {
		t.Errorf("expected JWT secret 'mysecret', got %s", cfg.JWT.Secret)
	}

	if cfg.JWT.ExpiresIn != 48*time.Hour {
		t.Errorf("expected 48h expiry, got %v", cfg.JWT.ExpiresIn)
	}

	if cfg.Server.ShutdownTimeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", cfg.Server.ShutdownTimeout)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
}

func TestLoad_InvalidJWTExpiry(t *testing.T) {
	defer setEnv("JWT_SECRET", "secret")()
	defer setEnv("JWT_EXPIRY_HOURS", "invalid")()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JWT_EXPIRY_HOURS")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	defer setEnv("JWT_SECRET", "secret")()

	os.Unsetenv("DB_HOST")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default host 'localhost', got %s", cfg.Database.Host)
	}
}