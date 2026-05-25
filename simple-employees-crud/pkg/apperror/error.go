// Package apperror provides domain-level error types that flow from the
// repository/service layers up to the centralized exception middleware.
// Handlers never write error responses directly; they call c.Error(err)
// and the middleware translates *AppError into the correct HTTP response.
package apperror

import "net/http"

// ErrorCode is a machine-readable error identifier returned in the response body.
type ErrorCode string

const (
	ErrCodeNotFound   ErrorCode = "NOT_FOUND"
	ErrCodeBadRequest ErrorCode = "BAD_REQUEST"
	ErrCodeUnauth     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden  ErrorCode = "FORBIDDEN"
	ErrCodeConflict   ErrorCode = "CONFLICT"
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
	ErrCodeInternal   ErrorCode = "INTERNAL_SERVER_ERROR"
)

// AppError is a structured, HTTP-aware application error.
// It implements the standard error interface so it can be used anywhere errors
// are expected while carrying enough context for the middleware to build a
// well-formed JSON error response.
type AppError struct {
	// Code is the machine-readable identifier (used by clients to branch logic).
	Code ErrorCode `json:"code"`
	// Message is a human-readable explanation suitable for display.
	Message string `json:"message"`
	// StatusCode is the HTTP status that should be sent; never serialised to JSON.
	StatusCode int `json:"-"`
	// Details holds optional extra context (e.g. validation field errors).
	Details any `json:"details,omitempty"`
}

// Error satisfies the error interface.
func (e *AppError) Error() string { return e.Message }

// ─── Constructors ────────────────────────────────────────────────────────────

// New creates an AppError with full control over all fields.
func New(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

// NewNotFound returns a 404 error for a named resource.
//
//	apperror.NewNotFound("Employee")  →  "Employee not found"
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    resource + " not found",
		StatusCode: http.StatusNotFound,
	}
}

// NewBadRequest returns a 400 error for malformed or semantically invalid input.
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:       ErrCodeBadRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// NewUnauthorized returns a 401 error (missing or invalid credentials).
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnauth,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewForbidden returns a 403 error (authenticated but insufficient permissions).
func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// NewConflict returns a 409 error for uniqueness/state conflicts.
func NewConflict(message string) *AppError {
	return &AppError{
		Code:       ErrCodeConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// NewValidation returns a 422 error with structured field-level details.
// details should be a map[string]string of {field: reason} pairs.
func NewValidation(details any) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    "request validation failed",
		StatusCode: http.StatusUnprocessableEntity,
		Details:    details,
	}
}

// NewInternal returns a 500 error. The message is safe to expose externally;
// underlying technical details should be logged before creating this error.
func NewInternal(message string) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

// IsNotFound reports whether err is a 404 AppError.
func IsNotFound(err error) bool {
	if e, ok := err.(*AppError); ok {
		return e.Code == ErrCodeNotFound
	}
	return false
}
