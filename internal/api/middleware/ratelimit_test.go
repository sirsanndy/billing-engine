package middleware

import (
	"billing-engine/internal/config"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfg := config.RateLimitConfig{
		Enabled: true,
		RPS:     1,
		Burst:   2,
	}

	middleware := NewRateLimiterMiddleware(cfg, logger)

	t.Run("allows requests under the rate limit", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Middleware(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("blocks requests exceeding the rate limit", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Middleware(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		// First request should pass
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req)
		if rec1.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec1.Code)
		}

		// Third request should be blocked
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req)
		if rec2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
		}

		var response map[string]interface{}
		err := json.NewDecoder(rec2.Body).Decode(&response)
		if err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["error"].(map[string]interface{})["message"] != "Rate limit exceeded" {
			t.Errorf("unexpected error message: %v", response)
		}
	})

	t.Run("extractIP handles various headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
		ip := middleware.extractIP(req)
		if ip != "192.168.1.1" {
			t.Errorf("expected IP %s, got %s", "192.168.1.1", ip)
		}

		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		ip = middleware.extractIP(req)
		if ip != "10.0.0.1" {
			t.Errorf("expected IP %s, got %s", "10.0.0.1", ip)
		}

		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		ip = middleware.extractIP(req)
		if ip != "127.0.0.1" {
			t.Errorf("expected IP %s, got %s", "127.0.0.1", ip)
		}
	})
}

// func TestCleanupLimiters(t *testing.T) {
// 	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
// 	cfg := config.RateLimitConfig{
// 		Enabled: true,
// 		RPS:     1,
// 		Burst:   1,
// 	}

// 	middleware := NewRateLimiterMiddleware(cfg, logger)

// 	ip := "127.0.0.1"
// 	limiter := middleware.getLimiter(ip)

// 	limiter.Allow()

// 	time.Sleep(8 * time.Second)

// 	_, exists := middleware.limiters.Load(ip)
// 	if exists {
// 		t.Errorf("expected limiter for IP %s to be cleaned up", ip)
// 	}
// }
