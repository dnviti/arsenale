package gateways

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ensureTunnelMTLSMaterialTx reuses a tenant CA when present and only rotates missing gateway client cert material.
func (s Service) ensureTunnelMTLSMaterialTx(ctx context.Context, tx pgx.Tx, tenantID, gatewayID string) (*mtlsAuditDetails, error) {
	var tenantCACert, tenantCAKey, tenantCAKeyIV, tenantCAKeyTag, tenantCAFingerprint *string
	if err := tx.QueryRow(ctx, `
SELECT "tunnelCaCert", "tunnelCaKey", "tunnelCaKeyIV", "tunnelCaKeyTag", "tunnelCaCertFingerprint"
FROM "Tenant"
WHERE id = $1
FOR UPDATE
`, tenantID).Scan(&tenantCACert, &tenantCAKey, &tenantCAKeyIV, &tenantCAKeyTag, &tenantCAFingerprint); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: http.StatusNotFound, message: "Tenant not found"}
		}
		return nil, fmt.Errorf("load tenant tunnel CA: %w", err)
	}

	var tunnelClientCert, tunnelClientKey, tunnelClientKeyIV, tunnelClientKeyTag *string
	if err := tx.QueryRow(ctx, `
SELECT "tunnelClientCert", "tunnelClientKey", "tunnelClientKeyIV", "tunnelClientKeyTag"
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
FOR UPDATE
`, gatewayID, tenantID).Scan(&tunnelClientCert, &tunnelClientKey, &tunnelClientKeyIV, &tunnelClientKeyTag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return nil, fmt.Errorf("load gateway tunnel certificate: %w", err)
	}

	tenantGenerated := false
	caCertPEM := derefString(tenantCACert)
	caFingerprint := derefString(tenantCAFingerprint)
	var caKeyPEM string

	if strings.TrimSpace(caCertPEM) == "" || strings.TrimSpace(derefString(tenantCAKey)) == "" || strings.TrimSpace(derefString(tenantCAKeyIV)) == "" || strings.TrimSpace(derefString(tenantCAKeyTag)) == "" {
		certPEM, keyPEM, err := generateCACert("arsenale-tenant-" + tenantID)
		if err != nil {
			return nil, fmt.Errorf("generate tenant CA: %w", err)
		}
		encKey, err := encryptValue(s.ServerEncryptionKey, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("encrypt tenant CA key: %w", err)
		}
		fingerprint, err := certificateFingerprint(certPEM)
		if err != nil {
			return nil, fmt.Errorf("fingerprint tenant CA: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE "Tenant"
SET "tunnelCaCert" = $2,
    "tunnelCaKey" = $3,
    "tunnelCaKeyIV" = $4,
    "tunnelCaKeyTag" = $5,
    "tunnelCaCertFingerprint" = $6,
    "updatedAt" = NOW()
WHERE id = $1
`, tenantID, certPEM, encKey.Ciphertext, encKey.IV, encKey.Tag, fingerprint); err != nil {
			return nil, fmt.Errorf("store tenant CA: %w", err)
		}
		tenantGenerated = true
		caCertPEM = certPEM
		caKeyPEM = keyPEM
		caFingerprint = fingerprint
	} else {
		decryptedKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
			Ciphertext: derefString(tenantCAKey),
			IV:         derefString(tenantCAKeyIV),
			Tag:        derefString(tenantCAKeyTag),
		})
		if err != nil {
			return nil, fmt.Errorf("decrypt tenant CA key: %w", err)
		}
		caKeyPEM = decryptedKey
		if strings.TrimSpace(caFingerprint) == "" {
			fingerprint, err := certificateFingerprint(caCertPEM)
			if err != nil {
				return nil, fmt.Errorf("fingerprint tenant CA: %w", err)
			}
			caFingerprint = fingerprint
		}
	}

	needsClientCert := strings.TrimSpace(derefString(tunnelClientCert)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKey)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKeyIV)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKeyTag)) == ""
	if !needsClientCert {
		return nil, nil
	}

	clientCertPEM, clientKeyPEM, clientExpiry, err := generateClientCertificate(caCertPEM, caKeyPEM, gatewayID, buildGatewaySPIFFEID(s.tunnelTrustDomain(), gatewayID))
	if err != nil {
		return nil, fmt.Errorf("generate tunnel client certificate: %w", err)
	}
	encClientKey, err := encryptValue(s.ServerEncryptionKey, clientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("encrypt tunnel client key: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelClientCert" = $2,
    "tunnelClientKey" = $3,
    "tunnelClientKeyIV" = $4,
    "tunnelClientKeyTag" = $5,
    "tunnelClientCertExp" = $6,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID, clientCertPEM, encClientKey.Ciphertext, encClientKey.IV, encClientKey.Tag, clientExpiry.UTC()); err != nil {
		return nil, fmt.Errorf("store tunnel client certificate: %w", err)
	}

	return &mtlsAuditDetails{
		tenantCAGenerated: tenantGenerated,
		caFingerprint:     caFingerprint,
		clientExpiry:      clientExpiry,
	}, nil
}

func lockGatewayForTenant(ctx context.Context, tx pgx.Tx, tenantID, gatewayID string) error {
	var id string
	if err := tx.QueryRow(ctx, `
SELECT id
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
FOR UPDATE
`, gatewayID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return fmt.Errorf("load gateway: %w", err)
	}
	return nil
}

func (s Service) loadGatewayOwnership(ctx context.Context, tenantID, gatewayID string) (string, error) {
	if s.DB == nil {
		return "", errors.New("database is unavailable")
	}

	var id string
	if err := s.DB.QueryRow(ctx, `
SELECT id
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
`, gatewayID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return "", fmt.Errorf("load gateway: %w", err)
	}
	return id, nil
}
