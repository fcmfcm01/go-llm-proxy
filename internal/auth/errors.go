package auth

import "errors"

// Error definitions for authentication
var (
	// ErrInvalidCredentials is returned when username or password is incorrect
	ErrInvalidCredentials = errors.New("invalid username or password")

	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrSessionNotFound is returned when session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when session has expired
	ErrSessionExpired = errors.New("session has expired")

	// ErrSessionInvalid is returned when session is invalid
	ErrSessionInvalid = errors.New("session is invalid")

	// ErrUnauthorized is returned when authentication is required
	ErrUnauthorized = errors.New("unauthorized")

	// ErrPasswordTooShort is returned when password is too short
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// ErrPasswordNoDigit is returned when password has no digit
	ErrPasswordNoDigit = errors.New("password must contain at least one digit")

	// ErrPasswordNoLetter is returned when password has no letter
	ErrPasswordNoLetter = errors.New("password must contain at least one letter")

	// ErrPasswordInvalidChars is returned when password contains invalid characters
	ErrPasswordInvalidChars = errors.New("password contains invalid characters")

	// ErrPasswordMismatch is returned when password doesn't match hash
	ErrPasswordMismatch = errors.New("password does not match")

	// ErrInvalidHashFormat is returned when hash format is invalid
	ErrInvalidHashFormat = errors.New("invalid hash format")

	// ErrEmptyPassword is returned when password is empty
	ErrEmptyPassword = errors.New("password cannot be empty")

	// ErrEmptyHash is returned when hash is empty
	ErrEmptyHash = errors.New("hash cannot be empty")
)
