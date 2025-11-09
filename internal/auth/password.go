package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Config holds password hashing configuration
type Config struct {
	Algorithm   string
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	BCryptCost  int
}

// DefaultArgon2Config returns default Argon2id configuration
func DefaultArgon2Config() *Config {
	return &Config{
		Algorithm:   "argon2id",
		Memory:      65536, // 64 MB
		Iterations:  3,
		Parallelism: 1,
	}
}

// DefaultBCryptConfig returns default BCrypt configuration
func DefaultBCryptConfig() *Config {
	return &Config{
		Algorithm:  "bcrypt",
		BCryptCost: 12,
	}
}

// HashPassword hashes a password using the specified algorithm
func HashPassword(password string, config *Config) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	if config == nil {
		config = DefaultArgon2Config()
	}

	switch config.Algorithm {
	case "argon2id":
		return hashPasswordArgon2id(password, config)
	case "bcrypt":
		return hashPasswordBCrypt(password, config)
	default:
		return "", fmt.Errorf("unsupported password algorithm: %s", config.Algorithm)
	}
}

// hashPasswordArgon2id hashes a password using Argon2id
func hashPasswordArgon2id(password string, config *Config) (string, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, 32)

	// Encode as hex
	hashHex := hex.EncodeToString(hash)
	saltHex := hex.EncodeToString(salt)

	// Return formatted hash
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		config.Memory,
		config.Iterations,
		config.Parallelism,
		saltHex,
		hashHex), nil
}

// hashPasswordBCrypt hashes a password using BCrypt
func hashPasswordBCrypt(password string, config *Config) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), config.BCryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword checks if a password matches the hash
func CheckPassword(password, hash string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if hash == "" {
		return fmt.Errorf("hash cannot be empty")
	}

	// Detect algorithm and verify
	if isArgon2Hash(hash) {
		return checkPasswordArgon2id(password, hash)
	}

	// Try BCrypt
	return checkPasswordBCrypt(password, hash)
}

// isArgon2Hash checks if a hash is an Argon2 hash
func isArgon2Hash(hash string) bool {
	return len(hash) > 10 && hash[:10] == "argon2id$v"
}

// checkPasswordArgon2id checks an Argon2id password
func checkPasswordArgon2id(password, hash string) error {
	// Parse hash format: argon2id$v=19$m=65536,t=3,p=1$salt$hash
	var memory, iterations uint32
	var parallelism uint8
	var saltHex, hashHex string

	_, err := fmt.Sscanf(hash, "argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		&memory, &iterations, &parallelism, &saltHex, &hashHex)
	if err != nil {
		return fmt.Errorf("invalid argon2id hash format: %w", err)
	}

	// Decode salt and hash
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return fmt.Errorf("invalid salt: %w", err)
	}

	originalHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("invalid hash: %w", err)
	}

	// Recompute hash
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, 32)

	// Compare hashes
	if !constantTimeEqual(originalHash, computedHash) {
		return fmt.Errorf("password does not match")
	}

	return nil
}

// checkPasswordBCrypt checks a BCrypt password
func checkPasswordBCrypt(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("password does not match")
	}
	return nil
}

// constantTimeEqual compares two byte slices in constant time
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// NeedsRehash checks if a password needs to be rehashed with newer parameters
func NeedsRehash(hash string, config *Config) bool {
	if config == nil {
		config = DefaultArgon2Config()
	}

	if isArgon2Hash(hash) {
		// For Argon2, check if parameters need updating
		var memory, iterations uint32
		var parallelism uint8
		_, err := fmt.Sscanf(hash, "argon2id$v=19$m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
		if err != nil {
			return true // Invalid format, rehash
		}

		return memory < config.Memory || iterations < config.Iterations || parallelism < config.Parallelism
	}

	// For BCrypt, we could check the cost factor
	return false
}

// GeneratePassword generates a random password
func GeneratePassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// Use hex encoding (safe for all systems)
	return hex.EncodeToString(bytes), nil
}

// ValidatePasswordStrength validates a password against certain criteria
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Check for at least one digit
	hasDigit := false
	hasLetter := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= '0' && char <= '9':
			hasDigit = true
		case (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'):
			hasLetter = true
		case char < 32 || char > 126:
			return fmt.Errorf("password contains invalid characters")
		default:
			hasSpecial = true
		}
	}

	// At least one digit and one letter required
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	if !hasLetter {
		return fmt.Errorf("password must contain at least one letter")
	}

	return nil
}

// ValidatePassword validates a password against a hash (wrapper for CheckPassword)
// Returns (valid bool, error) for compatibility with tests
func ValidatePassword(password, hash string) (bool, error) {
	err := CheckPassword(password, hash)
	if err != nil {
		if err.Error() == "password does not match" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
