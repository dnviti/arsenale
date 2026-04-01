package syncprofiles

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type syncTestResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type discoveredDevice struct {
	ExternalID  string         `json:"externalId"`
	Name        string         `json:"name"`
	Host        string         `json:"host"`
	Port        int            `json:"port"`
	Protocol    string         `json:"protocol"`
	SiteName    string         `json:"siteName,omitempty"`
	RackName    string         `json:"rackName,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata"`
}

type syncPlan struct {
	ToCreate []discoveredDevice   `json:"toCreate"`
	ToUpdate []syncPlanUpdateItem `json:"toUpdate"`
	ToSkip   []syncPlanSkipItem   `json:"toSkip"`
	Errors   []syncPlanErrorItem  `json:"errors"`
}

type syncPlanUpdateItem struct {
	Device       discoveredDevice `json:"device"`
	ConnectionID string           `json:"connectionId"`
	Changes      []string         `json:"changes"`
}

type syncPlanSkipItem struct {
	Device discoveredDevice `json:"device"`
	Reason string           `json:"reason"`
}

type syncPlanErrorItem struct {
	Device discoveredDevice `json:"device"`
	Error  string           `json:"error"`
}

type syncPreviewResponse struct {
	Plan syncPlan `json:"plan"`
}

type syncResultError struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Error      string `json:"error"`
}

type syncResultResponse struct {
	Created int               `json:"created"`
	Updated int               `json:"updated"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Errors  []syncResultError `json:"errors"`
}

type triggerSyncResponse struct {
	Plan   syncPlan            `json:"plan"`
	Result *syncResultResponse `json:"result,omitempty"`
}

type syncProfileRuntime struct {
	Profile           syncProfileResponse
	EncryptedAPIToken string
	APITokenIV        string
	APITokenTag       string
}

type netBoxPaginatedResponse[T any] struct {
	Count   int     `json:"count"`
	Next    *string `json:"next"`
	Results []T     `json:"results"`
}

type netBoxIP struct {
	Address string `json:"address"`
	Family  struct {
		Value int `json:"value"`
	} `json:"family"`
}

type netBoxPlatform struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type netBoxNamed struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type netBoxRack struct {
	Name string `json:"name"`
}

type netBoxStatus struct {
	Value string `json:"value"`
}

type netBoxDevice struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Display      string          `json:"display"`
	PrimaryIP4   *netBoxIP       `json:"primary_ip4"`
	PrimaryIP6   *netBoxIP       `json:"primary_ip6"`
	Platform     *netBoxPlatform `json:"platform"`
	Site         *netBoxNamed    `json:"site"`
	Rack         *netBoxRack     `json:"rack"`
	Location     *netBoxRack     `json:"location"`
	Status       *netBoxStatus   `json:"status"`
	Description  string          `json:"description"`
	CustomFields map[string]any  `json:"custom_fields"`
}

type netBoxVM struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Display      string          `json:"display"`
	PrimaryIP4   *netBoxIP       `json:"primary_ip4"`
	PrimaryIP6   *netBoxIP       `json:"primary_ip6"`
	Platform     *netBoxPlatform `json:"platform"`
	Site         *netBoxNamed    `json:"site"`
	Cluster      *netBoxRack     `json:"cluster"`
	Status       *netBoxStatus   `json:"status"`
	Description  string          `json:"description"`
	CustomFields map[string]any  `json:"custom_fields"`
}

func (s Service) HandleTestConnection(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	result, err := s.TestConnection(r.Context(), r.PathValue("id"), claims.TenantID)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "Sync profile not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleTriggerSync(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return nil
	}
	var payload struct {
		DryRun bool `json:"dryRun"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	result, err := s.TriggerSync(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), payload.DryRun)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "Sync profile not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}
	app.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (s Service) TestConnection(ctx context.Context, profileID, tenantID string) (syncTestResult, error) {
	runtime, err := s.loadProfileRuntime(ctx, profileID, tenantID)
	if err != nil {
		return syncTestResult{}, err
	}
	apiToken, err := decryptValue(s.ServerEncryptionKey, runtime.EncryptedAPIToken, runtime.APITokenIV, runtime.APITokenTag)
	if err != nil {
		return syncTestResult{OK: false, Error: "failed to decrypt sync API token"}, nil
	}
	ok, message := testNetBoxConnection(runtime.Profile.Config, apiToken)
	result := syncTestResult{OK: ok}
	if message != "" {
		result.Error = message
	}
	return result, nil
}

func (s Service) TriggerSync(ctx context.Context, userID, tenantID, profileID string, dryRun bool) (triggerSyncResponse, error) {
	runtime, err := s.loadProfileRuntime(ctx, profileID, tenantID)
	if err != nil {
		return triggerSyncResponse{}, err
	}

	apiToken, err := decryptValue(s.ServerEncryptionKey, runtime.EncryptedAPIToken, runtime.APITokenIV, runtime.APITokenTag)
	if err != nil {
		return triggerSyncResponse{}, fmt.Errorf("decrypt sync api token: %w", err)
	}

	syncLogID, err := s.beginSyncRun(ctx, userID, runtime.Profile.ID, dryRun)
	if err != nil {
		return triggerSyncResponse{}, err
	}

	plan, err := s.buildSyncPlan(ctx, runtime.Profile, apiToken)
	if err != nil {
		if dryRun {
			plan = discoveryErrorPlan(runtime.Profile, err)
			logDetails := map[string]any{
				"dryRun":   true,
				"toCreate": len(plan.ToCreate),
				"toUpdate": len(plan.ToUpdate),
				"toSkip":   len(plan.ToSkip),
				"errors":   len(plan.Errors),
			}
			profileDetails := map[string]any{
				"dryRun": true,
				"errors": []string{err.Error()},
			}
			if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, "ERROR", logDetails, profileDetails); err != nil {
				return triggerSyncResponse{}, err
			}
			return triggerSyncResponse{Plan: plan}, nil
		}
		if recordErr := s.failSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, err); recordErr != nil {
			return triggerSyncResponse{}, fmt.Errorf("sync failed: %w (recording error failed: %v)", err, recordErr)
		}
		return triggerSyncResponse{}, err
	}

	if dryRun {
		logDetails := map[string]any{
			"dryRun":   true,
			"toCreate": len(plan.ToCreate),
			"toUpdate": len(plan.ToUpdate),
			"toSkip":   len(plan.ToSkip),
			"errors":   len(plan.Errors),
		}
		if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, "SUCCESS", logDetails, map[string]any{"dryRun": true}); err != nil {
			return triggerSyncResponse{}, err
		}
		return triggerSyncResponse{Plan: plan}, nil
	}

	result := s.executeSyncPlan(ctx, plan, runtime.Profile)
	status := "SUCCESS"
	if result.Failed > 0 {
		status = "PARTIAL"
	}
	logDetails := map[string]any{
		"created": result.Created,
		"updated": result.Updated,
		"skipped": result.Skipped,
		"failed":  result.Failed,
		"errors":  result.Errors,
	}
	profileDetails := map[string]any{
		"created": result.Created,
		"updated": result.Updated,
		"skipped": result.Skipped,
		"failed":  result.Failed,
	}
	if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, status, logDetails, profileDetails); err != nil {
		return triggerSyncResponse{}, err
	}
	return triggerSyncResponse{Plan: plan, Result: &result}, nil
}

func (s Service) buildSyncPlan(ctx context.Context, profile syncProfileResponse, apiToken string) (syncPlan, error) {
	devices, err := discoverNetBoxDevices(profile.Config, apiToken)
	if err != nil {
		return syncPlan{}, err
	}
	return s.buildPlan(ctx, profile.ID, devices, profile.Config.ConflictStrategy)
}

func discoveryErrorPlan(profile syncProfileResponse, cause error) syncPlan {
	return syncPlan{
		ToCreate: []discoveredDevice{},
		ToUpdate: []syncPlanUpdateItem{},
		ToSkip:   []syncPlanSkipItem{},
		Errors: []syncPlanErrorItem{
			{
				Device: discoveredDevice{
					ExternalID: "provider:discovery",
					Name:       profile.Name,
					Protocol:   profile.Provider,
					Metadata:   map[string]any{"provider": profile.Provider},
				},
				Error: cause.Error(),
			},
		},
	}
}

func (s Service) loadProfileRuntime(ctx context.Context, profileID, tenantID string) (syncProfileRuntime, error) {
	if _, err := uuid.Parse(strings.TrimSpace(profileID)); err != nil {
		return syncProfileRuntime{}, &requestError{status: http.StatusBadRequest, message: "invalid sync profile id"}
	}
	row := s.DB.QueryRow(ctx, `
SELECT id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
       "lastSyncAt", "lastSyncStatus"::text,
       CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
       "createdById", "createdAt", "updatedAt", "encryptedApiToken",
       "apiTokenIV", "apiTokenTag"
FROM "SyncProfile"
WHERE id = $1 AND "tenantId" = $2
`, profileID, tenantID)

	var (
		profile           syncProfileResponse
		configText        string
		cronExpression    sql.NullString
		teamID            sql.NullString
		lastSyncAt        sql.NullTime
		lastSyncStatus    sql.NullString
		lastSyncDetails   sql.NullString
		encryptedAPIToken sql.NullString
		apiTokenIV        sql.NullString
		apiTokenTag       sql.NullString
	)
	if err := row.Scan(
		&profile.ID,
		&profile.Name,
		&profile.TenantID,
		&profile.Provider,
		&configText,
		&cronExpression,
		&profile.Enabled,
		&teamID,
		&lastSyncAt,
		&lastSyncStatus,
		&lastSyncDetails,
		&profile.CreatedByID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&encryptedAPIToken,
		&apiTokenIV,
		&apiTokenTag,
	); err != nil {
		return syncProfileRuntime{}, err
	}
	if err := json.Unmarshal([]byte(configText), &profile.Config); err != nil {
		return syncProfileRuntime{}, fmt.Errorf("decode sync profile config: %w", err)
	}
	normalizeConfig(&profile.Config)
	if cronExpression.Valid {
		profile.CronExpression = &cronExpression.String
	}
	if teamID.Valid {
		profile.TeamID = &teamID.String
	}
	if lastSyncAt.Valid {
		profile.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatus.Valid {
		profile.LastSyncStatus = &lastSyncStatus.String
	}
	if lastSyncDetails.Valid {
		profile.LastSyncDetails = json.RawMessage(lastSyncDetails.String)
	}
	profile.HasAPIToken = encryptedAPIToken.Valid && encryptedAPIToken.String != ""
	return syncProfileRuntime{
		Profile:           profile,
		EncryptedAPIToken: encryptedAPIToken.String,
		APITokenIV:        apiTokenIV.String,
		APITokenTag:       apiTokenTag.String,
	}, nil
}

func testNetBoxConnection(config syncProfileConfig, apiToken string) (bool, string) {
	u, err := neturl.Parse(config.URL)
	if err != nil {
		return false, err.Error()
	}
	statusURL := u.ResolveReference(&neturl.URL{Path: "/api/status/"}).String()
	req, err := http.NewRequest(http.MethodGet, statusURL, nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Token "+apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, ""
	}
	return false, fmt.Sprintf("NetBox returned HTTP %d", resp.StatusCode)
}

func discoverNetBoxDevices(config syncProfileConfig, apiToken string) ([]discoveredDevice, error) {
	devices := make([]discoveredDevice, 0)

	physicalDevices, err := fetchAllPages[netBoxDevice](config.URL, "/api/dcim/devices/", apiToken, config.Filters)
	if err != nil {
		return nil, err
	}
	for _, dev := range physicalDevices {
		ip := resolveIP(dev.PrimaryIP4, dev.PrimaryIP6)
		if ip == "" {
			continue
		}
		protocol := resolveProtocol(dev.Platform, config)
		port := resolvePort(protocol, config.DefaultPort)
		rackName := ""
		if dev.Rack != nil {
			rackName = dev.Rack.Name
		} else if dev.Location != nil {
			rackName = dev.Location.Name
		}
		siteName := ""
		if dev.Site != nil {
			siteName = dev.Site.Name
		}
		devices = append(devices, discoveredDevice{
			ExternalID:  fmt.Sprintf("device:%d", dev.ID),
			Name:        defaultDisplayName(dev.Name, dev.Display),
			Host:        ip,
			Port:        port,
			Protocol:    protocol,
			SiteName:    siteName,
			RackName:    rackName,
			Description: strings.TrimSpace(dev.Description),
			Metadata: map[string]any{
				"type":         "device",
				"netboxId":     dev.ID,
				"platform":     platformSlug(dev.Platform),
				"status":       statusValue(dev.Status),
				"customFields": dev.CustomFields,
			},
		})
	}

	vms, err := fetchAllPages[netBoxVM](config.URL, "/api/virtualization/virtual-machines/", apiToken, config.Filters)
	if err != nil {
		return nil, err
	}
	for _, vm := range vms {
		ip := resolveIP(vm.PrimaryIP4, vm.PrimaryIP6)
		if ip == "" {
			continue
		}
		protocol := resolveProtocol(vm.Platform, config)
		port := resolvePort(protocol, config.DefaultPort)
		siteName := ""
		if vm.Site != nil {
			siteName = vm.Site.Name
		}
		rackName := ""
		if vm.Cluster != nil {
			rackName = vm.Cluster.Name
		}
		devices = append(devices, discoveredDevice{
			ExternalID:  fmt.Sprintf("vm:%d", vm.ID),
			Name:        defaultDisplayName(vm.Name, vm.Display),
			Host:        ip,
			Port:        port,
			Protocol:    protocol,
			SiteName:    siteName,
			RackName:    rackName,
			Description: strings.TrimSpace(vm.Description),
			Metadata: map[string]any{
				"type":         "vm",
				"netboxId":     vm.ID,
				"platform":     platformSlug(vm.Platform),
				"status":       statusValue(vm.Status),
				"customFields": vm.CustomFields,
			},
		})
	}

	return devices, nil
}

func (s Service) buildPlan(ctx context.Context, profileID string, devices []discoveredDevice, conflictStrategy string) (syncPlan, error) {
	plan := syncPlan{
		ToCreate: []discoveredDevice{},
		ToUpdate: []syncPlanUpdateItem{},
		ToSkip:   []syncPlanSkipItem{},
		Errors:   []syncPlanErrorItem{},
	}

	rows, err := s.DB.Query(ctx, `
SELECT id, "externalId", name, host, port, type::text
FROM "Connection"
WHERE "syncProfileId" = $1
`, profileID)
	if err != nil {
		return syncPlan{}, fmt.Errorf("list existing sync connections: %w", err)
	}
	defer rows.Close()

	type existingConnection struct {
		ID         string
		ExternalID string
		Name       string
		Host       string
		Port       int
		Type       string
	}
	existing := make(map[string]existingConnection)
	for rows.Next() {
		var item existingConnection
		if err := rows.Scan(&item.ID, &item.ExternalID, &item.Name, &item.Host, &item.Port, &item.Type); err != nil {
			return syncPlan{}, fmt.Errorf("scan existing sync connection: %w", err)
		}
		if strings.TrimSpace(item.ExternalID) != "" {
			existing[item.ExternalID] = item
		}
	}
	if err := rows.Err(); err != nil {
		return syncPlan{}, fmt.Errorf("iterate existing sync connections: %w", err)
	}

	for _, device := range devices {
		if strings.TrimSpace(device.Host) == "" {
			plan.Errors = append(plan.Errors, syncPlanErrorItem{Device: device, Error: "No IP address resolved"})
			continue
		}

		current, ok := existing[device.ExternalID]
		if !ok {
			plan.ToCreate = append(plan.ToCreate, device)
			continue
		}

		if conflictStrategy == "skip" {
			plan.ToSkip = append(plan.ToSkip, syncPlanSkipItem{
				Device: device,
				Reason: "Connection already exists (skip strategy)",
			})
			continue
		}

		changes := make([]string, 0)
		if current.Name != device.Name {
			changes = append(changes, fmt.Sprintf(`name: "%s" → "%s"`, current.Name, device.Name))
		}
		if current.Host != device.Host {
			changes = append(changes, fmt.Sprintf(`host: "%s" → "%s"`, current.Host, device.Host))
		}
		if current.Port != device.Port {
			changes = append(changes, fmt.Sprintf("port: %d → %d", current.Port, device.Port))
		}
		if current.Type != device.Protocol {
			changes = append(changes, fmt.Sprintf("protocol: %s → %s", current.Type, device.Protocol))
		}

		if len(changes) == 0 {
			plan.ToSkip = append(plan.ToSkip, syncPlanSkipItem{
				Device: device,
				Reason: "No changes detected",
			})
			continue
		}
		if conflictStrategy == "update" || conflictStrategy == "overwrite" {
			plan.ToUpdate = append(plan.ToUpdate, syncPlanUpdateItem{
				Device:       device,
				ConnectionID: current.ID,
				Changes:      changes,
			})
		}
	}
	return plan, nil
}

func fetchAllPages[T any](baseURL, path, apiToken string, filters map[string]string) ([]T, error) {
	values := neturl.Values{}
	values.Set("limit", "100")
	for key, value := range filters {
		values.Set(key, value)
	}
	startURL, err := neturl.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	current := startURL.ResolveReference(&neturl.URL{Path: path, RawQuery: values.Encode()}).String()
	client := &http.Client{Timeout: 30 * time.Second}

	result := make([]T, 0)
	for current != "" {
		req, err := http.NewRequest(http.MethodGet, current, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Token "+apiToken)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var page netBoxPaginatedResponse[T]
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				io.Copy(io.Discard, resp.Body)
				err = fmt.Errorf("NetBox returned HTTP %d", resp.StatusCode)
				return
			}
			err = json.NewDecoder(resp.Body).Decode(&page)
		}()
		if err != nil {
			return nil, err
		}
		result = append(result, page.Results...)
		if page.Next != nil {
			current = *page.Next
		} else {
			current = ""
		}
	}
	return result, nil
}

func resolveIP(ip4, ip6 *netBoxIP) string {
	if ip4 != nil && strings.TrimSpace(ip4.Address) != "" {
		return stripCIDR(ip4.Address)
	}
	if ip6 != nil && strings.TrimSpace(ip6.Address) != "" {
		return stripCIDR(ip6.Address)
	}
	return ""
}

func stripCIDR(address string) string {
	if idx := strings.Index(address, "/"); idx >= 0 {
		return address[:idx]
	}
	return address
}

func resolveProtocol(platform *netBoxPlatform, config syncProfileConfig) string {
	if platform != nil {
		if mapped, ok := config.PlatformMapping[platform.Slug]; ok && strings.TrimSpace(mapped) != "" {
			return mapped
		}
	}
	return config.DefaultProtocol
}

func resolvePort(protocol string, defaults map[string]int) int {
	if defaults != nil {
		if value, ok := defaults[protocol]; ok && value > 0 {
			return value
		}
	}
	switch protocol {
	case "RDP":
		return 3389
	case "VNC":
		return 5900
	default:
		return 22
	}
}

func defaultDisplayName(name, display string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return display
}

func platformSlug(platform *netBoxPlatform) string {
	if platform == nil {
		return ""
	}
	return platform.Slug
}

func statusValue(status *netBoxStatus) string {
	if status == nil {
		return ""
	}
	return status.Value
}

func decryptValue(key []byte, ciphertextHex, ivHex, tagHex string) (string, error) {
	ciphertext, err := hex.DecodeString(strings.TrimSpace(ciphertextHex))
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	iv, err := hex.DecodeString(strings.TrimSpace(ivHex))
	if err != nil {
		return "", fmt.Errorf("decode iv: %w", err)
	}
	tag, err := hex.DecodeString(strings.TrimSpace(tagHex))
	if err != nil {
		return "", fmt.Errorf("decode tag: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, iv, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("decrypt sync token: %w", err)
	}
	return string(plaintext), nil
}
