package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/denisbrodbeck/machineid"
	"golang.org/x/crypto/pbkdf2"
)

type Keystore struct {
	encryptionKey []byte
	salt          []byte
}

const (
	saltFile = "tokentrail.salt"
	keyDerivationIterations = 100000
)

// NewKeystore creates a new keystore with machine-bound encryption key
func NewKeystore(dataDir string) (*Keystore, error) {
	// Get or create salt
	salt, err := getOrCreateSalt(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create salt: %w", err)
	}

	// Get machine ID
	id, err := machineid.ID()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine ID: %w", err)
	}

	// Derive encryption key using PBKDF2
	// Input: machine ID + app-specific string, salt, iterations, SHA256
	password := id + "tokentrail-v1"
	encryptionKey := pbkdf2.Key([]byte(password), salt, keyDerivationIterations, 32, sha256.New)

	return &Keystore{
		encryptionKey: encryptionKey,
		salt:          salt,
	}, nil
}

// Encrypt encrypts a plaintext string using AES-256-GCM
func (k *Keystore) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(k.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to create nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext string using AES-256-GCM
func (k *Keystore) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(k.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextOnly := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertextOnly, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// getOrCreateSalt gets the salt from disk or creates it if it doesn't exist
func getOrCreateSalt(dataDir string) ([]byte, error) {
	saltPath := filepath.Join(dataDir, saltFile)

	// Try to read existing salt
	if salt, err := os.ReadFile(saltPath); err == nil {
		return salt, nil
	}

	// Create new salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Write salt to file with restricted permissions
	if err := os.WriteFile(saltPath, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write salt: %w", err)
	}

	return salt, nil
}
