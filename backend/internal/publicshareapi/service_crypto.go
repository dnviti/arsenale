package publicshareapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func deriveKeyFromToken(token, shareID, saltBase64 string) ([]byte, error) {
	ikm, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	var salt []byte
	if strings.TrimSpace(saltBase64) != "" {
		salt, err = base64.StdEncoding.DecodeString(saltBase64)
		if err != nil {
			return nil, fmt.Errorf("decode token salt: %w", err)
		}
	}

	key := make([]byte, 32)
	reader := hkdf.New(sha256.New, ikm, salt, []byte(shareID))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("derive hkdf key: %w", err)
	}
	return key, nil
}

func deriveKeyFromTokenAndPin(token, pin, saltHex string) ([]byte, error) {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("decode pin salt: %w", err)
	}
	return argon2.IDKey([]byte(token+pin), salt, 3, 64*1024, 1, 32), nil
}

func decryptPayload(cipherHex, ivHex, tagHex string, key []byte) (string, error) {
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", fmt.Errorf("decode iv: %w", err)
	}
	tag, err := hex.DecodeString(tagHex)
	if err != nil {
		return "", fmt.Errorf("decode tag: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, append(ciphertext, tag...), nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func zero(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

func nullableString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableStringValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
