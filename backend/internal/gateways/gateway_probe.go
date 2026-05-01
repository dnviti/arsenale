package gateways

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type gatewayProbeTarget struct {
	Host string
	Port int
}

func (s Service) probeGatewayRecordConnectivity(ctx context.Context, record gatewayRecord, timeout time.Duration) (connectivityResult, error) {
	if record.TunnelEnabled {
		return s.probeTunnelGatewayConnectivity(ctx, record, timeout), nil
	}
	if deploymentModeIsGroup(record.DeploymentMode) {
		return s.probeManagedGatewayConnectivity(ctx, record, timeout)
	}
	return tcpProbe(ctx, record.Host, record.Port, timeout), nil
}

func (s Service) probeManagedGatewayConnectivity(ctx context.Context, record gatewayRecord, timeout time.Duration) (connectivityResult, error) {
	var instanceHost, instanceContainerID, instanceContainerName string
	var instancePort int
	err := s.DB.QueryRow(ctx, `
SELECT host, port, "containerId", "containerName"
  FROM "ManagedGatewayInstance"
 WHERE "gatewayId" = $1
   AND status = 'RUNNING'
 ORDER BY "createdAt" ASC
 LIMIT 1
`, record.ID).Scan(&instanceHost, &instancePort, &instanceContainerID, &instanceContainerName)
	if err != nil {
		message := "No deployed instances for this gateway group"
		return connectivityResult{
			Reachable: false,
			Error:     &message,
		}, nil
	}

	instanceIPAddress := ""
	if runtimeClient, _, runtimeErr := s.managedGatewayRuntime(ctx); runtimeErr == nil && strings.TrimSpace(instanceContainerID) != "" {
		if info, inspectErr := runtimeClient.inspectContainer(ctx, instanceContainerID); inspectErr == nil {
			instanceIPAddress = info.IPAddress
		}
	}

	targets := buildManagedGatewayProbeTargets(record.Host, record.Port, instanceHost, instancePort, instanceContainerName, instanceIPAddress)
	var result connectivityResult
	for _, target := range targets {
		result = tcpProbe(ctx, target.Host, target.Port, timeout)
		if result.Reachable {
			return result, nil
		}
	}
	return result, nil
}

func (s Service) probeTunnelGatewayConnectivity(ctx context.Context, record gatewayRecord, timeout time.Duration) connectivityResult {
	proxy, err := s.createTunnelTCPProxy(ctx, record.ID, "127.0.0.1", record.Port)
	if err != nil {
		message := err.Error()
		return connectivityResult{
			Reachable: false,
			Error:     &message,
		}
	}
	return tcpProbe(ctx, proxy.Host, proxy.Port, timeout)
}

func buildManagedGatewayProbeTargets(gatewayHost string, gatewayPort int, instanceHost string, instancePort int, instanceContainerName, instanceIPAddress string) []gatewayProbeTarget {
	targets := make([]gatewayProbeTarget, 0, 3)
	seen := make(map[string]struct{})

	add := func(host string, port int) {
		host = strings.TrimSpace(host)
		if host == "" || port < 1 {
			return
		}
		key := fmt.Sprintf("%s:%d", host, port)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		targets = append(targets, gatewayProbeTarget{Host: host, Port: port})
	}

	add(instanceIPAddress, gatewayPort)
	add(instanceHost, instancePort)
	add(instanceContainerName, gatewayPort)
	add(gatewayHost, gatewayPort)

	return targets
}

func tcpProbe(ctx context.Context, host string, port int, timeout time.Duration) connectivityResult {
	start := time.Now()
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		message := err.Error()
		return connectivityResult{
			Reachable: false,
			Error:     &message,
		}
	}
	_ = conn.Close()
	latency := int(time.Since(start).Milliseconds())
	return connectivityResult{
		Reachable: true,
		LatencyMS: &latency,
	}
}
