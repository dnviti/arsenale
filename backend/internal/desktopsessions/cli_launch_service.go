package desktopsessions

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/jackc/pgx/v5"
)

const (
	defaultDesktopLaunchTTLSeconds = 60
	defaultViewerControlTTLSeconds = 12 * 60 * 60
	desktopLaunchViewerPath        = "/cli/desktop-launch"
	desktopLaunchWebSocketPath     = "/guacamole/"
)

func (s Service) CreateDesktopLaunch(r *http.Request, claims authn.Claims, payload desktopLaunchRequest) (desktopLaunchResponse, error) {
	if s.DB == nil {
		return desktopLaunchResponse{}, fmt.Errorf("database is unavailable")
	}
	if strings.TrimSpace(claims.UserID) == "" {
		return desktopLaunchResponse{}, &requestError{status: http.StatusUnauthorized, message: "Invalid or expired token"}
	}

	protocol, err := normalizeDesktopLaunchProtocol(payload.Protocol)
	if err != nil {
		return desktopLaunchResponse{}, err
	}
	connectionID := strings.TrimSpace(payload.ConnectionID)
	if connectionID == "" {
		return desktopLaunchResponse{}, &requestError{status: http.StatusBadRequest, message: "connectionId is required"}
	}
	if err := s.checkDesktopLaunchAccess(r.Context(), claims, connectionID, protocol); err != nil {
		return desktopLaunchResponse{}, err
	}

	grantID, grantSecret, grantValue, err := newOpaqueGrant()
	if err != nil {
		return desktopLaunchResponse{}, fmt.Errorf("generate desktop launch grant: %w", err)
	}
	expiresIn := parseEnvInt("CLI_DESKTOP_LAUNCH_TTL_SECONDS", defaultDesktopLaunchTTLSeconds)
	if expiresIn <= 0 {
		expiresIn = defaultDesktopLaunchTTLSeconds
	}
	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)

	if _, err := s.DB.Exec(r.Context(), `
INSERT INTO "DesktopLaunchGrant" (
  id, "secretHash", "tenantId", "userId", "connectionId", protocol, "expiresAt", "createdIpAddress", "createdUserAgent"
) VALUES (
  $1, $2, NULLIF($3, ''), $4, $5, $6::"SessionProtocol", $7, NULLIF($8, ''), NULLIF($9, '')
)`,
		grantID,
		hashOpaqueSecret(grantSecret),
		claims.TenantID,
		claims.UserID,
		connectionID,
		protocol,
		expiresAt,
		requestIP(r),
		r.UserAgent(),
	); err != nil {
		return desktopLaunchResponse{}, fmt.Errorf("store desktop launch grant: %w", err)
	}

	return desktopLaunchResponse{
		Protocol:     protocol,
		ConnectionID: connectionID,
		LaunchURL:    s.desktopLaunchURL(r, grantValue),
		ExpiresAt:    expiresAt,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s Service) RedeemDesktopLaunch(r *http.Request, grantValue string) (desktopLaunchRedeemResponse, error) {
	if s.DB == nil || s.Store == nil {
		return desktopLaunchRedeemResponse{}, fmt.Errorf("desktop session dependencies are unavailable")
	}
	grant, err := s.consumeDesktopLaunchGrant(r.Context(), grantValue, requestIP(r))
	if err != nil {
		return desktopLaunchRedeemResponse{}, err
	}

	claims := authn.Claims{
		UserID:   grant.UserID,
		TenantID: grant.TenantID,
	}
	metadataCtx := sessionErrorContext{
		ConnectionID: grant.ConnectionID,
	}
	result, err := s.createDesktopSession(
		r.Context(),
		claims,
		createRequest{ConnectionID: grant.ConnectionID},
		grant.Protocol,
		requestIP(r),
		&metadataCtx,
	)
	if err != nil {
		s.recordSessionError(r.Context(), grant.UserID, grant.Protocol, metadataCtx, requestIP(r), err)
		return desktopLaunchRedeemResponse{}, err
	}

	controlToken, controlExpiresAt, err := s.issueViewerControlToken(r.Context(), grant, result.SessionID)
	if err != nil {
		_ = s.Store.EndOwnedSession(r.Context(), result.SessionID, grant.UserID, "cli_viewer_control_issue_failed")
		return desktopLaunchRedeemResponse{}, err
	}
	_ = s.markDesktopLaunchSession(r.Context(), grant.ID, result.SessionID)

	return desktopLaunchRedeemResponse{
		Protocol:              grant.Protocol,
		ConnectionID:          grant.ConnectionID,
		SessionID:             result.SessionID,
		Token:                 result.Token,
		WebSocketPath:         desktopLaunchWebSocketPath,
		ControlToken:          controlToken,
		ControlTokenExpiresAt: controlExpiresAt,
		EnableDrive:           result.EnableDrive,
		RecordingID:           result.RecordingID,
		DLPPolicy:             result.DLPPolicy,
		ResolvedUsername:      result.ResolvedUsername,
		ResolvedDomain:        result.ResolvedDomain,
	}, nil
}

func normalizeDesktopLaunchProtocol(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "RDP":
		return "RDP", nil
	case "VNC":
		return "VNC", nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "protocol must be RDP or VNC"}
	}
}

func (s Service) checkDesktopLaunchAccess(ctx context.Context, claims authn.Claims, connectionID, protocol string) error {
	if claims.TenantID != "" {
		membership, err := s.TenantAuth.ResolveMembership(ctx, claims.UserID, claims.TenantID)
		if err != nil {
			return fmt.Errorf("resolve tenant membership: %w", err)
		}
		if membership == nil || !membership.Permissions[tenantauth.CanConnect] {
			return &requestError{status: http.StatusForbidden, message: "Not allowed to start sessions in this tenant"}
		}
	}
	conn, err := s.Connections.GetConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return &requestError{status: http.StatusNotFound, message: "Connection not found or access denied"}
	}
	if !strings.EqualFold(conn.Type, protocol) {
		return &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("connection is type %s, not %s", conn.Type, protocol)}
	}
	return nil
}

func (s Service) desktopLaunchURL(r *http.Request, grant string) string {
	base := strings.TrimRight(strings.TrimSpace(s.ClientURL), "/")
	if base == "" {
		base = requestOrigin(r)
	}

	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		parsed = &url.URL{Scheme: "https", Host: "localhost:3000"}
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + desktopLaunchViewerPath
	parsed.RawQuery = ""
	q := parsed.Query()
	q.Set("grant", grant)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func requestOrigin(r *http.Request) string {
	scheme := firstNonEmpty(firstForwardedHeader(r.Header.Get("X-Forwarded-Proto")), "http")
	if r.TLS != nil && scheme == "http" {
		scheme = "https"
	}
	host := firstNonEmpty(firstForwardedHeader(r.Header.Get("X-Forwarded-Host")), strings.TrimSpace(r.Host), "localhost")
	return scheme + "://" + host
}

func (s Service) consumeDesktopLaunchGrant(ctx context.Context, grantValue, ipAddress string) (desktopLaunchGrantRecord, error) {
	grantID, grantSecret, err := splitOpaqueGrant(grantValue)
	if err != nil {
		return desktopLaunchGrantRecord{}, &requestError{status: http.StatusBadRequest, message: "invalid launch grant"}
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return desktopLaunchGrantRecord{}, fmt.Errorf("begin consume desktop launch grant: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := loadDesktopLaunchGrantForUpdate(ctx, tx, grantID)
	if err != nil {
		return desktopLaunchGrantRecord{}, err
	}
	if !opaqueSecretMatches(grantSecret, record.SecretHash) {
		return desktopLaunchGrantRecord{}, &requestError{status: http.StatusNotFound, message: "launch grant not found"}
	}
	if record.Consumed {
		return desktopLaunchGrantRecord{}, &requestError{status: http.StatusConflict, message: "launch grant has already been used"}
	}
	if !time.Now().UTC().Before(record.ExpiresAt.UTC()) {
		return desktopLaunchGrantRecord{}, &requestError{status: http.StatusGone, message: "launch grant has expired"}
	}

	if _, err := tx.Exec(ctx, `
UPDATE "DesktopLaunchGrant"
SET "consumedAt" = NOW(), "consumedIpAddress" = NULLIF($2, '')
WHERE id = $1`, record.ID, ipAddress); err != nil {
		return desktopLaunchGrantRecord{}, fmt.Errorf("consume desktop launch grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return desktopLaunchGrantRecord{}, fmt.Errorf("commit desktop launch grant: %w", err)
	}
	return record, nil
}

func loadDesktopLaunchGrantForUpdate(ctx context.Context, tx pgx.Tx, id string) (desktopLaunchGrantRecord, error) {
	var (
		record     desktopLaunchGrantRecord
		tenantID   sql.NullString
		consumedAt sql.NullTime
	)
	err := tx.QueryRow(ctx, `
SELECT id, "secretHash", COALESCE("tenantId", ''), "userId", "connectionId", protocol::text, "expiresAt", "consumedAt"
FROM "DesktopLaunchGrant"
WHERE id = $1
FOR UPDATE`, id).Scan(
		&record.ID,
		&record.SecretHash,
		&tenantID,
		&record.UserID,
		&record.ConnectionID,
		&record.Protocol,
		&record.ExpiresAt,
		&consumedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return desktopLaunchGrantRecord{}, &requestError{status: http.StatusNotFound, message: "launch grant not found"}
		}
		return desktopLaunchGrantRecord{}, fmt.Errorf("load desktop launch grant: %w", err)
	}
	record.TenantID = tenantID.String
	record.Consumed = consumedAt.Valid
	return record, nil
}

func (s Service) markDesktopLaunchSession(ctx context.Context, grantID, sessionID string) error {
	if s.DB == nil {
		return nil
	}
	_, err := s.DB.Exec(ctx, `UPDATE "DesktopLaunchGrant" SET "consumedSessionId" = $2 WHERE id = $1`, grantID, sessionID)
	return err
}
