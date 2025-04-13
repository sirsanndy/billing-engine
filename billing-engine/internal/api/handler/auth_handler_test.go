package handler

import (
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/config"
	"billing-engine/internal/pkg/apperrors"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) LoadConfig() string {
	args := m.Called()
	return args.String(0)
}

var logger = slog.New(slog.NewTextHandler(io.Discard, nil))

func newTestConfig() config.Config {
	return config.Config{
		Server: config.ServerConfig{
			Auth: config.AuthConfig{
				JWTSecret: "test-jwt-secret-key",
			},
		},
	}
}

func TestGenerateBearerToken(t *testing.T) {
	mockCfg := newTestConfig()

	handler := NewAuthHandler(mockCfg, logger)

	t.Run("successfully generates token", func(t *testing.T) {
		reqBody := dto.TokenRequest{Username: "testuser"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateBearerToken(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]string
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		assert.NoError(t, err)
		assert.Contains(t, respBody["token"], "Bearer ")
	})

	t.Run("fails with invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		handler.GenerateBearerToken(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var respBody dto.ErrorResponse
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		assert.NoError(t, err)
		assert.Contains(t, respBody.Error.Message, apperrors.ErrInvalidArgument.Error())
	})

	t.Run("fails with missing username", func(t *testing.T) {
		reqBody := dto.TokenRequest{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.GenerateBearerToken(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var respBody dto.ErrorResponse
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		assert.NoError(t, err)
		assert.Contains(t, respBody.Error.Message, "username is required")
	})
}
