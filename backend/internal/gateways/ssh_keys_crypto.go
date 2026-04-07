package gateways

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func generateSSHKeyMaterial() (privatePEM string, publicKey string, fingerprint string, err error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("generate ed25519 key pair: %w", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privateBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})

	sshPublicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", "", "", fmt.Errorf("marshal public key: %w", err)
	}

	return string(privateBytes), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))), ssh.FingerprintSHA256(sshPublicKey), nil
}
