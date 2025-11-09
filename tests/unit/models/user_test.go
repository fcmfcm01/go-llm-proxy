package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	// Test creating a new user
	username := "testuser"
	passwordHash := "hashed_password"

	user, err := NewUser(username, passwordHash, "admin")

	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.Equal(t, "admin", user.Role)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.True(t, user.CreatedAt.Equal(user.UpdatedAt) || user.UpdatedAt.After(user.CreatedAt))
}

func TestUser_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "valid user",
			user: User{
				Username:     "testuser",
				PasswordHash: "hash",
				Role:         RoleAdmin,
			},
			wantErr: false,
		},
		{
			name: "empty username",
			user: User{
				Username:     "",
				PasswordHash: "hash",
				Role:         RoleAdmin,
			},
			wantErr: true,
		},
		{
			name: "username too short",
			user: User{
				Username:     "ab",
				PasswordHash: "hash",
				Role:         RoleAdmin,
			},
			wantErr: true,
		},
		{
			name: "empty password hash",
			user: User{
				Username:     "testuser",
				PasswordHash: "",
				Role:         RoleAdmin,
			},
			wantErr: true,
		},
		{
			name: "empty role",
			user: User{
				Username:     "testuser",
				PasswordHash: "hash",
				Role:         "",
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			user: User{
				Username:     "testuser",
				PasswordHash: "hash",
				Role:         "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUser_SetPassword(t *testing.T) {
	t.Parallel()

	user := &User{
		ID:           "test-id",
		Username:     "testuser",
		PasswordHash: "old_hash",
		Role:         RoleAdmin,
		CreatedAt:    time.Now(),
	}

	newPassword := "new_password"
	err := user.SetPassword(newPassword)

	require.NoError(t, err)
	assert.NotEqual(t, "old_hash", user.PasswordHash)
	assert.NotEqual(t, newPassword, user.PasswordHash) // Should be hashed
	assert.True(t, user.UpdatedAt.After(user.CreatedAt))
}

func TestUser_CheckPassword(t *testing.T) {
	t.Parallel()

	username := "testuser"
	password := "test_password"
	initialTime := time.Now().Add(-time.Hour)

	user := User{
		ID:           "test-id",
		Username:     username,
		PasswordHash: "", // Will be set by SetPassword
		Role:         RoleAdmin,
		CreatedAt:    initialTime,
		UpdatedAt:    initialTime,
	}

	// Set password
	err := user.SetPassword(password)
	require.NoError(t, err)

	// Test correct password
	err = user.CheckPassword(password)
	assert.NoError(t, err)

	// Test wrong password
	err = user.CheckPassword("wrong_password")
	assert.Error(t, err)

	// Test empty password
	err = user.CheckPassword("")
	assert.Error(t, err)
}

func TestUser_UpdateUsername(t *testing.T) {
	t.Parallel()

	user := &User{
		ID:        "test-id",
		Username:  "olduser",
		Role:      RoleAdmin,
		CreatedAt: time.Now(),
	}

	newUsername := "newuser"
	err := user.UpdateUsername(newUsername)

	require.NoError(t, err)
	assert.Equal(t, newUsername, user.Username)
	assert.True(t, user.UpdatedAt.After(user.CreatedAt))
}

func TestUser_UpdateRole(t *testing.T) {
	t.Parallel()

	user := &User{
		ID:        "test-id",
		Username:  "testuser",
		Role:      RoleUser,
		CreatedAt: time.Now(),
	}

	err := user.UpdateRole(RoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.True(t, user.UpdatedAt.After(user.CreatedAt))
}

func TestUser_MarkLogin(t *testing.T) {
	t.Parallel()

	initialTime := time.Now().Add(-time.Hour)
	user := &User{
		ID:        "test-id",
		Username:  "testuser",
		Role:      RoleUser,
		CreatedAt: initialTime,
	}

	user.MarkLogin()

	assert.NotNil(t, user.LastLoginAt)
	assert.True(t, user.LastLoginAt.After(initialTime))
}

func TestUser_ToJSON(t *testing.T) {
	t.Parallel()

	user := User{
		ID:           "test-id",
		Username:     "testuser",
		PasswordHash: "secret_hash",
		Role:         RoleAdmin,
		CreatedAt:    time.Now(),
	}

	jsonUser := user.ToJSON()

	assert.Equal(t, user.ID, jsonUser.ID)
	assert.Equal(t, user.Username, jsonUser.Username)
	assert.Equal(t, user.Role, jsonUser.Role)
	assert.Equal(t, user.CreatedAt, jsonUser.CreatedAt)
	assert.Equal(t, user.UpdatedAt, jsonUser.UpdatedAt)
	assert.Equal(t, user.LastLoginAt, jsonUser.LastLoginAt)
	assert.Empty(t, jsonUser.PasswordHash) // Should not be exposed
}

func TestIsValidRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role  string
		valid bool
	}{
		{RoleAdmin, true},
		{RoleUser, true},
		{RoleViewer, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := IsValidRole(tt.role)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestUser_SetID(t *testing.T) {
	t.Parallel()

	user := &User{
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         RoleAdmin,
	}

	newID := "new-id"
	user.SetID(newID)

	assert.Equal(t, newID, user.ID)
}

func TestUser_SetCreatedAt(t *testing.T) {
	t.Parallel()

	user := &User{
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         RoleAdmin,
	}

	newTime := time.Now()
	user.SetCreatedAt(newTime)

	assert.Equal(t, newTime, user.CreatedAt)
}

func TestUser_SetUpdatedAt(t *testing.T) {
	t.Parallel()

	user := &User{
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         RoleAdmin,
	}

	newTime := time.Now()
	user.SetUpdatedAt(newTime)

	assert.Equal(t, newTime, user.UpdatedAt)
}
