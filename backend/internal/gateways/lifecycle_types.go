package gateways

import (
	"strconv"
	"strings"
	"time"
)

const (
	maxManagedGatewayReplicas  = 20
	defaultGatewayLogTailLines = 200
	maxGatewayLogTailLines     = 5000
	unavailableOrchestratorMsg = "Container orchestration not available. Configure Docker socket, Podman socket, or Kubernetes credentials."
	managedGatewayReadyTimeout = 60 * time.Second
	managedGatewayReadyPoll    = 250 * time.Millisecond
)

type scalePayload struct {
	Replicas int `json:"replicas"`
}

type scaleResult struct {
	Deployed int `json:"deployed"`
	Removed  int `json:"removed"`
}

type restartResult struct {
	Restarted bool `json:"restarted"`
}

type undeployResult struct {
	Undeployed bool `json:"undeployed"`
}

type instanceLogsResponse struct {
	Logs          string `json:"logs"`
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Timestamp     string `json:"timestamp"`
}

func isManagedLifecycleGatewayType(gatewayType string) bool {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH", "GUACD", "DB_PROXY":
		return true
	default:
		return false
	}
}

func parseGatewayLogTail(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value == 0 {
		return defaultGatewayLogTailLines
	}
	if value < 1 {
		return 1
	}
	if value > maxGatewayLogTailLines {
		return maxGatewayLogTailLines
	}
	return value
}
