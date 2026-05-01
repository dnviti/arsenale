package syncprofiles

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
)

func (s Service) validateCreatePayload(ctx context.Context, tenantID string, payload createPayload) (syncProfileConfig, *string, error) {
	if name := strings.TrimSpace(payload.Name); name == "" || len(name) > 100 {
		return syncProfileConfig{}, nil, &requestError{status: http.StatusBadRequest, message: "name must be between 1 and 100 characters"}
	}
	if strings.TrimSpace(payload.Provider) != "NETBOX" {
		return syncProfileConfig{}, nil, &requestError{status: http.StatusBadRequest, message: "provider must be NETBOX"}
	}
	if token := strings.TrimSpace(payload.APIToken); token == "" || len(token) > 500 {
		return syncProfileConfig{}, nil, &requestError{status: http.StatusBadRequest, message: "apiToken must be between 1 and 500 characters"}
	}

	config := syncProfileConfig{
		URL:              strings.TrimSpace(payload.URL),
		Filters:          cloneStringMap(payload.Filters),
		PlatformMapping:  cloneStringMap(payload.PlatformMapping),
		DefaultProtocol:  defaultStringPointer(payload.DefaultProtocol, "SSH"),
		DefaultPort:      cloneIntMap(payload.DefaultPort),
		ConflictStrategy: defaultStringPointer(payload.ConflictStrategy, "update"),
	}
	if err := validateConfig(config); err != nil {
		return syncProfileConfig{}, nil, err
	}

	teamID, err := s.normalizeTeamID(ctx, tenantID, payload.TeamID)
	if err != nil {
		return syncProfileConfig{}, nil, err
	}
	return config, teamID, nil
}

func (s Service) normalizeTeamID(ctx context.Context, tenantID string, teamID *string) (*string, error) {
	if teamID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*teamID)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil, &requestError{status: http.StatusBadRequest, message: "teamId must be a valid UUID"}
	}
	var exists bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "Team" WHERE id = $1 AND "tenantId" = $2)`, trimmed, tenantID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check team: %w", err)
	}
	if !exists {
		return nil, &requestError{status: http.StatusBadRequest, message: "teamId must belong to the current tenant"}
	}
	return &trimmed, nil
}

func (s Service) loadProfileRecord(ctx context.Context, profileID, tenantID string) (syncProfileResponse, bool, error) {
	item, err := s.GetProfile(ctx, profileID, tenantID)
	if err != nil {
		return syncProfileResponse{}, false, err
	}
	return item, item.HasAPIToken, nil
}

func validateConfig(config syncProfileConfig) error {
	if config.URL == "" || len(config.URL) > 500 {
		return &requestError{status: http.StatusBadRequest, message: "url must be a valid URL with at most 500 characters"}
	}
	if _, err := neturl.ParseRequestURI(config.URL); err != nil {
		return &requestError{status: http.StatusBadRequest, message: "url must be a valid URL with at most 500 characters"}
	}
	switch config.DefaultProtocol {
	case "SSH", "RDP", "VNC":
	default:
		return &requestError{status: http.StatusBadRequest, message: "defaultProtocol must be SSH, RDP, or VNC"}
	}
	switch config.ConflictStrategy {
	case "update", "skip", "overwrite":
	default:
		return &requestError{status: http.StatusBadRequest, message: "conflictStrategy must be update, skip, or overwrite"}
	}
	for key, value := range config.DefaultPort {
		if strings.TrimSpace(key) == "" || value < 1 || value > 65535 {
			return &requestError{status: http.StatusBadRequest, message: "defaultPort values must be integers between 1 and 65535"}
		}
	}
	return nil
}

func normalizeConfig(config *syncProfileConfig) {
	if config.Filters == nil {
		config.Filters = map[string]string{}
	}
	if config.PlatformMapping == nil {
		config.PlatformMapping = map[string]string{}
	}
	if config.DefaultPort == nil {
		config.DefaultPort = map[string]int{}
	}
	if strings.TrimSpace(config.DefaultProtocol) == "" {
		config.DefaultProtocol = "SSH"
	}
	if strings.TrimSpace(config.ConflictStrategy) == "" {
		config.ConflictStrategy = "update"
	}
}

func defaultStringPointer(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func normalizeCronExpression(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func requireTenantAdmin(claims authn.Claims) *requestError {
	if strings.TrimSpace(claims.TenantID) == "" {
		return &requestError{status: http.StatusForbidden, message: "You must belong to an organization to perform this action"}
	}
	if roleHierarchy[strings.TrimSpace(claims.TenantRole)] < roleHierarchy["ADMIN"] {
		return &requestError{status: http.StatusForbidden, message: "Insufficient tenant role"}
	}
	return nil
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
