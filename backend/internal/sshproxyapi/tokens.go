package sshproxyapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

func newProxyGrant() (string, string, string, error) {
	id, err := randomURLToken(18)
	if err != nil {
		return "", "", "", err
	}
	secret, err := randomURLToken(32)
	if err != nil {
		return "", "", "", err
	}
	return id, secret, id + "." + secret, nil
}

func splitProxyGrant(value string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("invalid proxy grant")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
