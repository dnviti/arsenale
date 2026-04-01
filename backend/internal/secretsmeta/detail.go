package secretsmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
)

type secretVersionDataResponse struct {
	Data json.RawMessage `json:"data"`
}

func (s Service) HandleGet(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	item, err := s.LoadSecret(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, item)
}

func (s Service) HandleListVersions(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	items, err := s.LoadVersions(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, items)
}

func (s Service) HandleGetVersionData(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		app.ErrorJSON(w, http.StatusBadRequest, "Invalid version number")
		return
	}

	data, err := s.LoadVersionData(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), version)
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, secretVersionDataResponse{Data: data})
}

func (s Service) LoadSecret(ctx context.Context, userID, tenantID, secretID string) (credentialresolver.SecretDetail, error) {
	return s.resolver().ResolveSecret(ctx, userID, secretID, tenantID)
}

func (s Service) LoadVersions(ctx context.Context, userID, tenantID, secretID string) ([]credentialresolver.SecretVersion, error) {
	return s.resolver().ListSecretVersions(ctx, userID, secretID, tenantID)
}

func (s Service) LoadVersionData(ctx context.Context, userID, tenantID, secretID string, version int) (json.RawMessage, error) {
	return s.resolver().ResolveSecretVersionData(ctx, userID, secretID, tenantID, version)
}

func (s Service) resolver() credentialresolver.Resolver {
	return credentialresolver.Resolver{
		DB:        s.DB,
		Redis:     s.Redis,
		ServerKey: s.ServerKey,
		VaultTTL:  s.VaultTTL,
	}
}

func (s Service) handleResolverError(w http.ResponseWriter, err error) {
	var reqErr *credentialresolver.RequestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.Status, reqErr.Message)
		return
	}

	app.ErrorJSON(w, http.StatusServiceUnavailable, serviceError(err))
}

func serviceError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
