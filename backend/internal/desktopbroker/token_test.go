package desktopbroker

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"testing"

	"golang.org/x/crypto/scrypt"
)

func TestDecryptToken(t *testing.T) {
	t.Parallel()

	token := ConnectionToken{}
	token.Connection.Type = "rdp"
	token.Connection.GuacdHost = "desktop-proxy"
	token.Connection.GuacdPort = 4822
	token.Connection.Settings = map[string]any{
		"hostname": "10.0.0.5",
		"port":     "3389",
		"username": "alice",
		"password": "secret",
		"width":    "1440",
	}
	token.Metadata = map[string]any{
		"userId":       "user-1",
		"connectionId": "conn-1",
		"recordingId":  "rec-1",
	}

	encrypted := mustEncryptToken(t, "integration-secret", token)

	decrypted, err := DecryptToken("integration-secret", encrypted)
	if err != nil {
		t.Fatalf("decrypt token: %v", err)
	}

	if decrypted.Connection.Type != "rdp" {
		t.Fatalf("unexpected connection type: %q", decrypted.Connection.Type)
	}
	if decrypted.Connection.GuacdHost != "desktop-proxy" {
		t.Fatalf("unexpected guacd host: %q", decrypted.Connection.GuacdHost)
	}
	if MetadataString(decrypted.Metadata, "recordingId") != "rec-1" {
		t.Fatalf("unexpected metadata recording id: %#v", decrypted.Metadata)
	}
}

func mustEncryptToken(t *testing.T, secret string, token ConnectionToken) string {
	t.Helper()

	payload, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	key, err := scrypt.Key([]byte(secret), []byte(guacamoleSalt), 16384, 8, 1, 32)
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}

	iv := []byte("0123456789ab")
	ciphertext := gcm.Seal(nil, iv, payload, nil)
	tagSize := gcm.Overhead()
	envelope := TokenEnvelope{
		IV:    base64.StdEncoding.EncodeToString(iv),
		Value: base64.StdEncoding.EncodeToString(ciphertext[:len(ciphertext)-tagSize]),
		Tag:   base64.StdEncoding.EncodeToString(ciphertext[len(ciphertext)-tagSize:]),
	}

	rawEnvelope, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return base64.StdEncoding.EncodeToString(rawEnvelope)
}
