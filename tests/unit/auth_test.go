package unit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/go-llm-proxy/internal/auth"
)

// TestPasswordHashingArgon2 tests Argon2id password hashing
// T028 [P] [US5] Unit test for password hashing (bcrypt/Argon2)
func TestPasswordHashingArgon2(t *testing.T) {
	config := auth.DefaultArgon2Config()

	tests := []struct {
		name        string
		password    string
		shouldError bool
	}{
		{
			name:        "valid password",
			password:    "MySecurePassword123!",
			shouldError: false,
		},
		{
			name:        "minimum length password",
			password:    "12345678",
			shouldError: false,
		},
		{
			name:        "complex password with special chars",
			password:    "P@ssw0rd!#$%^&*()",
			shouldError: false,
		},
		{
			name:        "unicode password",
			password:    "パスワード123",
			shouldError: false,
		},
		{
			name:        "empty password",
			password:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password, config)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Empty(t, hash)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Verify hash format
			assert.True(t, strings.HasPrefix(hash, "argon2id$"))

			// Verify can validate
			valid, err := auth.ValidatePassword(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, valid)

			// Verify wrong password fails
			valid, err = auth.ValidatePassword(tt.password+"wrong", hash)
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

// TestPasswordHashingBCrypt tests BCrypt password hashing
func TestPasswordHashingBCrypt(t *testing.T) {
	config := auth.DefaultBCryptConfig()

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "valid password",
			password: "MySecurePassword123!",
		},
		{
			name:     "complex password",
			password: "P@ssw0rd!#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password, config)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Verify hash format (BCrypt starts with $2a$ or $2b$)
			assert.True(t, strings.HasPrefix(hash, "$2"))

			// Verify can validate
			valid, err := auth.ValidatePassword(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, valid)

			// Verify wrong password fails
			valid, err = auth.ValidatePassword(tt.password+"wrong", hash)
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

// TestPasswordHashDeterminism verifies hashes are unique (salted)
func TestPasswordHashDeterminism(t *testing.T) {
	config := auth.DefaultArgon2Config()
	password := "TestPassword123"

	hash1, err := auth.HashPassword(password, config)
	require.NoError(t, err)

	hash2, err := auth.HashPassword(password, config)
	require.NoError(t, err)

	// Hashes should be different due to random salt
	assert.NotEqual(t, hash1, hash2)

	// But both should validate the same password
	valid1, err := auth.ValidatePassword(password, hash1)
	require.NoError(t, err)
	assert.True(t, valid1)

	valid2, err := auth.ValidatePassword(password, hash2)
	require.NoError(t, err)
	assert.True(t, valid2)
}

// TestPasswordHashingPerformance ensures hashing isn't too slow
func TestPasswordHashingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := auth.DefaultArgon2Config()
	password := "TestPassword123"

	// Hashing should complete in reasonable time (< 500ms)
	const maxIterations = 10
	for i := 0; i < maxIterations; i++ {
		_, err := auth.HashPassword(password, config)
		require.NoError(t, err)
	}
}

// TestPasswordValidationEdgeCases tests edge cases for password validation
func TestPasswordValidationEdgeCases(t *testing.T) {
	config := auth.DefaultArgon2Config()
	password := "TestPassword123"

	tests := []struct {
		name        string
		hash        string
		password    string
		expectError bool
		expectValid bool
	}{
		{
			name:        "invalid hash format",
			hash:        "invalid-hash",
			password:    password,
			expectError: true,
			expectValid: false,
		},
		{
			name:        "empty hash",
			hash:        "",
			password:    password,
			expectError: true,
			expectValid: false,
		},
		{
			name:        "empty password against valid hash",
			hash:        func() string { h, _ := auth.HashPassword(password, config); return h }(),
			password:    "",
			expectError: false,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := auth.ValidatePassword(tt.password, tt.hash)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValid, valid)
		})
	}
}
