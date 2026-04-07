package gateways

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
)

func (s Service) ListGatewayTemplates(ctx context.Context, tenantID string) ([]gatewayTemplateResponse, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}
	rows, err := s.DB.Query(ctx, gatewayTemplateSelect+`
WHERE t."tenantId" = $1
ORDER BY t.name ASC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list gateway templates: %w", err)
	}
	defer rows.Close()

	result := make([]gatewayTemplateResponse, 0)
	for rows.Next() {
		record, err := scanGatewayTemplate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, gatewayTemplateRecordToResponse(record))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gateway templates: %w", err)
	}
	return result, nil
}

func (s Service) CreateGatewayTemplate(ctx context.Context, claims authn.Claims, input createTemplatePayload, ipAddress string) (gatewayTemplateResponse, error) {
	if s.DB == nil {
		return gatewayTemplateResponse{}, fmt.Errorf("database is unavailable")
	}
	normalized, err := normalizeCreateTemplatePayload(input)
	if err != nil {
		return gatewayTemplateResponse{}, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("begin gateway template create transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id := uuid.NewString()
	if _, err := tx.Exec(ctx, insertGatewayTemplateSQL, id, normalized.Name, normalized.Type, normalized.Host, normalized.Port, normalized.DeploymentMode, trimStringPtr(normalized.Description), normalized.APIPort, normalized.AutoScale, normalized.MinReplicas, normalized.MaxReplicas, normalized.SessionsPerInstance, normalized.ScaleDownCooldownSeconds, normalized.MonitoringEnabled, normalized.MonitorIntervalMS, normalized.InactivityTimeoutSeconds, normalized.PublishPorts, normalized.LBStrategy, claims.TenantID, claims.UserID); err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("insert gateway template: %w", err)
	}

	if err := s.insertTemplateAuditLogTx(ctx, tx, claims.UserID, "GATEWAY_TEMPLATE_CREATE", id, map[string]any{
		"name": normalized.Name,
		"type": normalized.Type,
	}, ipAddress); err != nil {
		return gatewayTemplateResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("commit gateway template create transaction: %w", err)
	}

	record, err := s.loadGatewayTemplate(ctx, claims.TenantID, id)
	if err != nil {
		return gatewayTemplateResponse{}, err
	}
	return gatewayTemplateRecordToResponse(record), nil
}

func (s Service) UpdateGatewayTemplate(ctx context.Context, claims authn.Claims, templateID string, input updateTemplatePayload, ipAddress string) (gatewayTemplateResponse, error) {
	if s.DB == nil {
		return gatewayTemplateResponse{}, fmt.Errorf("database is unavailable")
	}
	record, err := s.loadGatewayTemplate(ctx, claims.TenantID, templateID)
	if err != nil {
		return gatewayTemplateResponse{}, err
	}
	if err := validateUpdateTemplatePayload(input); err != nil {
		return gatewayTemplateResponse{}, err
	}

	updatedName := strings.TrimSpace(chooseString(record.Name, input.Name))
	updatedType := chooseString(record.Type, input.Type)
	if input.Type.Present && input.Type.Value != nil {
		updatedType = strings.ToUpper(strings.TrimSpace(*input.Type.Value))
	}
	updatedHostInput := chooseString(record.Host, input.Host)
	deploymentModeInput := record.DeploymentMode
	if input.DeploymentMode.Present && input.DeploymentMode.Value != nil {
		deploymentModeInput = *input.DeploymentMode.Value
	}
	updatedDeploymentMode, err := normalizeDeploymentMode(&deploymentModeInput, updatedType, updatedHostInput)
	if err != nil {
		return gatewayTemplateResponse{}, err
	}
	updatedHost := normalizeGatewayHostForMode(updatedDeploymentMode, updatedHostInput)
	updatedPort := chooseInt(record.Port, input.Port)
	updatedDescription := chooseNullableString(record.Description, input.Description)
	updatedAPIPort := chooseNullableInt(record.APIPort, input.APIPort)
	updatedAutoScale := chooseBool(record.AutoScale, input.AutoScale)
	updatedMinReplicas := chooseInt(record.MinReplicas, input.MinReplicas)
	updatedMaxReplicas := chooseInt(record.MaxReplicas, input.MaxReplicas)
	updatedSessionsPerInstance := chooseInt(record.SessionsPerInstance, input.SessionsPerInstance)
	updatedScaleDownCooldown := chooseInt(record.ScaleDownCooldownSeconds, input.ScaleDownCooldownSeconds)
	updatedMonitoringEnabled := chooseBool(record.MonitoringEnabled, input.MonitoringEnabled)
	updatedMonitorInterval := chooseInt(record.MonitorIntervalMS, input.MonitorIntervalMS)
	updatedInactivityTimeout := chooseInt(record.InactivityTimeoutSeconds, input.InactivityTimeoutSeconds)
	updatedPublishPorts := chooseBool(record.PublishPorts, input.PublishPorts)
	updatedLBStrategy := chooseString(record.LBStrategy, input.LBStrategy)
	if normalized := normalizeLBStrategyPtr(input.LBStrategy.Value); normalized != nil {
		updatedLBStrategy = *normalized
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("begin gateway template update transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
UPDATE "GatewayTemplate"
   SET name = $2,
       type = $3::"GatewayType",
       host = $4,
       port = $5,
       "deploymentMode" = $6::"GatewayDeploymentMode",
       description = $7,
       "apiPort" = $8,
       "autoScale" = $9,
       "minReplicas" = $10,
       "maxReplicas" = $11,
       "sessionsPerInstance" = $12,
       "scaleDownCooldownSeconds" = $13,
       "monitoringEnabled" = $14,
       "monitorIntervalMs" = $15,
       "inactivityTimeoutSeconds" = $16,
       "publishPorts" = $17,
       "lbStrategy" = $18::"LoadBalancingStrategy",
       "updatedAt" = NOW()
 WHERE id = $1
`, templateID, updatedName, updatedType, updatedHost, updatedPort, updatedDeploymentMode, updatedDescription, updatedAPIPort, updatedAutoScale, updatedMinReplicas, updatedMaxReplicas, updatedSessionsPerInstance, updatedScaleDownCooldown, updatedMonitoringEnabled, updatedMonitorInterval, updatedInactivityTimeout, updatedPublishPorts, updatedLBStrategy); err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("update gateway template: %w", err)
	}

	if err := s.insertTemplateAuditLogTx(ctx, tx, claims.UserID, "GATEWAY_TEMPLATE_UPDATE", templateID, changedTemplateDetails(input), ipAddress); err != nil {
		return gatewayTemplateResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gatewayTemplateResponse{}, fmt.Errorf("commit gateway template update transaction: %w", err)
	}

	updated, err := s.loadGatewayTemplate(ctx, claims.TenantID, templateID)
	if err != nil {
		return gatewayTemplateResponse{}, err
	}
	return gatewayTemplateRecordToResponse(updated), nil
}

func (s Service) DeleteGatewayTemplate(ctx context.Context, claims authn.Claims, templateID, ipAddress string) (map[string]any, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}
	record, err := s.loadGatewayTemplate(ctx, claims.TenantID, templateID)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin gateway template delete transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM "GatewayTemplate" WHERE id = $1`, templateID); err != nil {
		return nil, fmt.Errorf("delete gateway template: %w", err)
	}
	if err := s.insertTemplateAuditLogTx(ctx, tx, claims.UserID, "GATEWAY_TEMPLATE_DELETE", templateID, map[string]any{"name": record.Name}, ipAddress); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit gateway template delete transaction: %w", err)
	}

	return map[string]any{"deleted": true}, nil
}

func (s Service) DeployGatewayTemplate(ctx context.Context, claims authn.Claims, templateID, ipAddress string) (gatewayResponse, error) {
	if s.DB == nil {
		return gatewayResponse{}, fmt.Errorf("database is unavailable")
	}

	template, err := s.loadGatewayTemplate(ctx, claims.TenantID, templateID)
	if err != nil {
		return gatewayResponse{}, err
	}
	if strings.EqualFold(template.Type, "MANAGED_SSH") {
		var exists bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "SshKeyPair" WHERE "tenantId" = $1)`, claims.TenantID).Scan(&exists); err != nil {
			return gatewayResponse{}, fmt.Errorf("check ssh key pair: %w", err)
		}
		if !exists {
			return gatewayResponse{}, &requestError{status: http.StatusBadRequest, message: "SSH key pair not found for this tenant. Generate one first."}
		}
	}

	id := uuid.NewString()
	name := buildTemplateDeploymentName(claims.TenantID, template.Name)
	deploymentMode := template.DeploymentMode
	if strings.TrimSpace(deploymentMode) == "" {
		deploymentMode = "SINGLE_INSTANCE"
	}
	host := normalizeGatewayHostForMode(deploymentMode, template.Host)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return gatewayResponse{}, fmt.Errorf("begin template deploy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	desiredReplicas := 0
	if deploymentModeIsGroup(deploymentMode) {
		desiredReplicas = 1
	}
	if _, err := tx.Exec(ctx, insertGatewayFromTemplateSQL, id, name, template.Type, host, template.Port, deploymentMode, trimStringPtr(template.Description), template.APIPort, deploymentModeIsGroup(deploymentMode), desiredReplicas, template.AutoScale, template.MinReplicas, template.MaxReplicas, template.SessionsPerInstance, template.ScaleDownCooldownSeconds, template.MonitoringEnabled, template.MonitorIntervalMS, template.InactivityTimeoutSeconds, template.PublishPorts, template.LBStrategy, claims.TenantID, claims.UserID, templateID); err != nil {
		return gatewayResponse{}, fmt.Errorf("insert gateway from template: %w", err)
	}

	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "GATEWAY_TEMPLATE_DEPLOY", templateID, map[string]any{
		"gatewayId":   id,
		"gatewayName": name,
	}, ipAddress); err != nil {
		return gatewayResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gatewayResponse{}, fmt.Errorf("commit template deploy transaction: %w", err)
	}

	record, err := s.loadGateway(ctx, claims.TenantID, id)
	if err != nil {
		return gatewayResponse{}, err
	}
	if deploymentModeIsGroup(deploymentMode) {
		if _, err := s.DeployGatewayInstance(ctx, claims, id); err != nil {
			return gatewayResponse{}, err
		}
	}
	record, err = s.loadGateway(ctx, claims.TenantID, id)
	if err != nil {
		return gatewayResponse{}, err
	}
	return gatewayRecordToResponse(record), nil
}
