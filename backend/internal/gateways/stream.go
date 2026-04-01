package gateways

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	stream "github.com/dnviti/arsenale/backend/internal/sse"
)

const gatewayStreamInterval = 5 * time.Second

type liveSnapshotResponse struct {
	Gateways       []gatewayResponse                           `json:"gateways"`
	TunnelOverview tunnelOverviewResponse                      `json:"tunnelOverview"`
	ScalingStatus  map[string]scalingStatusResponse            `json:"scalingStatus"`
	Instances      map[string][]managedGatewayInstanceResponse `json:"instances"`
}

func (s Service) HandleStream(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	watchedScaling := parseStreamIDs(r.URL.Query()["watchScaling"])
	watchedInstances := parseStreamIDs(r.URL.Query()["watchInstances"])

	snapshot, err := s.buildLiveSnapshot(r.Context(), claims.TenantID, watchedScaling, watchedInstances)
	if err != nil {
		s.writeError(w, err)
		return
	}

	sse, err := stream.Open(w)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := sse.Event("snapshot", snapshot); err != nil {
		return
	}

	ticker := time.NewTicker(gatewayStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			snapshot, err := s.buildLiveSnapshot(r.Context(), claims.TenantID, watchedScaling, watchedInstances)
			if err != nil {
				return
			}
			if err := sse.Event("snapshot", snapshot); err != nil {
				return
			}
		}
	}
}

func (s Service) HandleStreamInstanceLogs(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}

	tail := parseGatewayLogTail(r.URL.Query().Get("tail"))
	logs, err := s.GetGatewayInstanceLogs(r.Context(), claims, r.PathValue("id"), r.PathValue("instanceId"), tail)
	if err != nil {
		s.writeError(w, err)
		return
	}

	sse, err := stream.Open(w)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := sse.Event("snapshot", logs); err != nil {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			logs, err := s.GetGatewayInstanceLogs(r.Context(), claims, r.PathValue("id"), r.PathValue("instanceId"), tail)
			if err != nil {
				return
			}
			if err := sse.Event("snapshot", logs); err != nil {
				return
			}
		}
	}
}

func (s Service) buildLiveSnapshot(ctx context.Context, tenantID string, watchedScaling, watchedInstances []string) (liveSnapshotResponse, error) {
	gateways, err := s.ListGateways(ctx, tenantID)
	if err != nil {
		return liveSnapshotResponse{}, err
	}
	tunnelOverview, err := s.GetTunnelOverview(ctx, tenantID)
	if err != nil {
		return liveSnapshotResponse{}, err
	}

	scalingStatus := make(map[string]scalingStatusResponse, len(watchedScaling))
	for _, gatewayID := range watchedScaling {
		status, err := s.GetScalingStatus(ctx, tenantID, gatewayID)
		if err != nil {
			if isGatewayNotFound(err) {
				continue
			}
			return liveSnapshotResponse{}, err
		}
		scalingStatus[gatewayID] = status
	}

	instances := make(map[string][]managedGatewayInstanceResponse, len(watchedInstances))
	for _, gatewayID := range watchedInstances {
		items, err := s.ListGatewayInstances(ctx, tenantID, gatewayID)
		if err != nil {
			if isGatewayNotFound(err) {
				continue
			}
			return liveSnapshotResponse{}, err
		}
		instances[gatewayID] = items
	}

	return liveSnapshotResponse{
		Gateways:       gateways,
		TunnelOverview: tunnelOverview,
		ScalingStatus:  scalingStatus,
		Instances:      instances,
	}, nil
}

func parseStreamIDs(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			id := strings.TrimSpace(part)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func isGatewayNotFound(err error) bool {
	reqErr, ok := err.(*requestError)
	return ok && reqErr.status == http.StatusNotFound
}
