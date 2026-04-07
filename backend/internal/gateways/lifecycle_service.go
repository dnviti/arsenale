package gateways

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) DeployGatewayInstance(ctx context.Context, claims authn.Claims, gatewayID string) (managedGatewayInstanceResponse, error) {
	if s.DB == nil {
		return managedGatewayInstanceResponse{}, fmt.Errorf("database is unavailable")
	}

	record, err := s.loadGateway(ctx, claims.TenantID, gatewayID)
	if err != nil {
		return managedGatewayInstanceResponse{}, err
	}
	if !isManagedLifecycleGatewayType(record.Type) {
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusBadRequest, message: "Only MANAGED_SSH, GUACD, and DB_PROXY gateways can be deployed as managed containers"}
	}
	if !deploymentModeIsGroup(record.DeploymentMode) {
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusBadRequest, message: "Only MANAGED_GROUP gateways can be deployed as managed containers"}
	}
	if err := s.ensureManagedGatewayDeployable(ctx, record); err != nil {
		return managedGatewayInstanceResponse{}, err
	}

	currentCount, err := s.countManagedGatewayActiveInstances(ctx, record.ID)
	if err != nil {
		return managedGatewayInstanceResponse{}, err
	}
	if record.TunnelEnabled && currentCount >= 1 {
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusBadRequest, message: "Tunnel-enabled managed gateways currently support a single replica"}
	}

	runtimeClient, orchestratorType, err := s.managedGatewayRuntime(ctx)
	if err != nil {
		return managedGatewayInstanceResponse{}, err
	}

	return s.deployManagedGatewayInstance(ctx, record, runtimeClient, orchestratorType, claims.UserID, "", true)
}

func (s Service) UndeployGateway(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (undeployResult, error) {
	if _, err := s.ScaleGateway(ctx, claims, gatewayID, 0, ipAddress); err != nil {
		return undeployResult{}, err
	}
	return undeployResult{Undeployed: true}, nil
}

func (s Service) ScaleGateway(ctx context.Context, claims authn.Claims, gatewayID string, replicas int, ipAddress string) (scaleResult, error) {
	if s.DB == nil {
		return scaleResult{}, fmt.Errorf("database is unavailable")
	}
	if replicas < 0 || replicas > maxManagedGatewayReplicas {
		return scaleResult{}, &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Replicas must be between 0 and %d", maxManagedGatewayReplicas)}
	}

	record, err := s.loadGateway(ctx, claims.TenantID, gatewayID)
	if err != nil {
		return scaleResult{}, err
	}
	if !isManagedLifecycleGatewayType(record.Type) {
		return scaleResult{}, &requestError{status: http.StatusBadRequest, message: "Only MANAGED_SSH, GUACD, and DB_PROXY gateways can be scaled"}
	}
	if !deploymentModeIsGroup(record.DeploymentMode) {
		return scaleResult{}, &requestError{status: http.StatusBadRequest, message: "Only MANAGED_GROUP gateways can be scaled"}
	}
	if err := s.ensureManagedGatewayDeployable(ctx, record); err != nil {
		return scaleResult{}, err
	}
	if record.TunnelEnabled && replicas > 1 {
		return scaleResult{}, &requestError{status: http.StatusBadRequest, message: "Tunnel-enabled managed gateways currently support a single replica"}
	}

	currentInstances, err := s.listManagedGatewayInstancesForScale(ctx, gatewayID)
	if err != nil {
		return scaleResult{}, err
	}
	currentCount := len(currentInstances)

	var (
		runtimeClient    *dockerSocketClient
		orchestratorType string
	)
	if replicas != currentCount {
		runtimeClient, orchestratorType, err = s.managedGatewayRuntime(ctx)
		if err != nil {
			return scaleResult{}, err
		}
	}

	var (
		deployed int
		removed  int
		firstErr error
	)
	if replicas > currentCount {
		toCreate := replicas - currentCount
		for i := 0; i < toCreate; i++ {
			if _, err := s.deployManagedGatewayInstance(ctx, record, runtimeClient, orchestratorType, "", "", false); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			deployed++
		}
	} else if replicas < currentCount {
		toRemove := currentCount - replicas
		for i := 0; i < toRemove && i < len(currentInstances); i++ {
			if err := s.removeManagedGatewayInstance(ctx, runtimeClient, gatewayID, currentInstances[i]); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			removed++
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return scaleResult{}, fmt.Errorf("begin gateway scale transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
   SET "desiredReplicas" = $2,
       "lastScaleAction" = NOW(),
       "updatedAt" = NOW()
 WHERE id = $1
`, gatewayID, replicas); err != nil {
		return scaleResult{}, fmt.Errorf("update gateway scale state: %w", err)
	}

	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "GATEWAY_SCALE", gatewayID, map[string]any{
		"from":     currentCount,
		"to":       replicas,
		"deployed": deployed,
		"removed":  removed,
	}, ipAddress); err != nil {
		return scaleResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return scaleResult{}, fmt.Errorf("commit gateway scale transaction: %w", err)
	}

	if firstErr != nil && deployed == 0 && removed == 0 {
		return scaleResult{}, firstErr
	}

	return scaleResult{Deployed: deployed, Removed: removed}, nil
}

func (s Service) RestartGatewayInstance(ctx context.Context, claims authn.Claims, gatewayID, instanceID string) (restartResult, error) {
	if s.DB == nil {
		return restartResult{}, fmt.Errorf("database is unavailable")
	}
	if _, err := s.loadGateway(ctx, claims.TenantID, gatewayID); err != nil {
		return restartResult{}, err
	}
	instance, err := s.loadManagedGatewayInstance(ctx, gatewayID, instanceID)
	if err != nil {
		return restartResult{}, err
	}

	runtimeClient, _, err := s.managedGatewayRuntime(ctx)
	if err != nil {
		return restartResult{}, err
	}
	if err := runtimeClient.restartContainer(ctx, instance.ContainerID); err != nil {
		return restartResult{}, &requestError{status: http.StatusServiceUnavailable, message: fmt.Sprintf("Gateway restart failed: %v", err)}
	}

	info, err := runtimeClient.inspectContainer(ctx, instance.ContainerID)
	if err != nil {
		return restartResult{}, fmt.Errorf("inspect restarted gateway instance: %w", err)
	}
	status := inferInstanceStatus(info.Status)
	healthStatus := inferInstanceHealth(info.Status, info.Health)
	if _, err := s.DB.Exec(ctx, `
UPDATE "ManagedGatewayInstance"
   SET status = $2::"ManagedInstanceStatus",
       "healthStatus" = NULLIF($3, ''),
       "lastHealthCheck" = NOW(),
       "consecutiveFailures" = 0,
       "errorMessage" = NULL,
       "updatedAt" = NOW()
 WHERE id = $1
`, instanceID, status, healthStatus); err != nil {
		return restartResult{}, fmt.Errorf("update restarted gateway instance: %w", err)
	}
	if err := s.insertGatewayAuditLog(ctx, claims.UserID, "GATEWAY_RESTART", gatewayID, map[string]any{
		"instanceId":    instanceID,
		"containerId":   instance.ContainerID,
		"containerName": instance.ContainerName,
	}, ""); err != nil {
		return restartResult{}, err
	}

	return restartResult{Restarted: true}, nil
}

func (s Service) GetGatewayInstanceLogs(ctx context.Context, claims authn.Claims, gatewayID, instanceID string, tail int) (instanceLogsResponse, error) {
	if s.DB == nil {
		return instanceLogsResponse{}, fmt.Errorf("database is unavailable")
	}
	if _, err := s.loadGateway(ctx, claims.TenantID, gatewayID); err != nil {
		return instanceLogsResponse{}, err
	}
	instance, err := s.loadManagedGatewayInstance(ctx, gatewayID, instanceID)
	if err != nil {
		return instanceLogsResponse{}, err
	}
	runtimeClient, _, err := s.managedGatewayRuntime(ctx)
	if err != nil {
		return instanceLogsResponse{}, err
	}
	logs, err := runtimeClient.getContainerLogs(ctx, instance.ContainerID, tail)
	if err != nil {
		return instanceLogsResponse{}, &requestError{status: http.StatusServiceUnavailable, message: fmt.Sprintf("Gateway log retrieval failed: %v", err)}
	}

	return instanceLogsResponse{
		Logs:          logs,
		ContainerID:   instance.ContainerID,
		ContainerName: instance.ContainerName,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}
