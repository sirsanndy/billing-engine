package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredLogger(t *testing.T) {

	logBuffer := new(bytes.Buffer)

	testHandler := slog.NewJSONHandler(logBuffer, nil)
	testLogger := slog.New(testHandler)

	mockResponseStatus := http.StatusAccepted
	mockResponseBody := "Hello from next handler!"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(mockResponseStatus)
		_, _ = w.Write([]byte(mockResponseBody))
	})

	loggerMiddleware := StructuredLogger(testLogger)

	req := httptest.NewRequest("GET", "/test/path?query=1", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	testReqID := "test-request-id-123"
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, testReqID))

	rr := httptest.NewRecorder()

	handlerToTest := loggerMiddleware(nextHandler)
	handlerToTest.ServeHTTP(rr, req)

	assert.Equal(t, mockResponseStatus, rr.Code, "Next handler should set the status code")
	assert.Equal(t, mockResponseBody, rr.Body.String(), "Next handler should write the body")

	var logEntry map[string]interface{}
	err := json.Unmarshal(logBuffer.Bytes(), &logEntry)
	fmt.Print(string(logBuffer.Bytes()))
	require.NoError(t, err, "Failed to unmarshal log output")

	assert.Equal(t, "INFO", logEntry["level"], "Log level should be INFO")
	assert.Equal(t, "Served request", logEntry["msg"], "Log message mismatch")
	assert.Equal(t, req.Proto, logEntry["proto"], "Logged proto mismatch")
	assert.Equal(t, req.Method, logEntry["method"], "Logged method mismatch")
	assert.Equal(t, req.URL.Path, logEntry["path"], "Logged path mismatch")
	assert.Equal(t, req.RemoteAddr, logEntry["remote_addr"], "Logged remote_addr mismatch")
	assert.Equal(t, req.UserAgent(), logEntry["user_agent"], "Logged user_agent mismatch")

	assert.Equal(t, float64(mockResponseStatus), logEntry["status"], "Logged status mismatch")

	assert.Equal(t, float64(len(mockResponseBody)), logEntry["bytes_written"], "Logged bytes_written mismatch")
	assert.Equal(t, testReqID, logEntry["request_id"], "Logged request_id mismatch")

	latency, ok := logEntry["latency_ms"].(float64)
	assert.True(t, ok, "Latency should be a float64")
	assert.Greater(t, latency, 0.0, "Latency should be greater than 0")

	_, timeOk := logEntry["time"].(string)
	assert.True(t, timeOk, "Timestamp should exist in log entry")
}

func TestStructuredLoggerNoRequestID(t *testing.T) {
	logBuffer := new(bytes.Buffer)
	testHandler := slog.NewJSONHandler(logBuffer, nil)
	testLogger := slog.New(testHandler)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggerMiddleware := StructuredLogger(testLogger)
	req := httptest.NewRequest("POST", "/other", nil)
	rr := httptest.NewRecorder()

	handlerToTest := loggerMiddleware(nextHandler)
	handlerToTest.ServeHTTP(rr, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal(logBuffer.Bytes(), &logEntry)
	require.NoError(t, err, "Failed to unmarshal log output")

	assert.Equal(t, "", logEntry["request_id"], "Logged request_id should be empty string when not set")
	assert.Equal(t, float64(http.StatusOK), logEntry["status"], "Logged status mismatch")
	assert.Equal(t, req.Method, logEntry["method"], "Logged method mismatch")
	assert.Equal(t, req.URL.Path, logEntry["path"], "Logged path mismatch")
}
