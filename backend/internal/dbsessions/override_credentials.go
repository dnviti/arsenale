package dbsessions

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	overridePasswordCiphertextKey = "overridePasswordCiphertext"
	overridePasswordIVKey         = "overridePasswordIV"
	overridePasswordTagKey        = "overridePasswordTag"
)

type encryptedOverrideField struct {
	Ciphertext string
	IV         string
	Tag        string
}

func storeOverridePasswordMetadata(metadata map[string]any, password string, serverKey []byte) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("override password is required")
	}
	field, err := encryptOverrideValue(serverKey, password)
	if err != nil {
		return err
	}
	metadata[overridePasswordCiphertextKey] = field.Ciphertext
	metadata[overridePasswordIVKey] = field.IV
	metadata[overridePasswordTagKey] = field.Tag
	return nil
}

func resolveOverrideCredentials(metadata map[string]any, serverKey []byte) (string, string, error) {
	username := strings.TrimSpace(stringValue(metadata["username"]))
	if username == "" {
		return "", "", fmt.Errorf("override username is unavailable")
	}
	password, err := loadOverridePassword(metadata, serverKey)
	if err != nil {
		return "", "", err
	}
	return username, password, nil
}

func loadOverridePassword(metadata map[string]any, serverKey []byte) (string, error) {
	field := encryptedOverrideField{
		Ciphertext: strings.TrimSpace(stringValue(metadata[overridePasswordCiphertextKey])),
		IV:         strings.TrimSpace(stringValue(metadata[overridePasswordIVKey])),
		Tag:        strings.TrimSpace(stringValue(metadata[overridePasswordTagKey])),
	}
	if field.Ciphertext == "" || field.IV == "" || field.Tag == "" {
		return "", fmt.Errorf("override password is unavailable")
	}
	return decryptOverrideValue(serverKey, field)
}

func encryptOverrideValue(key []byte, plaintext string) (encryptedOverrideField, error) {
	if len(key) != 32 {
		return encryptedOverrideField{}, fmt.Errorf("server encryption key is unavailable")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedOverrideField{}, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return encryptedOverrideField{}, fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return encryptedOverrideField{}, fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	tagOffset := len(sealed) - gcm.Overhead()
	return encryptedOverrideField{
		Ciphertext: hex.EncodeToString(sealed[:tagOffset]),
		IV:         hex.EncodeToString(nonce),
		Tag:        hex.EncodeToString(sealed[tagOffset:]),
	}, nil
}

func decryptOverrideValue(key []byte, field encryptedOverrideField) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("server encryption key is unavailable")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	nonce, err := hex.DecodeString(field.IV)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	ciphertext, err := hex.DecodeString(field.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	tag, err := hex.DecodeString(field.Tag)
	if err != nil {
		return "", fmt.Errorf("decode tag: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("decrypt override credentials: %w", err)
	}
	return string(plaintext), nil
}
