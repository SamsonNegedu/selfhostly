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
	// Field names the request field the error is about, so a form can show it next to the input.
	Field string
	Cause error
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
// Common Domain Errors (sentinels used as return values or in errors.Is)
// ============================================================================

var (
	// Compose Errors
	ErrComposeVersionNotFound = &DomainError{
		Code:    "COMPOSE_VERSION_NOT_FOUND",
		Message: "compose version not found",
	}

	// Tunnel Errors
	ErrTunnelNotFound = &DomainError{
		Code:    "TUNNEL_NOT_FOUND",
		Message: "tunnel not found",
	}
	ErrTunnelNotConfigured = &DomainError{
		Code:    "TUNNEL_NOT_CONFIGURED",
		Message: "Cloudflare not configured",
	}
)

// ============================================================================
// Error Wrapping Helpers
// ============================================================================

// Error codes used by Wrap* and Is* (no sentinel vars; only these codes are checked)
const (
	codeAppNotFound              = "APP_NOT_FOUND"
	codeComposeInvalid           = "COMPOSE_INVALID"
	codeTunnelCreationFailed     = "TUNNEL_CREATION_FAILED"
	codeContainerNotFound        = "CONTAINER_NOT_FOUND"
	codeContainerOperationFailed = "CONTAINER_OPERATION_FAILED"
	codeSettingsNotFound         = "SETTINGS_NOT_FOUND"
	codeValidationFailed         = "VALIDATION_FAILED"
	codeRequiredFieldMissing     = "REQUIRED_FIELD_MISSING"
	codeAppNameInvalid           = "APP_NAME_INVALID"
	codeDatabaseOperation        = "DATABASE_OPERATION_FAILED"
	codeNodeNotFound             = "NODE_NOT_FOUND"
	codeConflict                 = "CONFLICT"
)

// Codes for conflicts the client can act on.
const (
	CodeNodeIDTaken      = "NODE_ID_TAKEN"
	CodeNodeNameTaken    = "NODE_NAME_TAKEN"
	CodePrimaryProtected = "PRIMARY_NODE_PROTECTED"
	CodeNodeHasApps      = "NODE_HAS_APPS"
	CodeCurrentNode      = "CURRENT_NODE"
	CodeNodeOnline       = "NODE_ONLINE"
)

// WrapNodeNotFound wraps an error as a node not found error
func WrapNodeNotFound(nodeID string, cause error) error {
	return &DomainError{Code: codeNodeNotFound, Message: fmt.Sprintf("node not found: %s", nodeID), Cause: cause}
}

// WrapConflict reports that the request cannot be done because of the current state, for example a name that is
// already used. The message is written for the person using the app, so it is safe to show.
func WrapConflict(code, field, message string) error {
	return &DomainError{Code: code, Field: field, Message: message}
}

// WrapAppNotFound wraps an error as an app not found error
func WrapAppNotFound(appID string, cause error) error {
	return &DomainError{
		Code:    codeAppNotFound,
		Message: fmt.Sprintf("app not found: %s", appID),
		Cause:   cause,
	}
}

// WrapComposeInvalid wraps an error as an invalid compose error
func WrapComposeInvalid(cause error) error {
	return &DomainError{
		Code:    codeComposeInvalid,
		Message: "invalid compose file format",
		Cause:   cause,
	}
}

// WrapTunnelCreationFailed wraps an error as a tunnel creation failure
func WrapTunnelCreationFailed(appName string, cause error) error {
	return &DomainError{
		Code:    codeTunnelCreationFailed,
		Message: fmt.Sprintf("failed to create tunnel for app: %s", appName),
		Cause:   cause,
	}
}

// WrapContainerOperationFailed wraps an error as a container operation failure
func WrapContainerOperationFailed(operation string, cause error) error {
	return &DomainError{
		Code:    codeContainerOperationFailed,
		Message: fmt.Sprintf("container operation failed: %s", operation),
		Cause:   cause,
	}
}

// WrapDatabaseOperation wraps an error as a database operation failure
func WrapDatabaseOperation(operation string, cause error) error {
	return &DomainError{
		Code:    codeDatabaseOperation,
		Message: fmt.Sprintf("database operation failed: %s", operation),
		Cause:   cause,
	}
}

// WrapValidationError wraps an error as a validation failure
// For validation errors, we include the cause details in the message since they're safe and helpful for users
func WrapValidationError(field string, cause error) error {
	message := fmt.Sprintf("validation failed for %s", field)
	if cause != nil {
		message = fmt.Sprintf("validation failed for %s: %s", field, cause.Error())
	}
	return &DomainError{
		Code:    codeValidationFailed,
		Field:   field,
		Message: message,
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
		return domainErr.Code == codeAppNotFound ||
			domainErr.Code == ErrTunnelNotFound.Code ||
			domainErr.Code == codeContainerNotFound ||
			domainErr.Code == ErrComposeVersionNotFound.Code ||
			domainErr.Code == codeSettingsNotFound ||
			domainErr.Code == codeNodeNotFound
	}
	return false
}

// IsConflictError checks if an error is a conflict with the current state
func IsConflictError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	switch domainErr.Code {
	case CodeNodeIDTaken, CodeNodeNameTaken, CodePrimaryProtected, CodeNodeHasApps, CodeCurrentNode, CodeNodeOnline, codeConflict:
		return true
	}
	return false
}

// ErrorCode returns the machine-readable code of a domain error, or an empty string for any other error.
func ErrorCode(err error) string {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return ""
}

// ErrorField returns the request field a domain error is about, if it names one.
func ErrorField(err error) string {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Field
	}
	return ""
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == codeValidationFailed ||
			domainErr.Code == codeRequiredFieldMissing ||
			domainErr.Code == codeAppNameInvalid ||
			domainErr.Code == codeComposeInvalid
	}
	return false
}

// PublicMessage returns a safe, user-facing message for API responses.
// For DomainError it returns only the Message (never Cause, to avoid leaking DB/driver internals).
// For other errors it returns a generic message.
func PublicMessage(err error) string {
	var de *DomainError
	if errors.As(err, &de) && de.Message != "" {
		return de.Message
	}
	return "An error occurred"
}
