package domain

import (
	"errors"
	"fmt"
)

// ============================================================================
// Domain Error Types
// ============================================================================

// DomainError represents a domain-specific error with a code and message
type DomainError struct {
	Code    string
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, cause error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ============================================================================
// Common Domain Errors
// ============================================================================

var (
	// App Errors
	ErrAppNotFound = &DomainError{
		Code:    "APP_NOT_FOUND",
		Message: "app not found",
	}
	ErrAppAlreadyExists = &DomainError{
		Code:    "APP_ALREADY_EXISTS",
		Message: "app with this name already exists",
	}
	ErrAppInvalidState = &DomainError{
		Code:    "APP_INVALID_STATE",
		Message: "invalid app state transition",
	}
	ErrAppNameInvalid = &DomainError{
		Code:    "APP_NAME_INVALID",
		Message: "app name is invalid",
	}

	// Compose Errors
	ErrComposeInvalid = &DomainError{
		Code:    "COMPOSE_INVALID",
		Message: "compose file is invalid",
	}
	ErrComposeVersionNotFound = &DomainError{
		Code:    "COMPOSE_VERSION_NOT_FOUND",
		Message: "compose version not found",
	}

	// Tunnel Errors
	ErrTunnelNotFound = &DomainError{
		Code:    "TUNNEL_NOT_FOUND",
		Message: "tunnel not found",
	}
	ErrTunnelCreationFailed = &DomainError{
		Code:    "TUNNEL_CREATION_FAILED",
		Message: "failed to create tunnel",
	}
	ErrTunnelNotConfigured = &DomainError{
		Code:    "TUNNEL_NOT_CONFIGURED",
		Message: "Cloudflare not configured",
	}

	// Container Errors
	ErrContainerNotFound = &DomainError{
		Code:    "CONTAINER_NOT_FOUND",
		Message: "container not found",
	}
	ErrContainerOperationFailed = &DomainError{
		Code:    "CONTAINER_OPERATION_FAILED",
		Message: "container operation failed",
	}

	// Settings Errors
	ErrSettingsNotFound = &DomainError{
		Code:    "SETTINGS_NOT_FOUND",
		Message: "settings not found",
	}

	// Validation Errors
	ErrValidationFailed = &DomainError{
		Code:    "VALIDATION_FAILED",
		Message: "validation failed",
	}
	ErrRequiredFieldMissing = &DomainError{
		Code:    "REQUIRED_FIELD_MISSING",
		Message: "required field is missing",
	}

	// Infrastructure Errors
	ErrDatabaseOperation = &DomainError{
		Code:    "DATABASE_OPERATION_FAILED",
		Message: "database operation failed",
	}
	ErrFileSystem = &DomainError{
		Code:    "FILESYSTEM_ERROR",
		Message: "filesystem operation failed",
	}
	ErrNetworkOperation = &DomainError{
		Code:    "NETWORK_OPERATION_FAILED",
		Message: "network operation failed",
	}
)

// ============================================================================
// Error Wrapping Helpers
// ============================================================================

// WrapAppNotFound wraps an error as an app not found error
func WrapAppNotFound(appID string, cause error) error {
	return &DomainError{
		Code:    ErrAppNotFound.Code,
		Message: fmt.Sprintf("app not found: %s", appID),
		Cause:   cause,
	}
}

// WrapAppAlreadyExists wraps an error as an app already exists error
func WrapAppAlreadyExists(appName string, cause error) error {
	return &DomainError{
		Code:    ErrAppAlreadyExists.Code,
		Message: fmt.Sprintf("app already exists: %s", appName),
		Cause:   cause,
	}
}

// WrapComposeInvalid wraps an error as an invalid compose error
func WrapComposeInvalid(cause error) error {
	return &DomainError{
		Code:    ErrComposeInvalid.Code,
		Message: "invalid compose file format",
		Cause:   cause,
	}
}

// WrapTunnelCreationFailed wraps an error as a tunnel creation failure
func WrapTunnelCreationFailed(appName string, cause error) error {
	return &DomainError{
		Code:    ErrTunnelCreationFailed.Code,
		Message: fmt.Sprintf("failed to create tunnel for app: %s", appName),
		Cause:   cause,
	}
}

// WrapContainerOperationFailed wraps an error as a container operation failure
func WrapContainerOperationFailed(operation string, cause error) error {
	return &DomainError{
		Code:    ErrContainerOperationFailed.Code,
		Message: fmt.Sprintf("container operation failed: %s", operation),
		Cause:   cause,
	}
}

// WrapDatabaseOperation wraps an error as a database operation failure
func WrapDatabaseOperation(operation string, cause error) error {
	return &DomainError{
		Code:    ErrDatabaseOperation.Code,
		Message: fmt.Sprintf("database operation failed: %s", operation),
		Cause:   cause,
	}
}

// ============================================================================
// Error Checking Helpers
// ============================================================================

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrAppNotFound.Code ||
			domainErr.Code == ErrTunnelNotFound.Code ||
			domainErr.Code == ErrContainerNotFound.Code ||
			domainErr.Code == ErrComposeVersionNotFound.Code ||
			domainErr.Code == ErrSettingsNotFound.Code
	}
	return false
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrValidationFailed.Code ||
			domainErr.Code == ErrRequiredFieldMissing.Code ||
			domainErr.Code == ErrAppNameInvalid.Code ||
			domainErr.Code == ErrComposeInvalid.Code
	}
	return false
}

// IsInfrastructureError checks if an error is an infrastructure error
func IsInfrastructureError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrDatabaseOperation.Code ||
			domainErr.Code == ErrFileSystem.Code ||
			domainErr.Code == ErrNetworkOperation.Code ||
			domainErr.Code == ErrContainerOperationFailed.Code
	}
	return false
}
