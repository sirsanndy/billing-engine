package apperrors

import (
	"testing"
)

func TestAppErrorError(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "With Code",
			appError: &AppError{
				Code:    "TEST_CODE",
				Message: "This is a test error",
			},
			expected: "[TEST_CODE] This is a test error",
		},
		{
			name: "Without Code",
			appError: &AppError{
				Message: "This is a test error without code",
			},
			expected: "This is a test error without code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.appError.Error()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
