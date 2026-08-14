package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const secretVersion = "v1"

func EncryptSecret(plaintext, keyMaterial string) (string, error) {
	if plaintext == "" {
		return "", errors.New("secret is required")
	}
	if len(keyMaterial) < 32 {
		return "", errors.New("ROUTER_CREDENTIAL_KEY must contain at least 32 characters")
	}
	aead, err := newGCM(keyMaterial)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate encryption nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return secretVersion + ":" + base64.RawStdEncoding.EncodeToString(payload), nil
}

func DecryptSecret(encoded, keyMaterial string) (string, error) {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 || parts[0] != secretVersion {
		return "", errors.New("unsupported encrypted secret format")
	}
	if len(keyMaterial) < 32 {
		return "", errors.New("ROUTER_CREDENTIAL_KEY must contain at least 32 characters")
	}
	aead, err := newGCM(keyMaterial)
	if err != nil {
		return "", err
	}
	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(payload) < aead.NonceSize() {
		return "", errors.New("invalid encrypted secret")
	}
	plaintext, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("unable to decrypt secret")
	}
	return string(plaintext), nil
}

func newGCM(keyMaterial string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
