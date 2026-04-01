package connections

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) CreateConnection(ctx context.Context, claims authn.Claims, payload createPayload, ip *string) (connectionResponse, error) {
	credentialSecretID := normalizeOptionalStringPtrValue(payload.CredentialSecretID)
	externalVaultProviderID := normalizeOptionalStringPtrValue(payload.ExternalVaultProviderID)
	externalVaultPath := normalizeOptionalStringPtrValue(payload.ExternalVaultPath)
	teamID := normalizeOptionalStringPtrValue(payload.TeamID)
	folderID := normalizeOptionalStringPtrValue(payload.FolderID)

	if credentialSecretID == nil && externalVaultProviderID == nil && (payload.Username == nil || payload.Password == nil) {
		return connectionResponse{}, &requestError{status: 400, message: "Either credentialSecretId, externalVaultProviderId, or both username and password must be provided"}
	}
	if s.DB == nil {
		return connectionResponse{}, errors.New("database is unavailable")
	}

	name := strings.TrimSpace(payload.Name)
	host := strings.TrimSpace(payload.Host)
	connType := strings.ToUpper(strings.TrimSpace(payload.Type))
	if name == "" {
		return connectionResponse{}, &requestError{status: 400, message: "name is required"}
	}
	if host == "" {
		return connectionResponse{}, &requestError{status: 400, message: "host is required"}
	}
	if payload.Port < 1 || payload.Port > 65535 {
		return connectionResponse{}, &requestError{status: 400, message: "port must be between 1 and 65535"}
	}
	if !validConnectionType(connType) {
		return connectionResponse{}, &requestError{status: 400, message: "type must be one of RDP, SSH, VNC, DATABASE, DB_TUNNEL"}
	}
	if err := validateConnectionHost(ctx, host); err != nil {
		return connectionResponse{}, err
	}
	gatewayID := normalizeOptionalStringPtrValue(payload.GatewayID)
	if gatewayRoutingMandatoryEnabled() && gatewayID == nil && connectionTypeRequiresGateway(connType) {
		resolvedGatewayID, err := s.resolveDefaultGatewayID(ctx, claims.TenantID, connType)
		if err != nil {
			return connectionResponse{}, err
		}
		gatewayID = resolvedGatewayID
	}
	if gatewayID != nil {
		if err := s.validateGatewayForConnectionType(ctx, claims.TenantID, *gatewayID, connType); err != nil {
			return connectionResponse{}, err
		}
	}
	if teamID != nil {
		if _, err := s.requireTeamRole(ctx, claims.UserID, claims.TenantID, *teamID); err != nil {
			return connectionResponse{}, err
		}
	}
	if credentialSecretID != nil {
		if err := s.validateCredentialSecretReference(ctx, claims.UserID, claims.TenantID, *credentialSecretID, connType); err != nil {
			return connectionResponse{}, err
		}
	}

	var (
		err         error
		key         []byte
		encUser     *encryptedField
		encPassword *encryptedField
		encDomain   *encryptedField
	)
	if payload.Username != nil && payload.Password != nil {
		key, err = s.resolveConnectionEncryptionKey(ctx, claims.UserID, teamID)
		if err != nil {
			return connectionResponse{}, err
		}
		defer zeroBytes(key)

		username := strings.TrimSpace(*payload.Username)
		password := *payload.Password
		if username == "" || password == "" {
			return connectionResponse{}, &requestError{status: 400, message: "username and password are required"}
		}

		encryptedUsername, err := encryptValue(key, username)
		if err != nil {
			return connectionResponse{}, err
		}
		encUser = &encryptedUsername

		encryptedPassword, err := encryptValue(key, password)
		if err != nil {
			return connectionResponse{}, err
		}
		encPassword = &encryptedPassword

		if payload.Domain != nil && strings.TrimSpace(*payload.Domain) != "" {
			encryptedDomain, err := encryptValue(key, strings.TrimSpace(*payload.Domain))
			if err != nil {
				return connectionResponse{}, err
			}
			encDomain = &encryptedDomain
		}
	}

	connectionID := uuid.NewString()
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "Connection" (
	id,
	"name",
	type,
	host,
	port,
	"folderId",
	"teamId",
	"credentialSecretId",
	"externalVaultProviderId",
	"externalVaultPath",
	"userId",
	"encryptedUsername",
	"usernameIV",
	"usernameTag",
	"encryptedPassword",
	"passwordIV",
	"passwordTag",
	"encryptedDomain",
	"domainIV",
	"domainTag",
	description,
	"enableDrive",
	"gatewayId",
	"sshTerminalConfig",
	"rdpSettings",
	"vncSettings",
	"dbSettings",
	"dlpPolicy",
	"defaultCredentialMode",
	"targetDbHost",
	"targetDbPort",
	"dbType",
	"bastionConnectionId",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1,
	$2,
	$3::"ConnectionType",
	$4,
	$5,
	$6,
	$7,
	$8,
	$9,
	$10,
	$11,
	$12,
	$13,
	$14,
	$15,
	$16,
	$17,
	$18,
	$19,
	$20,
	$21,
	$22,
	$23,
	$24,
	$25,
	$26,
	$27,
	$28,
	$29,
	$30,
	$31,
	$32,
	$33,
	$34,
	$35
)
RETURNING id
`,
		connectionID,
		name,
		connType,
		host,
		payload.Port,
		nullableString(folderID),
		nullableString(teamID),
		nullableString(credentialSecretID),
		nullableString(externalVaultProviderID),
		nullableString(externalVaultPath),
		claims.UserID,
		nullCiphertext(encUser),
		nullIV(encUser),
		nullTag(encUser),
		nullCiphertext(encPassword),
		nullIV(encPassword),
		nullTag(encPassword),
		nullCiphertext(encDomain),
		nullIV(encDomain),
		nullTag(encDomain),
		nullableString(payload.Description),
		boolOrDefault(payload.EnableDrive, false),
		nullableString(gatewayID),
		nullableJSON(payload.SSHTerminalConfig),
		nullableJSON(payload.RDPSettings),
		nullableJSON(payload.VNCSettings),
		nullableJSON(payload.DBSettings),
		nullableJSON(payload.DLPPolicy),
		nullableString(payload.DefaultCredentialMode),
		nullableString(payload.TargetDBHost),
		nullableInt(payload.TargetDBPort),
		nullableString(payload.DBType),
		nullableString(payload.BastionConnectionID),
		time.Now(),
		time.Now(),
	).Scan(&connectionID); err != nil {
		return connectionResponse{}, fmt.Errorf("create connection: %w", err)
	}

	_ = s.insertAuditLog(ctx, claims.UserID, "CREATE_CONNECTION", connectionID, map[string]any{
		"name":   name,
		"type":   connType,
		"host":   host,
		"teamId": teamID,
	}, ip)
	return s.GetConnection(ctx, claims.UserID, claims.TenantID, connectionID)
}

func (s Service) ImportSimpleConnection(ctx context.Context, claims authn.Claims, payload ImportPayload, ip *string) (connectionResponse, error) {
	username := strings.TrimSpace(payload.Username)
	password := payload.Password
	create := createPayload{
		Name:        strings.TrimSpace(payload.Name),
		Type:        strings.TrimSpace(payload.Type),
		Host:        strings.TrimSpace(payload.Host),
		Port:        payload.Port,
		Username:    &username,
		Password:    &password,
		Domain:      payload.Domain,
		FolderID:    payload.FolderID,
		Description: payload.Description,
	}
	return s.CreateConnection(ctx, claims, create, ip)
}

func (s Service) UpdateConnection(ctx context.Context, claims authn.Claims, connectionID string, payload updatePayload, ip *string) (connectionResponse, error) {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return connectionResponse{}, err
	}
	if access.AccessType == "shared" {
		return connectionResponse{}, pgx.ErrNoRows
	}
	if access.AccessType == "team" && (access.Connection.TeamRole == nil || !canManageTeam(*access.Connection.TeamRole)) {
		return connectionResponse{}, pgx.ErrNoRows
	}

	var updates []string
	var args []any
	addUpdate := func(column string, value any) {
		updates = append(updates, fmt.Sprintf(`%s = $%d`, column, len(args)+1))
		args = append(args, value)
	}

	if payload.Name.Present {
		if payload.Name.Value == nil || strings.TrimSpace(*payload.Name.Value) == "" {
			return connectionResponse{}, &requestError{status: 400, message: "name is required"}
		}
		addUpdate(`name`, strings.TrimSpace(*payload.Name.Value))
	}
	effectiveType := access.Connection.Type
	if payload.Type.Present {
		if payload.Type.Value == nil {
			return connectionResponse{}, &requestError{status: 400, message: "type is required"}
		}
		connType := strings.ToUpper(strings.TrimSpace(*payload.Type.Value))
		if !validConnectionType(connType) {
			return connectionResponse{}, &requestError{status: 400, message: "type must be one of RDP, SSH, VNC, DATABASE, DB_TUNNEL"}
		}
		effectiveType = connType
		updates = append(updates, fmt.Sprintf(`type = $%d::"ConnectionType"`, len(args)+1))
		args = append(args, connType)
	}
	if payload.Host.Present {
		if payload.Host.Value == nil || strings.TrimSpace(*payload.Host.Value) == "" {
			return connectionResponse{}, &requestError{status: 400, message: "host is required"}
		}
		host := strings.TrimSpace(*payload.Host.Value)
		if err := validateConnectionHost(ctx, host); err != nil {
			return connectionResponse{}, err
		}
		addUpdate(`host`, host)
	}
	if payload.Port.Present {
		if payload.Port.Value == nil || *payload.Port.Value < 1 || *payload.Port.Value > 65535 {
			return connectionResponse{}, &requestError{status: 400, message: "port must be between 1 and 65535"}
		}
		addUpdate(`port`, *payload.Port.Value)
	}
	if payload.Description.Present {
		addUpdate(`description`, nullableString(payload.Description.Value))
	}
	if payload.FolderID.Present {
		addUpdate(`"folderId"`, nullableString(payload.FolderID.Value))
	}
	if payload.EnableDrive.Present {
		if payload.EnableDrive.Value == nil {
			return connectionResponse{}, &requestError{status: 400, message: "enableDrive must be a boolean"}
		}
		addUpdate(`"enableDrive"`, *payload.EnableDrive.Value)
	}
	if payload.SSHTerminalConfig.Present {
		addUpdate(`"sshTerminalConfig"`, nullableJSON(payload.SSHTerminalConfig.Value))
	}
	if payload.RDPSettings.Present {
		addUpdate(`"rdpSettings"`, nullableJSON(payload.RDPSettings.Value))
	}
	if payload.VNCSettings.Present {
		addUpdate(`"vncSettings"`, nullableJSON(payload.VNCSettings.Value))
	}
	if payload.DBSettings.Present {
		addUpdate(`"dbSettings"`, nullableJSON(payload.DBSettings.Value))
	}
	if payload.DLPPolicy.Present {
		addUpdate(`"dlpPolicy"`, nullableJSON(payload.DLPPolicy.Value))
	}
	if payload.DefaultCredentialMode.Present {
		addUpdate(`"defaultCredentialMode"`, nullableString(payload.DefaultCredentialMode.Value))
	}
	if payload.TargetDBHost.Present {
		addUpdate(`"targetDbHost"`, nullableString(payload.TargetDBHost.Value))
	}
	if payload.TargetDBPort.Present {
		addUpdate(`"targetDbPort"`, nullableInt(payload.TargetDBPort.Value))
	}
	if payload.DBType.Present {
		addUpdate(`"dbType"`, nullableString(payload.DBType.Value))
	}
	if payload.BastionConnectionID.Present {
		addUpdate(`"bastionConnectionId"`, nullableString(payload.BastionConnectionID.Value))
	}

	effectiveGatewayID := normalizeOptionalStringPtrValue(access.Connection.GatewayID)
	if payload.GatewayID.Present {
		effectiveGatewayID = normalizeOptionalStringPtrValue(payload.GatewayID.Value)
	}
	if gatewayRoutingMandatoryEnabled() && effectiveGatewayID == nil && connectionTypeRequiresGateway(effectiveType) {
		effectiveGatewayID, err = s.resolveDefaultGatewayID(ctx, claims.TenantID, effectiveType)
		if err != nil {
			return connectionResponse{}, err
		}
	}
	if effectiveGatewayID != nil {
		if err := s.validateGatewayForConnectionType(ctx, claims.TenantID, *effectiveGatewayID, effectiveType); err != nil {
			return connectionResponse{}, err
		}
	}
	if payload.GatewayID.Present || !optionalStringPointersEqual(access.Connection.GatewayID, effectiveGatewayID) {
		addUpdate(`"gatewayId"`, nullableString(effectiveGatewayID))
	}
	if payload.CredentialSecretID.Present {
		secretID := normalizeOptionalStringPtrValue(payload.CredentialSecretID.Value)
		if secretID == nil {
			addUpdate(`"credentialSecretId"`, nil)
		} else {
			if err := s.validateCredentialSecretReference(ctx, claims.UserID, claims.TenantID, *secretID, effectiveType); err != nil {
				return connectionResponse{}, err
			}
			addUpdate(`"credentialSecretId"`, *secretID)
			addUpdate(`"externalVaultProviderId"`, nil)
			addUpdate(`"externalVaultPath"`, nil)
			addUpdate(`"encryptedUsername"`, nil)
			addUpdate(`"usernameIV"`, nil)
			addUpdate(`"usernameTag"`, nil)
			addUpdate(`"encryptedPassword"`, nil)
			addUpdate(`"passwordIV"`, nil)
			addUpdate(`"passwordTag"`, nil)
			addUpdate(`"encryptedDomain"`, nil)
			addUpdate(`"domainIV"`, nil)
			addUpdate(`"domainTag"`, nil)
		}
	}
	if payload.ExternalVaultProviderID.Present {
		providerID := normalizeOptionalStringPtrValue(payload.ExternalVaultProviderID.Value)
		pathValue := normalizeOptionalStringPtrValue(payload.ExternalVaultPath.Value)
		addUpdate(`"externalVaultProviderId"`, nullableString(providerID))
		addUpdate(`"externalVaultPath"`, nullableString(pathValue))
		if providerID != nil {
			addUpdate(`"credentialSecretId"`, nil)
			addUpdate(`"encryptedUsername"`, nil)
			addUpdate(`"usernameIV"`, nil)
			addUpdate(`"usernameTag"`, nil)
			addUpdate(`"encryptedPassword"`, nil)
			addUpdate(`"passwordIV"`, nil)
			addUpdate(`"passwordTag"`, nil)
			addUpdate(`"encryptedDomain"`, nil)
			addUpdate(`"domainIV"`, nil)
			addUpdate(`"domainTag"`, nil)
		}
	}

	needsVaultKey := payload.Username.Present || payload.Password.Present || payload.Domain.Present
	var key []byte
	if needsVaultKey {
		key, err = s.resolveConnectionEncryptionKey(ctx, claims.UserID, access.Connection.TeamID)
		if err != nil {
			return connectionResponse{}, err
		}
		defer zeroBytes(key)
	}

	if payload.Username.Present {
		if payload.Username.Value == nil || strings.TrimSpace(*payload.Username.Value) == "" {
			return connectionResponse{}, &requestError{status: 400, message: "username must be a non-empty string"}
		}
		encrypted, err := encryptValue(key, strings.TrimSpace(*payload.Username.Value))
		if err != nil {
			return connectionResponse{}, err
		}
		addUpdate(`"encryptedUsername"`, encrypted.Ciphertext)
		addUpdate(`"usernameIV"`, encrypted.IV)
		addUpdate(`"usernameTag"`, encrypted.Tag)
	}
	if payload.Password.Present {
		if payload.Password.Value == nil || *payload.Password.Value == "" {
			return connectionResponse{}, &requestError{status: 400, message: "password must be a non-empty string"}
		}
		encrypted, err := encryptValue(key, *payload.Password.Value)
		if err != nil {
			return connectionResponse{}, err
		}
		addUpdate(`"encryptedPassword"`, encrypted.Ciphertext)
		addUpdate(`"passwordIV"`, encrypted.IV)
		addUpdate(`"passwordTag"`, encrypted.Tag)
	}
	if payload.Domain.Present {
		if payload.Domain.Value == nil || strings.TrimSpace(*payload.Domain.Value) == "" {
			addUpdate(`"encryptedDomain"`, nil)
			addUpdate(`"domainIV"`, nil)
			addUpdate(`"domainTag"`, nil)
		} else {
			encrypted, err := encryptValue(key, strings.TrimSpace(*payload.Domain.Value))
			if err != nil {
				return connectionResponse{}, err
			}
			addUpdate(`"encryptedDomain"`, encrypted.Ciphertext)
			addUpdate(`"domainIV"`, encrypted.IV)
			addUpdate(`"domainTag"`, encrypted.Tag)
		}
	}

	if len(updates) == 0 {
		return access.Connection, nil
	}

	addUpdate(`"updatedAt"`, time.Now())
	args = append(args, connectionID)
	query := fmt.Sprintf(`UPDATE "Connection" SET %s WHERE id = $%d`, strings.Join(updates, ", "), len(args))
	command, err := s.DB.Exec(ctx, query, args...)
	if err != nil {
		return connectionResponse{}, fmt.Errorf("update connection: %w", err)
	}
	if command.RowsAffected() == 0 {
		return connectionResponse{}, pgx.ErrNoRows
	}

	updated, err := s.GetConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return connectionResponse{}, err
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "UPDATE_CONNECTION", connectionID, map[string]any{
		"fields": presentUpdateFields(payload),
	}, ip)
	return updated, nil
}

func gatewayRoutingMandatoryEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GATEWAY_ROUTING_MODE")), "gateway-mandatory")
}

func connectionTypeRequiresGateway(connType string) bool {
	switch strings.ToUpper(strings.TrimSpace(connType)) {
	case "SSH", "RDP", "VNC", "DATABASE", "DB_TUNNEL":
		return true
	default:
		return false
	}
}

func compatibleGatewayTypes(connType string) []string {
	switch strings.ToUpper(strings.TrimSpace(connType)) {
	case "SSH":
		return []string{"MANAGED_SSH", "SSH_BASTION"}
	case "RDP", "VNC":
		return []string{"GUACD"}
	case "DATABASE", "DB_TUNNEL":
		return []string{"DB_PROXY"}
	default:
		return nil
	}
}

func friendlyGatewayRequirement(connType string) string {
	switch strings.ToUpper(strings.TrimSpace(connType)) {
	case "SSH":
		return "SSH_BASTION or MANAGED_SSH"
	case "RDP", "VNC":
		return "GUACD"
	case "DATABASE", "DB_TUNNEL":
		return "DB_PROXY"
	default:
		return "a compatible gateway"
	}
}

func gatewayTypePriority(gatewayType string) int {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH":
		return 0
	case "SSH_BASTION":
		return 1
	case "GUACD", "DB_PROXY":
		return 0
	default:
		return 99
	}
}

func optionalStringPointersEqual(left, right *string) bool {
	leftValue := normalizeOptionalStringPtrValue(left)
	rightValue := normalizeOptionalStringPtrValue(right)
	switch {
	case leftValue == nil && rightValue == nil:
		return true
	case leftValue == nil || rightValue == nil:
		return false
	default:
		return *leftValue == *rightValue
	}
}

func (s Service) resolveDefaultGatewayID(ctx context.Context, tenantID, connType string) (*string, error) {
	gatewayTypes := compatibleGatewayTypes(connType)
	if len(gatewayTypes) == 0 {
		return nil, nil
	}

	rows, err := s.DB.Query(ctx, `
SELECT id, type::text, "isDefault"
FROM "Gateway"
WHERE "tenantId" = $1
  AND type::text = ANY($2)
ORDER BY "isDefault" DESC, "updatedAt" DESC
`, tenantID, gatewayTypes)
	if err != nil {
		return nil, fmt.Errorf("list compatible gateways: %w", err)
	}
	defer rows.Close()

	type gatewayCandidate struct {
		ID        string
		Type      string
		IsDefault bool
	}

	candidates := make([]gatewayCandidate, 0, 4)
	for rows.Next() {
		var item gatewayCandidate
		if err := rows.Scan(&item.ID, &item.Type, &item.IsDefault); err != nil {
			return nil, fmt.Errorf("scan compatible gateway: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate compatible gateways: %w", err)
	}
	if len(candidates) == 0 {
		return nil, &requestError{
			status:  400,
			message: fmt.Sprintf("This connection type requires a %s gateway, but none is configured for this tenant.", friendlyGatewayRequirement(connType)),
		}
	}

	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.IsDefault != selected.IsDefault {
			continue
		}
		if gatewayTypePriority(candidate.Type) < gatewayTypePriority(selected.Type) {
			selected = candidate
		}
	}
	if selected.IsDefault || len(candidates) == 1 {
		return &selected.ID, nil
	}

	return nil, &requestError{
		status:  400,
		message: fmt.Sprintf("Multiple compatible gateways exist for %s connections. Set gatewayId explicitly or mark one compatible gateway as default.", strings.ToUpper(strings.TrimSpace(connType))),
	}
}

func (s Service) validateGatewayForConnectionType(ctx context.Context, tenantID, gatewayID, connType string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		if gatewayRoutingMandatoryEnabled() && connectionTypeRequiresGateway(connType) {
			return &requestError{
				status:  400,
				message: fmt.Sprintf("This connection type requires a %s gateway.", friendlyGatewayRequirement(connType)),
			}
		}
		return nil
	}

	var gatewayType string
	if err := s.DB.QueryRow(ctx, `
SELECT type::text
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
`, gatewayID, tenantID).Scan(&gatewayType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: 400, message: "gatewayId is invalid for this tenant"}
		}
		return fmt.Errorf("load gateway for validation: %w", err)
	}

	for _, candidate := range compatibleGatewayTypes(connType) {
		if strings.EqualFold(candidate, gatewayType) {
			return nil
		}
	}

	return &requestError{
		status:  400,
		message: fmt.Sprintf("Connection gateway must be of type %s for %s connections.", friendlyGatewayRequirement(connType), strings.ToUpper(strings.TrimSpace(connType))),
	}
}

func (s Service) DeleteConnection(ctx context.Context, claims authn.Claims, connectionID string, ip *string) error {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return err
	}
	if access.AccessType == "shared" {
		return pgx.ErrNoRows
	}
	if access.AccessType == "team" && (access.Connection.TeamRole == nil || !canManageTeam(*access.Connection.TeamRole)) {
		return pgx.ErrNoRows
	}
	command, err := s.DB.Exec(ctx, `DELETE FROM "Connection" WHERE id = $1`, connectionID)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DELETE_CONNECTION", connectionID, nil, ip)
	return nil
}

func (s Service) ToggleFavorite(ctx context.Context, claims authn.Claims, connectionID string, ip *string) (map[string]any, error) {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return nil, err
	}
	switch access.AccessType {
	case "shared":
		return nil, &requestError{status: 403, message: "Cannot favorite shared connections"}
	case "team":
		if access.Connection.TeamRole == nil || !canManageTeam(*access.Connection.TeamRole) {
			return nil, &requestError{status: 403, message: "Viewers cannot toggle favorites on team connections"}
		}
	}

	var isFavorite bool
	if err := s.DB.QueryRow(ctx, `UPDATE "Connection" SET "isFavorite" = NOT "isFavorite" WHERE id = $1 RETURNING "isFavorite"`, connectionID).Scan(&isFavorite); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("toggle favorite: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "CONNECTION_FAVORITE", connectionID, map[string]any{"isFavorite": isFavorite}, ip)
	return map[string]any{"id": connectionID, "isFavorite": isFavorite}, nil
}
