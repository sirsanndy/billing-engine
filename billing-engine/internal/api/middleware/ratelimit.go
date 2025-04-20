package middleware

import (
	"billing-engine/internal/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiterMiddleware struct {
	redisClient *redis.Client
	cfg         config.RateLimitConfig
	logger      *slog.Logger
	window      time.Duration
}

func NewRateLimiterMiddleware(
	cfg config.RateLimitConfig,
	redisClient *redis.Client,
	logger *slog.Logger,
) *RateLimiterMiddleware {

	logger.Info("Initializing rate limiter middleware component...")

	if !cfg.Enabled {
		logger.Info("Rate limiting is disabled via configuration.")

	} else if redisClient == nil {
		logger.Warn("Rate limiting enabled but no Redis client provided; disabling.")
		cfg.Enabled = false
	} else {
		logger.Info("Rate limiter middleware configured", "rps", cfg.RPS, "window", 1*time.Second)
	}

	return &RateLimiterMiddleware{
		redisClient: redisClient,
		cfg:         cfg,
		logger:      logger,
		window:      1 * time.Second,
	}
}

func (rl *RateLimiterMiddleware) IsEnabled() bool {

	return rl.cfg.Enabled && rl.redisClient != nil
}

func (rl *RateLimiterMiddleware) GetConfig() config.RateLimitConfig {
	return rl.cfg
}

func (rl *RateLimiterMiddleware) extractIP(r *http.Request) string {

	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		ip := strings.TrimSpace(ips[0])

		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		ip := strings.TrimSpace(xRealIP)

		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}

	parsedIP := net.ParseIP(r.RemoteAddr)
	if parsedIP != nil {
		return parsedIP.String()
	}

	rl.logger.Warn("Could not determine client IP for rate limiting", "remoteAddr", r.RemoteAddr, "x-forwarded-for", xff, "x-real-ip", xRealIP)
	return "unknown"
}

func (rl *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {

	if !rl.IsEnabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractIP(r)
		if ip == "unknown" {

			rl.logger.Error("Blocking request due to unknown client IP for rate limiting")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx := r.Context()
		key := fmt.Sprintf("ratelimit:%s", ip)

		pipe := rl.redisClient.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		ttlCmd := pipe.TTL(ctx, key)

		_, err := pipe.Exec(ctx)
		if err != nil {
			rl.logger.Error("Redis pipeline failed during rate limiting check", "error", err, "ip", ip, "key", key)

			next.ServeHTTP(w, r)
			return
		}

		currentCount, errIncr := incrCmd.Result()
		ttl, errTTL := ttlCmd.Result()

		if errIncr != nil {
			rl.logger.Error("Failed to get INCR result after pipeline exec", "error", errIncr, "ip", ip, "key", key)
			next.ServeHTTP(w, r)
			return
		}
		if errTTL != nil {
			rl.logger.Error("Failed to get TTL result after pipeline exec", "error", errTTL, "ip", ip, "key", key)

		}

		if ttl == -1 || ttl == -2 {

			if err := rl.redisClient.Expire(ctx, key, rl.window).Err(); err != nil {

				rl.logger.Error("Failed to set Redis EXPIRE for rate limit key", "error", err, "ip", ip, "key", key)
			}
		}

		if currentCount > int64(rl.cfg.RPS) {
			rl.logger.Warn("Rate limit exceeded", "ip", ip, "count", currentCount, "limit", rl.cfg.RPS)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rl.window.Seconds()))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": fmt.Sprintf("Rate limit exceeded. Limit is %d requests per %v.", rl.cfg.RPS, rl.window),
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
