package middleware

import (
	"billing-engine/internal/config"
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	const statusErrorMsg = "expected status %d, got %d"

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	secret := "testsecret"

	cfg := config.AuthConfig{
		Enabled:   true,
		JWTSecret: secret,
	}

	t.Run("should allow request when middleware is disabled", func(t *testing.T) {
		cfg.Enabled = false
		middleware := AuthMiddleware(cfg, logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf(statusErrorMsg, http.StatusOK, rec.Code)
		}
	})

	t.Run("should reject request with missing Authorization header", func(t *testing.T) {
		cfg.Enabled = true
		middleware := AuthMiddleware(cfg, logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf(statusErrorMsg, http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("should reject request with invalid token", func(t *testing.T) {
		middleware := AuthMiddleware(cfg, logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalidtoken")
		rec := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("should allow request with valid token", func(t *testing.T) {
		middleware := AuthMiddleware(cfg, logger)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "1234567890",
		})
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}
