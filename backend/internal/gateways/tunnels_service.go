package gateways

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
	"github.com/jackc/pgx/v5"
)

func (s Service) GetTunnelOverview(ctx context.Context, tenantID string) (tunnelOverviewResponse, error) {
	if s.DB == nil {
		return tunnelOverviewResponse{}, errors.New("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT id
FROM "Gateway"
WHERE "tenantId" = $1
  AND "tunnelEnabled" = true
`, tenantID)
	if err != nil {
		return tunnelOverviewResponse{}, fmt.Errorf("list tunneled gateways: %w", err)
	}
	defer rows.Close()

	enabledGateways := make(map[string]struct{})
	for rows.Next() {
		var gatewayID string
		if err := rows.Scan(&gatewayID); err != nil {
			return tunnelOverviewResponse{}, fmt.Errorf("scan tunneled gateway: %w", err)
		}
		enabledGateways[gatewayID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return tunnelOverviewResponse{}, fmt.Errorf("iterate tunneled gateways: %w", err)
	}

	statuses, err := s.listTunnelStatuses(ctx)
	if err != nil {
		statuses = nil
	}
	return aggregateTunnelOverview(enabledGateways, statuses), nil
}

func (s Service) GenerateTunnelToken(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (tunnelTokenResponse, error) {
	if s.DB == nil {
		return tunnelTokenResponse{}, errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("begin tunnel token transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	mtlsDetails, err := s.ensureTunnelMTLSMaterialTx(ctx, tx, claims.TenantID, gatewayID)
	if err != nil {
		return tunnelTokenResponse{}, err
	}

	rawToken, err := randomTunnelToken()
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("generate tunnel token: %w", err)
	}
	encToken, err := encryptValue(s.ServerEncryptionKey, rawToken)
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("encrypt tunnel token: %w", err)
	}
	tokenHash := hashToken(rawToken)

	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelEnabled" = true,
    "encryptedTunnelToken" = $2,
    "tunnelTokenIV" = $3,
    "tunnelTokenTag" = $4,
    "tunnelTokenHash" = $5,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID, encToken.Ciphertext, encToken.IV, encToken.Tag, tokenHash); err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("store tunnel token: %w", err)
	}

	if mtlsDetails != nil {
		if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_GENERATE", gatewayID, map[string]any{
			"mtlsCertsGenerated": true,
			"tenantId":           claims.TenantID,
			"tenantCaGenerated":  mtlsDetails.tenantCAGenerated,
			"caFingerprint":      truncateString(mtlsDetails.caFingerprint, 16),
			"clientCertExpiry":   mtlsDetails.clientExpiry.UTC().Format(time.RFC3339),
		}, ipAddress); err != nil {
			return tunnelTokenResponse{}, err
		}
	}
	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_GENERATE", gatewayID, nil, ipAddress); err != nil {
		return tunnelTokenResponse{}, err
	}

	bundle, err := s.loadTunnelTokenBundleTx(ctx, tx, claims.TenantID, gatewayID)
	if err != nil {
		return tunnelTokenResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("commit tunnel token transaction: %w", err)
	}

	bundle.Token = rawToken
	bundle.TunnelEnabled = true
	bundle.TunnelConnected = false
	return bundle, nil
}

func (s Service) loadTunnelTokenBundleTx(ctx context.Context, tx pgx.Tx, tenantID, gatewayID string) (tunnelTokenResponse, error) {
	var (
		gatewayType         string
		gatewayPort         int
		tunnelClientCert    *string
		tunnelClientKey     *string
		tunnelClientKeyIV   *string
		tunnelClientKeyTag  *string
		tunnelClientCertExp sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
SELECT type::text, port, "tunnelClientCert", "tunnelClientKey", "tunnelClientKeyIV", "tunnelClientKeyTag", "tunnelClientCertExp"
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
`, gatewayID, tenantID).Scan(
		&gatewayType,
		&gatewayPort,
		&tunnelClientCert,
		&tunnelClientKey,
		&tunnelClientKeyIV,
		&tunnelClientKeyTag,
		&tunnelClientCertExp,
	); err != nil {
		return tunnelTokenResponse{}, mapLoadGatewayError(err)
	}

	clientKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: derefString(tunnelClientKey),
		IV:         derefString(tunnelClientKeyIV),
		Tag:        derefString(tunnelClientKeyTag),
	})
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("decrypt tunnel client key: %w", err)
	}

	var certExp *time.Time
	if tunnelClientCertExp.Valid {
		certExp = &tunnelClientCertExp.Time
	}

	return tunnelTokenResponse{
		GatewayID:        gatewayID,
		GatewayType:      gatewayType,
		TunnelLocalHost:  gatewayruntime.TunnelLocalHost(gatewayType),
		TunnelLocalPort:  tunnelLocalPortForGateway(gatewayType, gatewayPort),
		TunnelClientCert: strings.TrimSpace(derefString(tunnelClientCert)),
		TunnelClientKey:  strings.TrimSpace(clientKey),
		TunnelClientExp:  certExp,
	}, nil
}

func tunnelLocalPortForGateway(gatewayType string, configuredPort int) int {
	return gatewayruntime.TunnelLocalPort(gatewayType, configuredPort)
}

func (s Service) RevokeTunnelToken(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (revokeTunnelTokenResponse, error) {
	if s.DB == nil {
		return revokeTunnelTokenResponse{}, errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("begin tunnel revoke transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockGatewayForTenant(ctx, tx, claims.TenantID, gatewayID); err != nil {
		return revokeTunnelTokenResponse{}, err
	}

	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelEnabled" = false,
    "encryptedTunnelToken" = NULL,
    "tunnelTokenIV" = NULL,
    "tunnelTokenTag" = NULL,
    "tunnelTokenHash" = NULL,
    "tunnelConnectedAt" = NULL,
    "tunnelLastHeartbeat" = NULL,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID); err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("revoke tunnel token: %w", err)
	}

	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_ROTATE", gatewayID, map[string]any{
		"revoked": true,
	}, ipAddress); err != nil {
		return revokeTunnelTokenResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("commit tunnel revoke transaction: %w", err)
	}

	_ = s.disconnectTunnel(ctx, gatewayID)

	return revokeTunnelTokenResponse{
		Revoked:       true,
		TunnelEnabled: false,
	}, nil
}

func (s Service) ForceDisconnectTunnel(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (disconnectTunnelResponse, error) {
	if s.DB == nil {
		return disconnectTunnelResponse{}, errors.New("database is unavailable")
	}

	if _, err := s.loadGatewayOwnership(ctx, claims.TenantID, gatewayID); err != nil {
		return disconnectTunnelResponse{}, err
	}

	status, err := s.getTunnelStatus(ctx, gatewayID)
	if err != nil || status == nil || !status.Connected {
		return disconnectTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "Tunnel is not connected"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("begin tunnel disconnect transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockGatewayForTenant(ctx, tx, claims.TenantID, gatewayID); err != nil {
		return disconnectTunnelResponse{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelConnectedAt" = NULL,
    "tunnelLastHeartbeat" = NULL,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID); err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("clear tunnel connection state: %w", err)
	}
	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_DISCONNECT", gatewayID, map[string]any{
		"forced": true,
	}, ipAddress); err != nil {
		return disconnectTunnelResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("commit tunnel disconnect transaction: %w", err)
	}

	_ = s.disconnectTunnel(ctx, gatewayID)

	return disconnectTunnelResponse{Disconnected: true}, nil
}

func (s Service) GetTunnelEvents(ctx context.Context, tenantID, gatewayID string) (tunnelEventsResponse, error) {
	if s.DB == nil {
		return tunnelEventsResponse{}, errors.New("database is unavailable")
	}
	if _, err := s.loadGatewayOwnership(ctx, tenantID, gatewayID); err != nil {
		return tunnelEventsResponse{}, err
	}

	rows, err := s.DB.Query(ctx, `
SELECT action::text, "createdAt", details, "ipAddress"
FROM "AuditLog"
WHERE "targetId" = $1
  AND "targetType" = 'Gateway'
  AND action IN ('TUNNEL_CONNECT'::"AuditAction", 'TUNNEL_DISCONNECT'::"AuditAction")
ORDER BY "createdAt" DESC
LIMIT 20
`, gatewayID)
	if err != nil {
		return tunnelEventsResponse{}, fmt.Errorf("list tunnel events: %w", err)
	}
	defer rows.Close()

	events := make([]tunnelEventResponse, 0)
	for rows.Next() {
		var item tunnelEventResponse
		var rawDetails []byte
		if err := rows.Scan(&item.Action, &item.Timestamp, &rawDetails, &item.IPAddress); err != nil {
			return tunnelEventsResponse{}, fmt.Errorf("scan tunnel event: %w", err)
		}
		item.Details = sanitizeTunnelEventDetails(rawDetails)
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return tunnelEventsResponse{}, fmt.Errorf("iterate tunnel events: %w", err)
	}

	return tunnelEventsResponse{Events: events}, nil
}

func (s Service) GetTunnelMetrics(ctx context.Context, tenantID, gatewayID string) (tunnelMetricsResponse, error) {
	if s.DB == nil {
		return tunnelMetricsResponse{}, errors.New("database is unavailable")
	}
	if _, err := s.loadGatewayOwnership(ctx, tenantID, gatewayID); err != nil {
		return tunnelMetricsResponse{}, err
	}

	status, err := s.getTunnelStatus(ctx, gatewayID)
	if err != nil {
		return tunnelMetricsResponse{Connected: false}, nil
	}
	return tunnelMetricsFromStatus(status), nil
}
