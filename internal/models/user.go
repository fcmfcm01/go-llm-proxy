package models

import (
	"strings"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/crypto"
	"github.com/go-playground/validator/v10"
)

// UserRole represents the role of a user
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleUser   UserRole = "user"
	RoleViewer UserRole = "viewer"
)

// User represents a system user
type User struct {
	ID           string     `json:"id" validate:"required"`
	Username     string     `json:"username" validate:"required,alphanum,min=3,max=50"`
	PasswordHash string     `json:"password_hash" validate:"required,min=60,max=100"`
	Role         UserRole   `json:"role" validate:"required,userRole"`
	CreatedAt    time.Time  `json:"created_at" validate:"required"`
	UpdatedAt    time.Time  `json:"updated_at" validate:"required"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// NewUser creates a new user
func NewUser(username, passwordHash, role string) (*User, error) {
	// Validate role
	if !IsValidRole(UserRole(role)) {
		return nil, ErrInvalidRole
	}

	now := time.Now()
	user := &User{
		ID:           generateID(),
		Username:     strings.ToLower(username),
		PasswordHash: passwordHash,
		Role:         UserRole(role),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate validates the user
func (u *User) Validate() error {
	validate := validator.New()

	// Custom validation for user role
	validate.RegisterValidation("userRole", validateUserRole)

	err := validate.Struct(u)
	if err != nil {
		return NewErrorWithCause("VALIDATION_FAILED", ErrValidationFailed.Error(), err)
	}

	return nil
}

// SetPassword sets a new password hash
func (u *User) SetPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	// Hash password using default Argon2id config
	hashedPassword, err := crypto.HashPassword(password, crypto.DefaultArgon2Config())
	if err != nil {
		return NewErrorWithCause("PASSWORD_HASH_FAILED", ErrPasswordHashFailed.Error(), err)
	}

	u.PasswordHash = hashedPassword
	u.UpdatedAt = time.Now()

	return nil
}

// CheckPassword checks if the password matches
func (u *User) CheckPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	if err := crypto.CheckPassword(password, u.PasswordHash); err != nil {
		return ErrPasswordMismatch
	}

	return nil
}

// UpdateUsername updates the username
func (u *User) UpdateUsername(newUsername string) error {
	if newUsername == "" {
		return ErrUsernameRequired
	}

	if len(newUsername) < 3 || len(newUsername) > 50 {
		return ErrUsernameInvalidLength
	}

	u.Username = strings.ToLower(newUsername)
	u.UpdatedAt = time.Now()

	return nil
}

// UpdateRole updates the user role
func (u *User) UpdateRole(newRole UserRole) error {
	if !IsValidRole(newRole) {
		return ErrInvalidRole
	}

	u.Role = newRole
	u.UpdatedAt = time.Now()

	return nil
}

// MarkLogin marks the user as logged in
func (u *User) MarkLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}

// ToJSON returns a JSON-safe version of the user
func (u *User) ToJSON() *User {
	return &User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: "", // Don't expose password hash
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		LastLoginAt:  u.LastLoginAt,
	}
}

// IsAdmin returns true if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsUser returns true if the user is a regular user
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

// CanAccess returns true if the user can access resources with the given role
func (u *User) CanAccess(requiredRole UserRole) bool {
	roleHierarchy := map[UserRole]int{
		RoleAdmin:  3,
		RoleUser:   2,
		RoleViewer: 1,
	}

	userLevel := roleHierarchy[u.Role]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// SetID implements the Entity interface
func (u *User) SetID(id string) {
	u.ID = id
}

// SetCreatedAt implements the Entity interface
func (u *User) SetCreatedAt(t time.Time) {
	u.CreatedAt = t
}

// SetUpdatedAt implements the Entity interface
func (u *User) SetUpdatedAt(t time.Time) {
	u.UpdatedAt = t
}

// IsValidRole checks if a role is valid
func IsValidRole(role UserRole) bool {
	switch role {
	case RoleAdmin, RoleUser, RoleViewer:
		return true
	default:
		return false
	}
}

// validateUserRole is a custom validator for UserRole
func validateUserRole(fl validator.FieldLevel) bool {
	role := UserRole(fl.Field().String())
	return IsValidRole(role)
}
