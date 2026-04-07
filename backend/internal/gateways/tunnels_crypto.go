package gateways

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

func randomTunnelToken() (string, error) {
	raw := make([]byte, tunnelTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateCACert(commonName string) (string, string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate CA key pair: %w", err)
	}

	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          randomSerialNumber(),
		Subject:               pkix.Name{CommonName: commonName, Organization: []string{"Arsenale"}},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(tunnelCAValidityDays * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return "", "", fmt.Errorf("create CA certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal CA private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM), nil
}

func generateClientCertificate(caCertPEM, caKeyPEM, commonName, spiffeID string) (string, string, time.Time, error) {
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return "", "", time.Time{}, errors.New("decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse CA certificate: %w", err)
	}

	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return "", "", time.Time{}, errors.New("decode CA private key PEM")
	}
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse CA private key: %w", err)
	}

	uri, err := url.Parse(spiffeID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse SPIFFE ID: %w", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate client key pair: %w", err)
	}

	now := time.Now().UTC()
	expiry := now.Add(tunnelClientValidityDays * 24 * time.Hour)
	template := &x509.Certificate{
		SerialNumber:          randomSerialNumber(),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, privateKey.Public(), caKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("create client certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("marshal client private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM), expiry, nil
}

func certificateFingerprint(certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", errors.New("decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func randomSerialNumber() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return big.NewInt(time.Now().UnixNano())
	}
	return serial
}

func buildGatewaySPIFFEID(trustDomain, gatewayID string) string {
	domain := strings.ToLower(strings.TrimSpace(trustDomain))
	if domain == "" {
		domain = defaultTunnelTrustDomain
	}
	return fmt.Sprintf("spiffe://%s/gateway/%s", domain, url.PathEscape(strings.TrimSpace(gatewayID)))
}

func truncateString(value string, limit int) string {
	if len(value) <= limit || limit <= 0 {
		return value
	}
	return value[:limit]
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
