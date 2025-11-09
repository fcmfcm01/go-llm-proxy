package models

import "errors"

// Custom errors
var (
	// User errors
	ErrUserIDRequired        = errors.New("user ID is required")
	ErrUsernameRequired      = errors.New("username is required")
	ErrUsernameInvalidLength = errors.New("username must be between 3 and 50 characters")
	ErrPasswordRequired      = errors.New("password is required")
	ErrPasswordHashFailed    = errors.New("failed to hash password")
	ErrPasswordMismatch      = errors.New("password does not match")
	ErrInvalidRole           = errors.New("invalid user role")
	ErrValidationFailed      = errors.New("validation failed")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrUserDisabled          = errors.New("user is disabled")
	ErrUserLocked            = errors.New("user is locked")

	// Session errors
	ErrInvalidSessionDuration = errors.New("session duration must be positive")
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionExpired         = errors.New("session has expired")
	ErrSessionInvalid         = errors.New("session is invalid")
	ErrCSRFTokenMismatch      = errors.New("CSRF token mismatch")
	ErrSessionMismatch        = errors.New("session does not match")

	// Repository errors
	ErrRepositoryNotInitialized = errors.New("repository not initialized")
	ErrDuplicateKey             = errors.New("duplicate key")
	ErrTransactionFailed        = errors.New("transaction failed")

	// Service errors
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrCacheMiss          = errors.New("cache miss")
	ErrCacheWriteFailed   = errors.New("cache write failed")
	ErrCacheDeleteFailed  = errors.New("cache delete failed")

	// Model-specific errors
	ErrInvalidUserID           = errors.New("invalid user ID")
	ErrInvalidUsername         = errors.New("invalid username")
	ErrInvalidOperation        = errors.New("invalid operation type")
	ErrInvalidResult           = errors.New("invalid result")
	ErrInvalidModelName        = errors.New("invalid model name")
	ErrInvalidProviderID       = errors.New("invalid provider ID")
	ErrInvalidProviderName     = errors.New("invalid provider name")
	ErrInvalidProviderURL      = errors.New("invalid provider URL")
	ErrInvalidProviderPriority = errors.New("invalid provider priority")

	// Common errors
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrInternal       = errors.New("internal error")
	ErrNotImplemented = errors.New("not implemented")
	ErrTimeout        = errors.New("timeout")
)

// Error is a custom error type that wraps other errors
type Error struct {
	code    string
	message string
	err     error
}

// NewError creates a new error
func NewError(code, message string) *Error {
	return &Error{
		code:    code,
		message: message,
	}
}

// NewErrorWithCause creates a new error with a cause
func NewErrorWithCause(code, message string, err error) *Error {
	return &Error{
		code:    code,
		message: message,
		err:     err,
	}
}

// Error returns the error message
func (e *Error) Error() string {
	if e.err != nil {
		return e.message + ": " + e.err.Error()
	}
	return e.message
}

// Code returns the error code
func (e *Error) Code() string {
	return e.code
}

// Cause returns the underlying error
func (e *Error) Cause() error {
	return e.err
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.err
}

// Wrap wraps an error
func (e *Error) Wrap(err error) *Error {
	return NewErrorWithCause(e.code, e.message, err)
}
