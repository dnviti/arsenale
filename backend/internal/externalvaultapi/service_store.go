package externalvaultapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) createProvider(ctx context.Context, claims authn.Claims, payload providerCreatePayload) (providerResponse, error) {
	normalized, encryptedAuth, err := s.normalizeCreatePayload(payload)
	if err != nil {
		return providerResponse{}, err
	}

	record := providerRecord{}
	row := s.DB.QueryRow(ctx, `
INSERT INTO "ExternalVaultProvider" (
	id,
	"tenantId",
	name,
	"providerType",
	"serverUrl",
	"authMethod",
	namespace,
	"mountPath",
	"encryptedAuthPayload",
	"authPayloadIV",
	"authPayloadTag",
	"caCertificate",
	"cacheTtlSeconds",
	enabled,
	"createdAt",
	"updatedAt"
) VALUES (
	$1,$2,$3,$4::"ExternalVaultType",$5,$6::"ExternalVaultAuthMethod",$7,$8,$9,$10,$11,$12,$13,true,NOW(),NOW()
)
RETURNING id,name,"providerType"::text,"serverUrl","authMethod"::text,namespace,"mountPath","encryptedAuthPayload","authPayloadIV","authPayloadTag","caCertificate","cacheTtlSeconds",enabled,"createdAt","updatedAt"
`, uuid.NewString(), claims.TenantID, normalized.Name, normalized.ProviderType, normalized.ServerURL, normalized.AuthMethod, normalized.Namespace, normalized.MountPath, encryptedAuth.Ciphertext, encryptedAuth.IV, encryptedAuth.Tag, normalized.CACertificate, normalized.CacheTTLSeconds)
	if err := scanProvider(row, &record); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return providerResponse{}, &requestError{status: http.StatusConflict, message: "A vault provider with this name already exists"}
		}
		return providerResponse{}, fmt.Errorf("create external vault provider: %w", err)
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "VAULT_PROVIDER_CREATE", record.ID, map[string]any{
		"name":         record.Name,
		"providerType": record.ProviderType,
		"serverUrl":    record.ServerURL,
		"authMethod":   record.AuthMethod,
	}); err != nil {
		return providerResponse{}, fmt.Errorf("insert create audit log: %w", err)
	}

	return toProviderResponse(record, false), nil
}

func (s Service) updateProvider(ctx context.Context, claims authn.Claims, providerID string, payload providerUpdatePayload) (providerResponse, error) {
	existing, err := s.getProvider(ctx, claims.TenantID, providerID)
	if err != nil {
		return providerResponse{}, err
	}

	normalized, encryptedAuth, changedFields, err := s.normalizeUpdatePayload(existing, payload)
	if err != nil {
		return providerResponse{}, err
	}
	if len(changedFields) == 0 {
		return toProviderResponse(existing, false), nil
	}

	record := providerRecord{}
	row := s.DB.QueryRow(ctx, `
UPDATE "ExternalVaultProvider"
SET
	name = $3,
	"providerType" = $4::"ExternalVaultType",
	"serverUrl" = $5,
	"authMethod" = $6::"ExternalVaultAuthMethod",
	namespace = $7,
	"mountPath" = $8,
	"encryptedAuthPayload" = $9,
	"authPayloadIV" = $10,
	"authPayloadTag" = $11,
	"caCertificate" = $12,
	"cacheTtlSeconds" = $13,
	enabled = $14,
	"updatedAt" = NOW()
WHERE id = $1 AND "tenantId" = $2
RETURNING id,name,"providerType"::text,"serverUrl","authMethod"::text,namespace,"mountPath","encryptedAuthPayload","authPayloadIV","authPayloadTag","caCertificate","cacheTtlSeconds",enabled,"createdAt","updatedAt"
`, providerID, claims.TenantID, normalized.Name, normalized.ProviderType, normalized.ServerURL, normalized.AuthMethod, normalized.Namespace, normalized.MountPath, encryptedAuth.Ciphertext, encryptedAuth.IV, encryptedAuth.Tag, normalized.CACertificate, normalized.CacheTTLSeconds, normalized.Enabled)
	if err := scanProvider(row, &record); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return providerResponse{}, &requestError{status: http.StatusNotFound, message: "Vault provider not found"}
		}
		return providerResponse{}, fmt.Errorf("update external vault provider: %w", err)
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "VAULT_PROVIDER_UPDATE", record.ID, map[string]any{
		"changes": changedFields,
	}); err != nil {
		return providerResponse{}, fmt.Errorf("insert update audit log: %w", err)
	}

	return toProviderResponse(record, false), nil
}

func (s Service) deleteProvider(ctx context.Context, claims authn.Claims, providerID string) error {
	record, err := s.getProvider(ctx, claims.TenantID, providerID)
	if err != nil {
		return err
	}

	if _, err := s.DB.Exec(ctx, `
UPDATE "Connection"
SET "externalVaultProviderId" = NULL, "externalVaultPath" = NULL
WHERE "externalVaultProviderId" = $1
`, providerID); err != nil {
		return fmt.Errorf("clear provider connections: %w", err)
	}

	if _, err := s.DB.Exec(ctx, `DELETE FROM "ExternalVaultProvider" WHERE id = $1 AND "tenantId" = $2`, providerID, claims.TenantID); err != nil {
		return fmt.Errorf("delete external vault provider: %w", err)
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "VAULT_PROVIDER_DELETE", providerID, map[string]any{
		"name": record.Name,
	}); err != nil {
		return fmt.Errorf("insert delete audit log: %w", err)
	}
	return nil
}

func (s Service) listProviders(ctx context.Context, tenantID string) ([]providerResponse, error) {
	rows, err := s.DB.Query(ctx, `
SELECT id,name,"providerType"::text,"serverUrl","authMethod"::text,namespace,"mountPath","encryptedAuthPayload","authPayloadIV","authPayloadTag","caCertificate","cacheTtlSeconds",enabled,"createdAt","updatedAt"
FROM "ExternalVaultProvider"
WHERE "tenantId" = $1
ORDER BY name ASC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list external vault providers: %w", err)
	}
	defer rows.Close()

	items := make([]providerResponse, 0)
	for rows.Next() {
		var record providerRecord
		if err := scanProvider(rows, &record); err != nil {
			return nil, fmt.Errorf("scan external vault provider: %w", err)
		}
		items = append(items, toProviderResponse(record, false))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate external vault providers: %w", err)
	}
	return items, nil
}

func (s Service) getProvider(ctx context.Context, tenantID, providerID string) (providerRecord, error) {
	row := s.DB.QueryRow(ctx, `
SELECT id,name,"providerType"::text,"serverUrl","authMethod"::text,namespace,"mountPath","encryptedAuthPayload","authPayloadIV","authPayloadTag","caCertificate","cacheTtlSeconds",enabled,"createdAt","updatedAt"
FROM "ExternalVaultProvider"
WHERE id = $1 AND "tenantId" = $2
`, providerID, tenantID)

	var record providerRecord
	if err := scanProvider(row, &record); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return providerRecord{}, &requestError{status: http.StatusNotFound, message: "Vault provider not found"}
		}
		return providerRecord{}, fmt.Errorf("get external vault provider: %w", err)
	}
	return record, nil
}

func scanProvider(row interface{ Scan(...any) error }, dest *providerRecord) error {
	return row.Scan(
		&dest.ID,
		&dest.Name,
		&dest.ProviderType,
		&dest.ServerURL,
		&dest.AuthMethod,
		&dest.Namespace,
		&dest.MountPath,
		&dest.EncryptedAuthPayload,
		&dest.AuthPayloadIV,
		&dest.AuthPayloadTag,
		&dest.CACertificate,
		&dest.CacheTTLSeconds,
		&dest.Enabled,
		&dest.CreatedAt,
		&dest.UpdatedAt,
	)
}

func toProviderResponse(record providerRecord, includeCA bool) providerResponse {
	resp := providerResponse{
		ID:              record.ID,
		Name:            record.Name,
		ProviderType:    record.ProviderType,
		ServerURL:       record.ServerURL,
		AuthMethod:      record.AuthMethod,
		Namespace:       record.Namespace,
		MountPath:       record.MountPath,
		CacheTTLSeconds: record.CacheTTLSeconds,
		Enabled:         record.Enabled,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		HasAuthPayload:  strings.TrimSpace(record.EncryptedAuthPayload) != "",
	}
	if includeCA {
		resp.CACertificate = record.CACertificate
	}
	return resp
}
