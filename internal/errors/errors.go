package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application-level error with HTTP context
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error constructors
func NewInvalidRequest(message string, err error) *AppError {
	return &AppError{
		Code:       "INVALID_REQUEST",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Err:        err,
	}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

func NewConflict(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

func NewRateLimited(message string) *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewServiceUnavailable(message string) *AppError {
	return &AppError{
		Code:       "SERVICE_UNAVAILABLE",
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

func NewUpstreamError(message string, err error) *AppError {
	return &AppError{
		Code:       "UPSTREAM_ERROR",
		Message:    message,
		HTTPStatus: http.StatusBadGateway,
		Err:        err,
	}
}
