package gateways

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) loadGatewayTemplate(ctx context.Context, tenantID, templateID string) (gatewayTemplateRecord, error) {
	row := s.DB.QueryRow(ctx, gatewayTemplateSelect+`
WHERE t."tenantId" = $1 AND t.id = $2
`, tenantID, templateID)
	record, err := scanGatewayTemplate(row)
	if err != nil {
		return gatewayTemplateRecord{}, err
	}
	return record, nil
}

func scanGatewayTemplate(row rowScanner) (gatewayTemplateRecord, error) {
	var (
		item        gatewayTemplateRecord
		description sql.NullString
		apiPort     sql.NullInt32
	)
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.Host,
		&item.Port,
		&item.DeploymentMode,
		&description,
		&apiPort,
		&item.AutoScale,
		&item.MinReplicas,
		&item.MaxReplicas,
		&item.SessionsPerInstance,
		&item.ScaleDownCooldownSeconds,
		&item.MonitoringEnabled,
		&item.MonitorIntervalMS,
		&item.InactivityTimeoutSeconds,
		&item.PublishPorts,
		&item.LBStrategy,
		&item.TenantID,
		&item.CreatedByID,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.GatewayCount,
	); err != nil {
		if err == pgx.ErrNoRows {
			return gatewayTemplateRecord{}, &requestError{status: http.StatusNotFound, message: "Gateway template not found"}
		}
		return gatewayTemplateRecord{}, fmt.Errorf("scan gateway template: %w", err)
	}
	item.Description = nullStringPtr(description)
	item.APIPort = nullIntPtr(apiPort)
	if strings.TrimSpace(item.DeploymentMode) == "" {
		mode, err := normalizeDeploymentMode(nil, item.Type, item.Host)
		if err == nil {
			item.DeploymentMode = mode
		}
	}
	return item, nil
}

func gatewayTemplateRecordToResponse(item gatewayTemplateRecord) gatewayTemplateResponse {
	deploymentMode := item.DeploymentMode
	if strings.TrimSpace(deploymentMode) == "" {
		mode, err := normalizeDeploymentMode(nil, item.Type, item.Host)
		if err == nil {
			deploymentMode = mode
		}
	}
	return gatewayTemplateResponse{
		ID:                       item.ID,
		Name:                     item.Name,
		Type:                     item.Type,
		Host:                     item.Host,
		Port:                     item.Port,
		DeploymentMode:           deploymentMode,
		Description:              item.Description,
		APIPort:                  item.APIPort,
		AutoScale:                item.AutoScale,
		MinReplicas:              item.MinReplicas,
		MaxReplicas:              item.MaxReplicas,
		SessionsPerInstance:      item.SessionsPerInstance,
		ScaleDownCooldownSeconds: item.ScaleDownCooldownSeconds,
		MonitoringEnabled:        item.MonitoringEnabled,
		MonitorIntervalMS:        item.MonitorIntervalMS,
		InactivityTimeoutSeconds: item.InactivityTimeoutSeconds,
		PublishPorts:             item.PublishPorts,
		LBStrategy:               item.LBStrategy,
		TenantID:                 item.TenantID,
		CreatedByID:              item.CreatedByID,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
		Count:                    gatewayTemplateCount{Gateways: item.GatewayCount},
	}
}

func (s Service) insertTemplateAuditLogTx(ctx context.Context, tx pgx.Tx, userID, action, targetID string, details map[string]any, ipAddress string) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal gateway template audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt")
VALUES ($1, $2, $3::"AuditAction", 'GatewayTemplate', $4, $5::jsonb, NULLIF($6, ''), NOW())
`, uuid.NewString(), userID, action, targetID, string(payload), ipAddress)
	if err != nil {
		return fmt.Errorf("insert gateway template audit log: %w", err)
	}
	return nil
}
