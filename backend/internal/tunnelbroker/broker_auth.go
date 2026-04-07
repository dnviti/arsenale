package tunnelbroker

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func (b *Broker) authenticateTunnel(ctx context.Context, gatewayID, bearerToken, clientCertPEM string) (GatewayAuthRecord, error) {
	record, err := b.config.Store.LoadGatewayAuth(ctx, gatewayID)
	if err != nil {
		return GatewayAuthRecord{}, fmt.Errorf("load gateway auth: %w", err)
	}
	if !record.TunnelEnabled || record.TunnelTokenHash == "" {
		return GatewayAuthRecord{}, errors.New("gateway tunneling is disabled")
	}

	if hashToken(bearerToken) != record.TunnelTokenHash {
		return GatewayAuthRecord{}, errors.New("tunnel token mismatch")
	}
	if record.EncryptedTunnelToken != "" && record.TunnelTokenIV != "" && record.TunnelTokenTag != "" {
		plain, err := decryptWithServerKey(b.config.ServerEncryptionKey, record.EncryptedTunnelToken, record.TunnelTokenIV, record.TunnelTokenTag)
		if err != nil {
			return GatewayAuthRecord{}, fmt.Errorf("decrypt tunnel token: %w", err)
		}
		if subtle.ConstantTimeCompare([]byte(plain), []byte(bearerToken)) != 1 {
			return GatewayAuthRecord{}, errors.New("encrypted tunnel token mismatch")
		}
	}

	cert, err := parseClientCert(clientCertPEM)
	if err != nil {
		return GatewayAuthRecord{}, fmt.Errorf("parse client certificate: %w", err)
	}

	expectedSPIFFE := buildGatewaySPIFFEID(b.config.SpiffeTrustDomain, gatewayID)
	actualSPIFFE := extractSPIFFEID(cert)
	if subtle.ConstantTimeCompare([]byte(actualSPIFFE), []byte(expectedSPIFFE)) != 1 {
		return GatewayAuthRecord{}, fmt.Errorf("client certificate SPIFFE ID mismatch: got %q expected %q", actualSPIFFE, expectedSPIFFE)
	}

	if record.TenantTunnelCACertPEM != "" {
		if err := verifyCertChain(cert, record.TenantTunnelCACertPEM); err != nil {
			return GatewayAuthRecord{}, fmt.Errorf("client certificate does not chain to tenant CA: %w", err)
		}
	}

	return record, nil
}

func parseClientCertHeader(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func parseClientCert(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("missing PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func verifyCertChain(cert *x509.Certificate, caPEM string) error {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(caPEM)) {
		return errors.New("failed to parse tenant CA certificate")
	}
	_, err := cert.Verify(x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return err
}

func buildGatewaySPIFFEID(trustDomain, gatewayID string) string {
	return fmt.Sprintf("spiffe://%s/gateway/%s", strings.ToLower(strings.TrimSpace(trustDomain)), url.PathEscape(strings.TrimSpace(gatewayID)))
}

func extractSPIFFEID(cert *x509.Certificate) string {
	for _, uri := range cert.URIs {
		if uri == nil {
			continue
		}
		if uri.Scheme == "spiffe" {
			return uri.String()
		}
	}
	return ""
}

func extractClientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func decryptWithServerKey(key []byte, ciphertextHex, ivHex, tagHex string) (string, error) {
	if len(key) == 0 {
		return "", errors.New("server encryption key is required")
	}
	if len(key) != aesKeyBytes {
		return "", fmt.Errorf("server encryption key must be %d bytes", aesKeyBytes)
	}
	ciphertext, err := hex.DecodeString(ciphertextHex)
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
	if len(iv) != aesIVBytes {
		return "", fmt.Errorf("invalid iv length %d", len(iv))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithNonceSize(block, aesIVBytes)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	plaintext, err := aead.Open(nil, iv, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}
	return string(plaintext), nil
}

func LoadServerEncryptionKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SERVER_ENCRYPTION_KEY"))
	if raw == "" {
		if path := strings.TrimSpace(os.Getenv("SERVER_ENCRYPTION_KEY_FILE")); path != "" {
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read SERVER_ENCRYPTION_KEY_FILE: %w", err)
			}
			raw = strings.TrimSpace(string(payload))
		}
	}
	if raw == "" {
		return nil, nil
	}
	if len(raw) != aesKeyBytes*2 {
		return nil, fmt.Errorf("SERVER_ENCRYPTION_KEY must be exactly %d hex characters", aesKeyBytes*2)
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode SERVER_ENCRYPTION_KEY: %w", err)
	}
	return key, nil
}
