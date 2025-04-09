package handler

import (
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/config"
	"billing-engine/internal/pkg/apperrors"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	cfg    config.Config
	logger *slog.Logger
}

func NewAuthHandler(cfg config.Config, l *slog.Logger) *AuthHandler {
	return &AuthHandler{
		cfg:    cfg,
		logger: l.With("component", "AuthHandler"),
	}
}

// GenerateBearerToken generates a JWT bearer token using the provided secret.
//
// @Summary Generate a JWT bearer token
// @Description This function generates a JWT bearer token based on a given secret.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.TokenRequest true "username"
// @Success 200 {object} map[string]string "Token successfully generated"
// @Failure 400 {object} dto.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /auth/token [post]
func (h *AuthHandler) GenerateBearerToken(w http.ResponseWriter, r *http.Request) {
	var req dto.TokenRequest
	h.logger.Info("Generating bearer token")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	if req.Username == "" {
		h.logger.Error("username is required")
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, "username is required"))
		return
	}
	claims := jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	h.logger.Info("Creating token", token)
	h.logger.Info("Checking token signing method", token.Method)
	tokenString, _ := token.SignedString([]byte(h.cfg.Server.Auth.JWTSecret))
	respondJSON(w, http.StatusOK, map[string]string{"token": fmt.Sprintf("Bearer %s", tokenString)})
}
