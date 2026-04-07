package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ensureBootstrapSetup(ctx context.Context, deps *apiDependencies, options devBootstrapOptions) error {
	payload := map[string]any{
		"admin": map[string]any{
			"email":    options.adminEmail,
			"username": options.adminUsername,
			"password": options.adminPassword,
		},
		"tenant": map[string]any{
			"name": options.tenantName,
		},
		"settings": map[string]any{
			"selfSignupEnabled": false,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode setup payload: %w", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://localhost/api/setup/complete", bytes.NewReader(body))
	req.RemoteAddr = devBootstrapIP + ":0"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", devBootstrapUserAgent)

	if _, err := deps.setupService.CompleteSetup(ctx, req); err != nil && !strings.Contains(err.Error(), "Setup has already been completed") {
		return fmt.Errorf("complete setup: %w", err)
	}
	return nil
}

func lookupBootstrapUserID(ctx context.Context, deps *apiDependencies, email string) (string, error) {
	var userID string
	err := deps.db.QueryRow(ctx, `SELECT id FROM "User" WHERE email = $1`, strings.TrimSpace(strings.ToLower(email))).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	err = deps.db.QueryRow(ctx, `SELECT id FROM "User" ORDER BY "createdAt" ASC LIMIT 1`).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("resolve bootstrap user: %w", err)
	}
	return userID, nil
}

func ensureBootstrapVaultUnlocked(ctx context.Context, deps *apiDependencies, userID, password string) error {
	if deps == nil {
		return fmt.Errorf("bootstrap dependencies are unavailable")
	}

	status, err := deps.vaultService.GetStatus(ctx, userID)
	if err != nil {
		return fmt.Errorf("load bootstrap vault status: %w", err)
	}
	if status.Unlocked {
		return nil
	}

	if _, err := deps.vaultService.Unlock(ctx, userID, password, devBootstrapIP); err != nil {
		return fmt.Errorf("unlock bootstrap vault: %w", err)
	}
	return nil
}

func ensureBootstrapTenant(ctx context.Context, deps *apiDependencies, userID, tenantName string) (string, error) {
	var tenantID string
	err := deps.db.QueryRow(ctx, `SELECT id FROM "Tenant" WHERE name = $1`, strings.TrimSpace(tenantName)).Scan(&tenantID)
	if err == nil {
		return tenantID, nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return "", fmt.Errorf("resolve bootstrap tenant: %w", err)
	}
	if !deps.features.MultiTenancyEnabled {
		err = deps.db.QueryRow(ctx, `SELECT id FROM "Tenant" ORDER BY "createdAt" ASC LIMIT 1`).Scan(&tenantID)
		if err == nil {
			return tenantID, nil
		}
		if err != nil && err != pgx.ErrNoRows {
			return "", fmt.Errorf("resolve single-tenant bootstrap tenant: %w", err)
		}
	}

	created, err := deps.tenantService.CreateTenant(ctx, userID, tenantName, devBootstrapIP)
	if err != nil {
		return "", fmt.Errorf("ensure bootstrap tenant: %w", err)
	}
	return created.ID, nil
}

func ensureBootstrapMembership(ctx context.Context, deps *apiDependencies, tenantID, userID string) error {
	if _, err := deps.db.Exec(ctx, `
UPDATE "TenantMember"
SET "isActive" = false
WHERE "userId" = $1
  AND "isActive" = true
  AND "tenantId" <> $2
`, userID, tenantID); err != nil {
		return fmt.Errorf("deactivate extra memberships: %w", err)
	}

	if _, err := deps.db.Exec(ctx, `
INSERT INTO "TenantMember" (id, "tenantId", "userId", role, status, "isActive", "updatedAt")
VALUES ($1, $2, $3, 'OWNER', 'ACCEPTED', true, NOW())
ON CONFLICT ("tenantId", "userId") DO UPDATE
SET role = 'OWNER',
    status = 'ACCEPTED',
    "isActive" = true,
    "updatedAt" = NOW()
`, uuid.NewString(), tenantID, userID); err != nil {
		return fmt.Errorf("ensure bootstrap membership: %w", err)
	}
	return nil
}

func ensureBootstrapSSHKeyPair(ctx context.Context, deps *apiDependencies, tenantID, userID string) error {
	var exists bool
	if err := deps.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "SshKeyPair" WHERE "tenantId" = $1)`, tenantID).Scan(&exists); err != nil {
		return fmt.Errorf("check tenant ssh key pair: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := deps.gatewayService.GenerateSSHKeyPair(ctx, userID, tenantID, devBootstrapIP); err != nil {
		return fmt.Errorf("generate tenant ssh key pair: %w", err)
	}
	return nil
}

func ensureBootstrapOrchestratorConnection(ctx context.Context, deps *apiDependencies, options devBootstrapOptions) error {
	if deps == nil || deps.store == nil {
		return fmt.Errorf("orchestrator store is unavailable")
	}

	_, err := deps.store.UpsertConnection(ctx, contracts.OrchestratorConnection{
		Name:      options.orchestratorName,
		Kind:      options.orchestratorKind,
		Scope:     options.orchestratorScope,
		Endpoint:  options.orchestratorURL,
		Namespace: "",
		Labels: map[string]string{
			"environment": "development",
			"managedBy":   "dev-bootstrap",
		},
		Capabilities: []string{
			"workload.deploy",
			"workload.restart",
			"workload.logs.read",
			"workload.delete",
		},
	})
	if err != nil {
		return fmt.Errorf("upsert development orchestrator connection: %w", err)
	}
	return nil
}
