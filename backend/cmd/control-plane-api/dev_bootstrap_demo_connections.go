package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func ensureDemoDatabaseConnections(ctx context.Context, deps *apiDependencies, tenantID, userID string) error {
	if deps == nil || deps.connectionService.DB == nil {
		return fmt.Errorf("connection service is unavailable")
	}

	claims := authn.Claims{
		UserID:     userID,
		TenantID:   tenantID,
		TenantRole: "OWNER",
		Type:       "access",
	}

	for _, spec := range buildDevDemoDatabaseSpecs() {
		if err := upsertDemoDatabaseConnection(ctx, deps, claims, spec); err != nil {
			return err
		}
	}
	return nil
}

func upsertDemoDatabaseConnection(ctx context.Context, deps *apiDependencies, claims authn.Claims, spec devDemoDatabaseSpec) error {
	settingsJSON, err := json.Marshal(spec.DBSettings)
	if err != nil {
		return fmt.Errorf("encode demo database settings for %s: %w", spec.Name, err)
	}

	payload := map[string]any{
		"name":        spec.Name,
		"type":        "DATABASE",
		"host":        spec.Host,
		"port":        spec.Port,
		"username":    spec.Username,
		"password":    spec.Password,
		"description": spec.Description,
		"dbSettings":  json.RawMessage(settingsJSON),
	}

	existingID, err := findOwnedDemoConnectionID(ctx, deps, claims.UserID, spec.Name)
	if err != nil {
		return err
	}

	method := http.MethodPost
	urlPath := "https://localhost/api/connections"
	expectedStatus := http.StatusCreated
	if existingID != "" {
		method = http.MethodPut
		urlPath = "https://localhost/api/connections/" + existingID
		expectedStatus = http.StatusOK
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode demo connection payload for %s: %w", spec.Name, err)
	}

	req := httptest.NewRequest(method, urlPath, bytes.NewReader(body)).WithContext(ctx)
	req.RemoteAddr = devBootstrapIP + ":0"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", devBootstrapUserAgent)
	if existingID != "" {
		req.SetPathValue("id", existingID)
	}

	recorder := httptest.NewRecorder()
	if existingID != "" {
		if err := deps.connectionService.HandleUpdate(recorder, req, claims); err != nil {
			return fmt.Errorf("update demo connection %s: %w", spec.Name, err)
		}
	} else {
		if err := deps.connectionService.HandleCreate(recorder, req, claims); err != nil {
			return fmt.Errorf("create demo connection %s: %w", spec.Name, err)
		}
	}

	if recorder.Code != expectedStatus {
		return fmt.Errorf("%s demo connection %s failed: %s", strings.ToLower(method), spec.Name, strings.TrimSpace(recorder.Body.String()))
	}
	return nil
}

func findOwnedDemoConnectionID(ctx context.Context, deps *apiDependencies, userID, name string) (string, error) {
	var connectionID string
	err := deps.db.QueryRow(ctx, `
SELECT id
FROM "Connection"
WHERE "userId" = $1
  AND type = 'DATABASE'::"ConnectionType"
  AND name = $2
ORDER BY "updatedAt" DESC
LIMIT 1
`, userID, name).Scan(&connectionID)
	if err == nil {
		return connectionID, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "no rows") {
		return "", nil
	}
	return "", fmt.Errorf("find demo connection %s: %w", name, err)
}
