package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type gatewaySummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	OperationalStatus string `json:"operationalStatus"`
	OperationalReason string `json:"operationalReason"`
	HealthyInstances  int    `json:"healthyInstances"`
	RunningInstances  int    `json:"runningInstances"`
	DesiredReplicas   int    `json:"desiredReplicas"`
	TunnelConnected   bool   `json:"tunnelConnected"`
}

type gatewayInstance struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	HealthStatus    string `json:"healthStatus"`
	Host            string `json:"host"`
	Port            any    `json:"port"`
	ContainerName   string `json:"containerName"`
	UpdatedAt       string `json:"updatedAt"`
	CreatedAt       string `json:"createdAt"`
	LastHealthCheck string `json:"lastHealthCheck"`
}

func authenticatedGatewayConfig() *CLIConfig {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}
	return cfg
}

func getGatewayByID(cfg *CLIConfig, gatewayID string) (gatewaySummary, error) {
	body, status, err := apiGet("/api/gateways", cfg)
	if err != nil {
		return gatewaySummary{}, err
	}
	checkAPIError(status, body)

	var gateways []gatewaySummary
	if err := json.Unmarshal(body, &gateways); err != nil {
		return gatewaySummary{}, fmt.Errorf("parse gateway list: %w", err)
	}
	for _, gateway := range gateways {
		if gateway.ID == gatewayID {
			return gateway, nil
		}
	}
	return gatewaySummary{}, fmt.Errorf("gateway %q not found", gatewayID)
}

func getGatewayInstances(cfg *CLIConfig, gatewayID string) ([]gatewayInstance, error) {
	body, status, err := apiGet(fmt.Sprintf("/api/gateways/%s/instances", gatewayID), cfg)
	if err != nil {
		return nil, err
	}
	checkAPIError(status, body)

	var instances []gatewayInstance
	if err := json.Unmarshal(body, &instances); err != nil {
		return nil, fmt.Errorf("parse gateway instances: %w", err)
	}
	return instances, nil
}

func selectGatewayInstance(instances []gatewayInstance, wantedID string) (gatewayInstance, error) {
	if len(instances) == 0 {
		return gatewayInstance{}, fmt.Errorf("gateway has no managed instances")
	}
	if wantedID != "" {
		for _, instance := range instances {
			if instance.ID == wantedID {
				return instance, nil
			}
		}
		return gatewayInstance{}, fmt.Errorf("gateway instance %q not found", wantedID)
	}

	best := instances[0]
	for _, candidate := range instances[1:] {
		if compareGatewayInstances(candidate, best) > 0 {
			best = candidate
		}
	}
	return best, nil
}

func compareGatewayInstances(a, b gatewayInstance) int {
	if rankA, rankB := gatewayInstanceRank(a), gatewayInstanceRank(b); rankA != rankB {
		if rankA > rankB {
			return 1
		}
		return -1
	}

	timeA := gatewayInstanceTimestamp(a)
	timeB := gatewayInstanceTimestamp(b)
	if timeA.After(timeB) {
		return 1
	}
	if timeB.After(timeA) {
		return -1
	}
	return 0
}

func gatewayInstanceRank(instance gatewayInstance) int {
	status := strings.ToUpper(strings.TrimSpace(instance.Status))
	health := strings.ToLower(strings.TrimSpace(instance.HealthStatus))

	switch {
	case status == "RUNNING" && health == "healthy":
		return 3
	case status == "RUNNING":
		return 2
	case health == "healthy":
		return 1
	default:
		return 0
	}
}

func gatewayInstanceTimestamp(instance gatewayInstance) time.Time {
	for _, raw := range []string{instance.UpdatedAt, instance.CreatedAt, instance.LastHealthCheck} {
		ts, err := time.Parse(time.RFC3339, raw)
		if err == nil {
			return ts
		}
	}
	return time.Time{}
}

func mustMarshalJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		fatal("marshal output: %v", err)
	}
	return data
}
