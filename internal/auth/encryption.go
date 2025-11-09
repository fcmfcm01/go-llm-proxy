package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionService provides encryption/decryption functionality
type EncryptionService struct {
	key   []byte
	block cipher.Block
}

// NewEncryptionService creates a new encryption service with a given key
func NewEncryptionService(key string) (*EncryptionService, error) {
	// Derive key from passphrase using SHA-256
	hash := sha256.Sum256([]byte(key))

	// Create AES cipher
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &EncryptionService{
		key:   hash[:],
		block: block,
	}, nil
}

// Encrypt encrypts plain text and returns base64-encoded cipher text
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	// Create cipher mode
	stream := cipher.NewCFBEncrypter(e.block, iv)

	// Encrypt
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, []byte(plaintext))

	// Combine IV and ciphertext
	result := append(iv, ciphertext...)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt decrypts base64-encoded cipher text
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode: %w", err)
	}

	if len(data) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract IV and ciphertext
	iv := data[:aes.BlockSize]
	ciphertextBytes := data[aes.BlockSize:]

	// Create cipher mode
	stream := cipher.NewCFBDecrypter(e.block, iv)

	// Decrypt
	plaintext := make([]byte, len(ciphertextBytes))
	stream.XORKeyStream(plaintext, ciphertextBytes)

	return string(plaintext), nil
}

// EncryptProviderAPIKey encrypts a provider API key for storage
func (e *EncryptionService) EncryptProviderAPIKey(apiKey string) (string, error) {
	return e.Encrypt(apiKey)
}

// DecryptProviderAPIKey decrypts a stored provider API key
func (e *EncryptionService) DecryptProviderAPIKey(encryptedKey string) (string, error) {
	return e.Decrypt(encryptedKey)
}

// Simple XOR encryption for non-sensitive data (faster than AES)
type XorEncryption struct {
	key []byte
}

// NewXorEncryption creates a new XOR encryption service
func NewXorEncryption(key string) *XorEncryption {
	return &XorEncryption{
		key: []byte(key),
	}
}

// Encrypt encrypts data using XOR (suitable for non-sensitive data)
func (x *XorEncryption) Encrypt(data []byte) []byte {
	encrypted := make([]byte, len(data))
	for i := range data {
		encrypted[i] = data[i] ^ x.key[i%len(x.key)]
	}
	return encrypted
}

// Decrypt decrypts XOR-encrypted data
func (x *XorEncryption) Decrypt(data []byte) []byte {
	return x.Encrypt(data) // XOR is symmetric
}
