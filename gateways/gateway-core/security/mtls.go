// Package security provides mTLS validation, certificate pinning, and
// identity binding for Arsenale gateway agents.
package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"math"
	"time"
)

// MTLSConfig holds the parameters for a full mTLS validation pass.
type MTLSConfig struct {
	CACertPEM           string
	ClientCertPEM       string
	ExpectedGatewayID   string
	ExpectedFingerprint string // empty = skip pinning
}

// ValidateCertChain verifies the client certificate is signed by the given CA.
func ValidateCertChain(caCertPEM, clientCertPEM string) error {
	caCert, err := parsePEMCertificate(caCertPEM)
	if err != nil {
		return fmt.Errorf("mTLS validation failed: parsing CA cert: %w", err)
	}

	clientCert, err := parsePEMCertificate(clientCertPEM)
	if err != nil {
		return fmt.Errorf("mTLS validation failed: parsing client cert: %w", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	_, err = clientCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		return fmt.Errorf("mTLS validation failed: client cert not signed by CA: %w", err)
	}

	return nil
}

// VerifyCertExpiry checks how many days remain before a certificate expires.
// Returns negative days if already expired. Logs a warning if < 7 days remain.
func VerifyCertExpiry(certPEM string) (daysRemaining int, err error) {
	cert, err := parsePEMCertificate(certPEM)
	if err != nil {
		return 0, fmt.Errorf("mTLS validation failed: parsing cert for expiry check: %w", err)
	}

	remaining := time.Until(cert.NotAfter)
	days := int(math.Floor(remaining.Hours() / 24))

	if days < 7 {
		log.Printf("[security] certificate expiry warning: %d days remaining (expires %s)", days, cert.NotAfter.Format(time.RFC3339))
	}

	return days, nil
}

// ComputeCACertFingerprint returns the lowercase hex-encoded SHA-256 fingerprint
// of the DER-encoded CA certificate.
func ComputeCACertFingerprint(caCertPEM string) (string, error) {
	block, _ := pem.Decode([]byte(caCertPEM))
	if block == nil {
		return "", fmt.Errorf("mTLS validation failed: no PEM block found in CA cert")
	}

	hash := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(hash[:]), nil
}

// VerifyCACertFingerprint computes the CA certificate fingerprint and compares
// it against the expected value using constant-time comparison.
func VerifyCACertFingerprint(caCertPEM, expectedFingerprint string) error {
	actual, err := ComputeCACertFingerprint(caCertPEM)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(actual), []byte(expectedFingerprint)) != 1 {
		log.Printf("[security] CA fingerprint mismatch: expected=%s actual=%s", expectedFingerprint, actual)
		return fmt.Errorf("mTLS validation failed: CA certificate fingerprint mismatch")
	}

	return nil
}

// VerifyGatewayIdentity checks that the client certificate's Common Name
// matches the expected gateway ID, preventing gateway impersonation.
func VerifyGatewayIdentity(clientCertPEM, expectedGatewayID string) error {
	cert, err := parsePEMCertificate(clientCertPEM)
	if err != nil {
		return fmt.Errorf("mTLS validation failed: parsing client cert for identity check: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(cert.Subject.CommonName), []byte(expectedGatewayID)) != 1 {
		return fmt.Errorf("mTLS validation failed: gateway identity mismatch: cert CN=%q expected=%q", cert.Subject.CommonName, expectedGatewayID)
	}

	return nil
}

// ValidateConnectionForCredentialPush verifies TLS connection state before
// accepting CREDENTIAL_PUSH (type 12) frames. This MUST be called before
// processing any credential push to prevent unauthorized access.
func ValidateConnectionForCredentialPush(tlsState *tls.ConnectionState, expectedGatewayID string) error {
	if tlsState == nil {
		return fmt.Errorf("mTLS validation failed: connection is not TLS")
	}

	if !tlsState.HandshakeComplete {
		return fmt.Errorf("mTLS validation failed: TLS handshake not complete")
	}

	if len(tlsState.PeerCertificates) == 0 {
		return fmt.Errorf("mTLS validation failed: no peer certificates presented")
	}

	peerCN := tlsState.PeerCertificates[0].Subject.CommonName
	if subtle.ConstantTimeCompare([]byte(peerCN), []byte(expectedGatewayID)) != 1 {
		return fmt.Errorf("mTLS validation failed: peer identity mismatch: cert CN=%q expected=%q", peerCN, expectedGatewayID)
	}

	return nil
}

// MustValidateMTLS runs all mTLS validations in sequence: cert chain, expiry,
// fingerprint pinning (if configured), and identity binding. Returns the first
// error encountered.
func MustValidateMTLS(cfg MTLSConfig) error {
	// 1. Validate certificate chain.
	if err := ValidateCertChain(cfg.CACertPEM, cfg.ClientCertPEM); err != nil {
		return err
	}

	// 2. Check certificate expiry.
	days, err := VerifyCertExpiry(cfg.ClientCertPEM)
	if err != nil {
		return err
	}
	if days < 0 {
		return fmt.Errorf("mTLS validation failed: client certificate expired %d days ago", -days)
	}

	// 3. Fingerprint pinning (optional).
	if cfg.ExpectedFingerprint != "" {
		if err := VerifyCACertFingerprint(cfg.CACertPEM, cfg.ExpectedFingerprint); err != nil {
			return err
		}
	}

	// 4. Gateway identity binding.
	if err := VerifyGatewayIdentity(cfg.ClientCertPEM, cfg.ExpectedGatewayID); err != nil {
		return err
	}

	return nil
}

// parsePEMCertificate decodes a PEM string and parses the X.509 certificate.
func parsePEMCertificate(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}
