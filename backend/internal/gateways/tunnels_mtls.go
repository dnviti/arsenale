package gateways

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
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

	var (
		gatewayType                             string
		tunnelClientCert, tunnelClientKey       *string
		tunnelClientKeyIV, tunnelClientKeyTag   *string
		tunnelClientCertExp                     sql.NullTime
		tunnelServiceCert, tunnelServiceKey     *string
		tunnelServiceKeyIV, tunnelServiceKeyTag *string
		tunnelServiceCertExp                    sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
SELECT type::text,
       "tunnelClientCert", "tunnelClientKey", "tunnelClientKeyIV", "tunnelClientKeyTag", "tunnelClientCertExp",
       "tunnelServiceCert", "tunnelServiceKey", "tunnelServiceKeyIV", "tunnelServiceKeyTag", "tunnelServiceCertExp"
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
FOR UPDATE
`, gatewayID, tenantID).Scan(
		&gatewayType,
		&tunnelClientCert,
		&tunnelClientKey,
		&tunnelClientKeyIV,
		&tunnelClientKeyTag,
		&tunnelClientCertExp,
		&tunnelServiceCert,
		&tunnelServiceKey,
		&tunnelServiceKeyIV,
		&tunnelServiceKeyTag,
		&tunnelServiceCertExp,
	); err != nil {
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

	details := &mtlsAuditDetails{
		tenantCAGenerated: tenantGenerated,
		caFingerprint:     caFingerprint,
	}

	if tenantGenerated || tunnelCertMaterialNeedsRefresh(tunnelClientCert, tunnelClientKey, tunnelClientKeyIV, tunnelClientKeyTag, tunnelClientCertExp) {
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
		details.clientGenerated = true
		details.clientExpiry = clientExpiry
	}

	if gatewayruntime.ServiceTLSRequired(gatewayType) && (tenantGenerated || tunnelCertMaterialNeedsRefresh(tunnelServiceCert, tunnelServiceKey, tunnelServiceKeyIV, tunnelServiceKeyTag, tunnelServiceCertExp)) {
		serviceCertPEM, serviceKeyPEM, serviceExpiry, err := generateGatewayServiceCertificate(caCertPEM, caKeyPEM, gatewayID, gatewayType, s.tunnelTrustDomain())
		if err != nil {
			return nil, err
		}
		encServiceKey, err := encryptValue(s.ServerEncryptionKey, serviceKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("encrypt tunnel service key: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelServiceCert" = $2,
    "tunnelServiceKey" = $3,
    "tunnelServiceKeyIV" = $4,
    "tunnelServiceKeyTag" = $5,
    "tunnelServiceCertExp" = $6,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID, serviceCertPEM, encServiceKey.Ciphertext, encServiceKey.IV, encServiceKey.Tag, serviceExpiry.UTC()); err != nil {
			return nil, fmt.Errorf("store tunnel service certificate: %w", err)
		}
		details.serviceGenerated = true
		details.serviceExpiry = serviceExpiry
	}

	if !details.clientGenerated && !details.serviceGenerated && !details.tenantCAGenerated {
		return nil, nil
	}
	return details, nil
}

func tunnelCertMaterialNeedsRefresh(cert, key, keyIV, keyTag *string, exp sql.NullTime) bool {
	if strings.TrimSpace(derefString(cert)) == "" ||
		strings.TrimSpace(derefString(key)) == "" ||
		strings.TrimSpace(derefString(keyIV)) == "" ||
		strings.TrimSpace(derefString(keyTag)) == "" {
		return true
	}
	if !exp.Valid {
		return true
	}
	return !exp.Time.After(time.Now().UTC().Add(7 * 24 * time.Hour))
}

func generateGatewayServiceCertificate(caCertPEM, caKeyPEM, gatewayID, gatewayType, trustDomain string) (string, string, time.Time, error) {
	switch gatewayruntime.NormalizeType(gatewayType) {
	case gatewayruntime.TypeGuacd:
		certPEM, keyPEM, expiry, err := generateServerCertificate(
			caCertPEM,
			caKeyPEM,
			"arsenale-guacd",
			[]string{"arsenale-guacd", "guacd", "localhost"},
			[]net.IP{net.ParseIP("127.0.0.1")},
			[]string{buildGatewayServiceSPIFFEID(trustDomain, gatewayID, "guacd")},
		)
		if err != nil {
			return "", "", time.Time{}, fmt.Errorf("generate guacd service certificate: %w", err)
		}
		return certPEM, keyPEM, expiry, nil
	default:
		return "", "", time.Time{}, fmt.Errorf("gateway type %q does not require tunnel service TLS", gatewayType)
	}
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
