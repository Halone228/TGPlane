package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// TokenEncryptor encrypts/decrypts strings using AES-256-GCM.
// If no key is configured (empty key), it operates in plaintext passthrough mode.
type TokenEncryptor struct {
	gcm     cipher.AEAD
	enabled bool
}

// NewTokenEncryptor creates a new TokenEncryptor. If hexKey is empty, the
// encryptor works in passthrough mode (no encryption). Otherwise hexKey must
// be a 64-character hex string (32 bytes for AES-256).
func NewTokenEncryptor(hexKey string) (*TokenEncryptor, error) {
	if hexKey == "" {
		return &TokenEncryptor{enabled: false}, nil
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	return &TokenEncryptor{gcm: gcm, enabled: true}, nil
}

// IsEnabled reports whether encryption is active.
func (e *TokenEncryptor) IsEnabled() bool {
	return e != nil && e.enabled
}

// Encrypt encrypts plaintext and returns a base64-encoded string containing
// the nonce prepended to the ciphertext. In passthrough mode it returns the
// plaintext unchanged.
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if !e.IsEnabled() {
		return plaintext, nil
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes a base64 string, extracts the nonce, and decrypts the
// ciphertext. In passthrough mode it returns the input unchanged.
func (e *TokenEncryptor) Decrypt(ciphertext string) (string, error) {
	if !e.IsEnabled() {
		return ciphertext, nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}

	nonce, sealed := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}

	return string(plaintext), nil
}
