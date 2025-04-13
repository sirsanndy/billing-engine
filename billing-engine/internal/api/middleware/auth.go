package middleware

import (
	"billing-engine/internal/config"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(cfg config.AuthConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !validateJWT(r, cfg.JWTSecret, logger) {
				http.Error(w, `{"error":{"message":"Unauthorized"}}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validateJWT(r *http.Request, secret string, logger *slog.Logger) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logger.Warn("AuthMiddleware: Missing Authorization header")
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		logger.Warn("AuthMiddleware: Invalid Authorization header format")
		return false
	}
	tokenString := parts[1]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Warn("AuthMiddleware: Unexpected signing method")
			return nil, http.ErrAbortHandler
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		logger.Warn("AuthMiddleware: Invalid token", "error", err)
		return false
	}

	logger.Info("AuthMiddleware: Authenticated request", "token", tokenString)
	return true
}
