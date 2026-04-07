package gateways

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s Service) loadManagedGatewayInstance(ctx context.Context, gatewayID, instanceID string) (managedGatewayInstanceResponse, error) {
	if s.DB == nil {
		return managedGatewayInstanceResponse{}, fmt.Errorf("database is unavailable")
	}

	row := s.DB.QueryRow(ctx, `
SELECT
	id,
	"gatewayId",
	"containerId",
	"containerName",
	host,
	port,
	"apiPort",
	status::text,
	"orchestratorType",
	"healthStatus",
	"lastHealthCheck",
	"errorMessage",
	"consecutiveFailures",
	"tunnelProxyHost",
	"tunnelProxyPort",
	"createdAt",
	"updatedAt"
FROM "ManagedGatewayInstance"
WHERE id = $1
  AND "gatewayId" = $2
`, instanceID, gatewayID)

	item, err := scanManagedGatewayInstance(row)
	if err != nil {
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusNotFound, message: "Instance not found"}
	}
	return item, nil
}

func (s Service) ensureManagedGatewayDeployable(ctx context.Context, record gatewayRecord) error {
	if !strings.EqualFold(record.Type, "MANAGED_SSH") {
		return nil
	}

	var exists bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "SshKeyPair" WHERE "tenantId" = $1)`, record.TenantID).Scan(&exists); err != nil {
		return fmt.Errorf("check ssh key pair: %w", err)
	}
	if !exists {
		return &requestError{status: http.StatusBadRequest, message: "SSH key pair not found for this tenant. Generate one first."}
	}
	return nil
}

func (s Service) countManagedGatewayInstanceRecords(ctx context.Context, gatewayID string) (int, error) {
	var count int
	if err := s.DB.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "ManagedGatewayInstance"
WHERE "gatewayId" = $1
`, gatewayID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count managed gateway instances: %w", err)
	}
	return count, nil
}

func (s Service) countManagedGatewayActiveInstances(ctx context.Context, gatewayID string) (int, error) {
	var count int
	if err := s.DB.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "ManagedGatewayInstance"
WHERE "gatewayId" = $1
  AND status NOT IN ('ERROR'::"ManagedInstanceStatus", 'REMOVING'::"ManagedInstanceStatus")
`, gatewayID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active managed gateway instances: %w", err)
	}
	return count, nil
}

func (s Service) listManagedGatewayInstancesForScale(ctx context.Context, gatewayID string) ([]managedGatewayInstanceResponse, error) {
	rows, err := s.DB.Query(ctx, `
SELECT
	id,
	"gatewayId",
	"containerId",
	"containerName",
	host,
	port,
	"apiPort",
	status::text,
	"orchestratorType",
	"healthStatus",
	"lastHealthCheck",
	"errorMessage",
	"consecutiveFailures",
	"tunnelProxyHost",
	"tunnelProxyPort",
	"createdAt",
	"updatedAt"
FROM "ManagedGatewayInstance"
WHERE "gatewayId" = $1
  AND status NOT IN ('ERROR'::"ManagedInstanceStatus", 'REMOVING'::"ManagedInstanceStatus")
ORDER BY "createdAt" DESC
`, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("list managed gateway instances: %w", err)
	}
	defer rows.Close()

	result := make([]managedGatewayInstanceResponse, 0)
	for rows.Next() {
		item, err := scanManagedGatewayInstance(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate managed gateway instances: %w", err)
	}
	return result, nil
}

func (s Service) deployManagedGatewayInstance(ctx context.Context, record gatewayRecord, runtimeClient *dockerSocketClient, orchestratorType, auditUserID, ipAddress string, updateDesired bool) (managedGatewayInstanceResponse, error) {
	instanceIndex, err := s.countManagedGatewayInstanceRecords(ctx, record.ID)
	if err != nil {
		return managedGatewayInstanceResponse{}, err
	}

	configs, err := s.buildManagedGatewayContainerConfig(ctx, record, instanceIndex+1)
	if err != nil {
		return managedGatewayInstanceResponse{}, err
	}
	if len(configs) == 0 {
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusInternalServerError, message: "No managed container configuration was generated for this gateway"}
	}

	var (
		containerInfo managedContainerInfo
		deployErr     error
		containerName = configs[0].Name
	)
	for _, cfg := range configs {
		containerName = cfg.Name
		containerInfo, deployErr = runtimeClient.deployContainer(ctx, cfg)
		if deployErr == nil {
			break
		}
	}
	if deployErr != nil {
		_ = s.recordManagedGatewayDeploymentFailure(ctx, record, orchestratorType, containerName, deployErr)
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusServiceUnavailable, message: fmt.Sprintf("Gateway deployment failed: %v", deployErr)}
	}

	instanceID := uuid.NewString()
	instanceHost, instancePort := managedGatewayInstanceAddress(record, containerInfo, s.managedGatewayPrimaryPort(record.Type), s.managedGatewayNetworks(record))
	apiPort := managedGatewayAPIPort(record, s.DefaultGRPCPort)

	if err := s.waitForManagedGatewayReady(ctx, record, runtimeClient, containerInfo.ID, containerInfo.Name, apiPort); err != nil {
		_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
		_ = s.recordManagedGatewayDeploymentFailure(ctx, record, orchestratorType, containerInfo.Name, err)
		return managedGatewayInstanceResponse{}, &requestError{status: http.StatusServiceUnavailable, message: fmt.Sprintf("Gateway deployment failed: %v", err)}
	}

	inspectedInfo, err := runtimeClient.inspectContainer(ctx, containerInfo.ID)
	if err == nil {
		containerInfo = inspectedInfo
	}
	instanceHost, instancePort = managedGatewayInstanceAddress(record, containerInfo, s.managedGatewayPrimaryPort(record.Type), s.managedGatewayNetworks(record))
	status := inferInstanceStatus(containerInfo.Status)
	healthStatus := inferInstanceHealth(containerInfo.Status, containerInfo.Health)
	if status == "RUNNING" {
		healthStatus = "healthy"
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
		return managedGatewayInstanceResponse{}, fmt.Errorf("begin managed gateway deploy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO "ManagedGatewayInstance" (
	id,
	"gatewayId",
	"containerId",
	"containerName",
	host,
	port,
	"apiPort",
	status,
	"orchestratorType",
	"healthStatus",
	"lastHealthCheck",
	"consecutiveFailures",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8::"ManagedInstanceStatus",
	$9,
	NULLIF($10, ''),
	NOW(),
	0,
	NOW(),
	NOW()
)
`, instanceID, record.ID, containerInfo.ID, containerInfo.Name, instanceHost, instancePort, apiPort, status, orchestratorType, healthStatus); err != nil {
		_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
		return managedGatewayInstanceResponse{}, fmt.Errorf("insert managed gateway instance: %w", err)
	}

	if updateDesired {
		activeCount, err := s.countManagedGatewayActiveInstances(ctx, record.ID)
		if err != nil {
			_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
			return managedGatewayInstanceResponse{}, err
		}
		desiredReplicas := record.DesiredReplicas
		if desiredReplicas < activeCount+1 {
			desiredReplicas = activeCount + 1
		}

		if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
   SET "desiredReplicas" = $2,
       "lastScaleAction" = NOW(),
       "updatedAt" = NOW()
 WHERE id = $1
`, record.ID, desiredReplicas); err != nil {
			_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
			return managedGatewayInstanceResponse{}, fmt.Errorf("update gateway deploy state: %w", err)
		}
	}

	if auditUserID != "" {
		if err := s.insertAuditLogTx(ctx, tx, auditUserID, "GATEWAY_DEPLOY", record.ID, map[string]any{
			"instanceId":       instanceID,
			"containerId":      containerInfo.ID,
			"containerName":    containerInfo.Name,
			"orchestratorType": orchestratorType,
			"host":             instanceHost,
			"port":             instancePort,
			"healthStatus":     healthStatus,
			"instanceStatus":   status,
			"publishedToHost":  record.PublishPorts && !record.TunnelEnabled,
			"tunnelEnabled":    record.TunnelEnabled,
			"gatewayType":      record.Type,
		}, ipAddress); err != nil {
			_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
			return managedGatewayInstanceResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		_ = runtimeClient.removeContainer(ctx, containerInfo.ID)
		return managedGatewayInstanceResponse{}, fmt.Errorf("commit managed gateway deploy transaction: %w", err)
	}

	return s.loadManagedGatewayInstance(ctx, record.ID, instanceID)
}

func (s Service) removeManagedGatewayInstance(ctx context.Context, runtimeClient *dockerSocketClient, gatewayID string, instance managedGatewayInstanceResponse) error {
	if _, err := s.DB.Exec(ctx, `
UPDATE "ManagedGatewayInstance"
   SET status = 'REMOVING'::"ManagedInstanceStatus",
       "updatedAt" = NOW()
 WHERE id = $1
`, instance.ID); err != nil {
		return fmt.Errorf("mark managed gateway instance removing: %w", err)
	}

	if err := runtimeClient.removeContainer(ctx, instance.ContainerID); err != nil {
		if _, updateErr := s.DB.Exec(ctx, `
UPDATE "ManagedGatewayInstance"
   SET status = 'ERROR'::"ManagedInstanceStatus",
       "healthStatus" = 'unhealthy',
       "errorMessage" = $2,
       "updatedAt" = NOW()
 WHERE id = $1
`, instance.ID, err.Error()); updateErr != nil {
			return fmt.Errorf("remove managed gateway container: %v (also failed to persist error: %w)", err, updateErr)
		}
		return &requestError{status: http.StatusServiceUnavailable, message: fmt.Sprintf("Gateway undeploy failed: %v", err)}
	}

	if _, err := s.DB.Exec(ctx, `DELETE FROM "ManagedGatewayInstance" WHERE id = $1`, instance.ID); err != nil {
		return fmt.Errorf("delete managed gateway instance: %w", err)
	}
	return nil
}

func (s Service) recordManagedGatewayDeploymentFailure(ctx context.Context, record gatewayRecord, orchestratorType, containerName string, deploymentErr error) error {
	if s.DB == nil || deploymentErr == nil {
		return nil
	}

	message := strings.TrimSpace(deploymentErr.Error())
	if message == "" {
		message = "container deployment failed"
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "ManagedGatewayInstance" (
	id,
	"gatewayId",
	"containerId",
	"containerName",
	host,
	port,
	status,
	"orchestratorType",
	"healthStatus",
	"errorMessage",
	"consecutiveFailures",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1,
	$2,
	$3,
	$4,
	'unknown',
	0,
	'ERROR'::"ManagedInstanceStatus",
	NULLIF($5, ''),
	'unhealthy',
	$6,
	1,
	NOW(),
	NOW()
)
`, uuid.NewString(), record.ID, fmt.Sprintf("failed-%d", time.Now().UTC().UnixNano()), containerName, orchestratorType, message)
	if err != nil {
		return fmt.Errorf("record managed gateway deployment failure: %w", err)
	}
	return nil
}
