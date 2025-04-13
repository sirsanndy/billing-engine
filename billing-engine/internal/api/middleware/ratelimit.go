package middleware

import (
	"billing-engine/internal/config"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiterMiddleware struct {
	limiters sync.Map
	cfg      config.RateLimitConfig
	logger   *slog.Logger
}

func NewRateLimiterMiddleware(cfg config.RateLimitConfig, logger *slog.Logger) *RateLimiterMiddleware {
	rl := &RateLimiterMiddleware{
		cfg:    cfg,
		logger: logger,
	}

	go rl.cleanupLimiters()

	return rl
}

func (rl *RateLimiterMiddleware) getLimiter(ip string) *rate.Limiter {
	limiter, exists := rl.limiters.Load(ip)
	if !exists {
		newLimiter := rate.NewLimiter(rate.Limit(rl.cfg.RPS), rl.cfg.Burst)
		rl.limiters.Store(ip, newLimiter)
		return newLimiter
	}
	return limiter.(*rate.Limiter)
}

func (rl *RateLimiterMiddleware) cleanupLimiters() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.limiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rate.Limiter)
			if limiter.AllowN(time.Now(), 0) {
				rl.limiters.Delete(key)
			}
			return true
		})
	}
}

func (rl *RateLimiterMiddleware) extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (rl *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {
	if !rl.cfg.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			rl.logger.Warn("Rate limit exceeded", "ip", ip)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": "Rate limit exceeded",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
