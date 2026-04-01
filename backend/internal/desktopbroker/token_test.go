package desktopbroker

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
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

func TestEncryptTokenReturnsURLSafeOuterEncoding(t *testing.T) {
	t.Parallel()

	token := sampleConnectionToken()

	encrypted, err := EncryptToken("integration-secret", token)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	if strings.ContainsAny(encrypted, "+/=") {
		t.Fatalf("expected URL-safe token, got %q", encrypted)
	}

	decrypted, err := DecryptToken("integration-secret", encrypted)
	if err != nil {
		t.Fatalf("decrypt token: %v", err)
	}
	if decrypted.Connection.Type != token.Connection.Type {
		t.Fatalf("unexpected connection type after round-trip: %q", decrypted.Connection.Type)
	}
	if MetadataString(decrypted.Metadata, "recordingId") != MetadataString(token.Metadata, "recordingId") {
		t.Fatalf("unexpected metadata after round-trip: %#v", decrypted.Metadata)
	}
}

func TestDecodeBase64TokenAcceptsLegacyPlusCorruptedByQueryDecoding(t *testing.T) {
	t.Parallel()

	decoded, err := decodeBase64Token("A AA")
	if err != nil {
		t.Fatalf("decode corrupted legacy base64: %v", err)
	}
	if encoded := base64.StdEncoding.EncodeToString(decoded); encoded != "A+AA" {
		t.Fatalf("unexpected decoded payload: %q", encoded)
	}
}

func TestLoadSecretTrimsTrailingNewlineFromFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "secret-*")
	if err != nil {
		t.Fatalf("create temp secret: %v", err)
	}

	if _, err := file.WriteString("integration-secret\r\n"); err != nil {
		t.Fatalf("write temp secret: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp secret: %v", err)
	}

	t.Setenv("ARSENALE_TEST_SECRET", "")
	t.Setenv("ARSENALE_TEST_SECRET_FILE", file.Name())

	value, err := LoadSecret("ARSENALE_TEST_SECRET", "ARSENALE_TEST_SECRET_FILE")
	if err != nil {
		t.Fatalf("load secret: %v", err)
	}
	if value != "integration-secret" {
		t.Fatalf("unexpected secret value: %q", value)
	}
}

func sampleConnectionToken() ConnectionToken {
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
	return token
}

func mustEncryptToken(t *testing.T, secret string, token ConnectionToken) string {
	t.Helper()

	return newLegacyTokenEncryptor(t, secret).encrypt(t, token)
}

type legacyTokenEncryptor struct {
	gcm cipher.AEAD
}

func newLegacyTokenEncryptor(t *testing.T, secret string) legacyTokenEncryptor {
	t.Helper()

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

	return legacyTokenEncryptor{gcm: gcm}
}

func (e legacyTokenEncryptor) encrypt(t *testing.T, token ConnectionToken) string {
	t.Helper()

	payload, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	iv := []byte("0123456789ab")
	ciphertext := e.gcm.Seal(nil, iv, payload, nil)
	tagSize := e.gcm.Overhead()
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
