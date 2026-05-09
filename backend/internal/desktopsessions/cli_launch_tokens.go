package desktopsessions

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const desktopLaunchSecretBytes = 32

func newOpaqueGrant() (string, string, string, error) {
	id, err := randomURLToken(18)
	if err != nil {
		return "", "", "", err
	}
	secret, err := randomURLToken(desktopLaunchSecretBytes)
	if err != nil {
		return "", "", "", err
	}
	return id, secret, id + "." + secret, nil
}

func splitOpaqueGrant(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("invalid grant")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func hashOpaqueSecret(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

func opaqueSecretMatches(secret, expectedHash string) bool {
	got := hashOpaqueSecret(secret)
	return subtle.ConstantTimeCompare([]byte(got), []byte(strings.TrimSpace(expectedHash))) == 1
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
