package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name               string
		field              string
		cause              error
		expectedPublicMsg  string
		shouldContainInMsg []string
	}{
		{
			name:              "with cause error",
			field:             "compose content",
			cause:             errors.New("invalid compose file: services section is required"),
			expectedPublicMsg: "validation failed for compose content: invalid compose file: services section is required",
			shouldContainInMsg: []string{
				"validation failed for compose content",
				"invalid compose file",
				"services section is required",
			},
		},
		{
			name:              "with nil cause",
			field:             "app name",
			cause:             nil,
			expectedPublicMsg: "validation failed for app name",
			shouldContainInMsg: []string{
				"validation failed for app name",
			},
		},
		{
			name:              "with network validation error",
			field:             "compose content",
			cause:             errors.New("service \"web\" refers to undefined network \"mynet\": network must be defined in the networks section"),
			expectedPublicMsg: "validation failed for compose content: service \"web\" refers to undefined network \"mynet\": network must be defined in the networks section",
			shouldContainInMsg: []string{
				"validation failed for compose content",
				"service \"web\"",
				"undefined network \"mynet\"",
				"network must be defined in the networks section",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapValidationError(tt.field, tt.cause)

			// Check error is not nil
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			// Check it's a DomainError
			var domainErr *DomainError
			if !errors.As(err, &domainErr) {
				t.Error("expected error to be a DomainError")
			}

			// Check that Error() contains the code
			errMsg := err.Error()
			if !strings.Contains(errMsg, "VALIDATION_FAILED") {
				t.Errorf("expected error message to contain code VALIDATION_FAILED, but got: %q", errMsg)
			}

			// Check PublicMessage returns the helpful message (without code)
			publicMsg := PublicMessage(err)
			if publicMsg != tt.expectedPublicMsg {
				t.Errorf("expected public message:\n  %q\nbut got:\n  %q", tt.expectedPublicMsg, publicMsg)
			}

			// Check that all expected substrings are present in public message
			for _, substr := range tt.shouldContainInMsg {
				if !strings.Contains(publicMsg, substr) {
					t.Errorf("expected public message to contain %q, but got: %q", substr, publicMsg)
				}
			}
		})
	}
}

func TestPublicMessage(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "domain error with message",
			err:         &DomainError{Code: "TEST_ERROR", Message: "test message"},
			expectedMsg: "test message",
		},
		{
			name:        "validation error with details",
			err:         WrapValidationError("compose content", errors.New("compose file content cannot be empty")),
			expectedMsg: "validation failed for compose content: compose file content cannot be empty",
		},
		{
			name:        "non-domain error",
			err:         errors.New("some random error"),
			expectedMsg: "An error occurred",
		},
		{
			name:        "nil error returns generic message",
			err:         nil,
			expectedMsg: "An error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := PublicMessage(tt.err)
			if msg != tt.expectedMsg {
				t.Errorf("expected message %q, but got %q", tt.expectedMsg, msg)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error from WrapValidationError",
			err:      WrapValidationError("test field", errors.New("test error")),
			expected: true,
		},
		{
			name:     "compose invalid error",
			err:      WrapComposeInvalid(errors.New("test error")),
			expected: true,
		},
		{
			name:     "not found error",
			err:      WrapAppNotFound("test-app", errors.New("test error")),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "random error",
			err:      errors.New("random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}
