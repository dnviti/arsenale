package gateways

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func (s Service) waitForManagedGatewayReady(ctx context.Context, record gatewayRecord, runtimeClient *dockerSocketClient, containerID, containerName string, apiPort *int) error {
	primaryPort := s.managedGatewayPrimaryPort(record.Type)
	if primaryPort <= 0 {
		return fmt.Errorf("managed gateway %q has no routable readiness target", record.Name)
	}
	preferredNetworks := s.managedGatewayNetworks(record)

	deadlineCtx, cancel := context.WithTimeout(ctx, managedGatewayReadyTimeout)
	defer cancel()

	var lastErr string
	for {
		currentInfo, err := runtimeClient.inspectContainer(deadlineCtx, containerID)
		if err != nil {
			lastErr = fmt.Sprintf("inspect container %s: %v", containerName, err)
		}
		probeHost := managedGatewayProbeHost(currentInfo, preferredNetworks)
		if probeHost == "" {
			log.Printf("managed gateway readiness waiting for container IP: container=%s id=%s", containerName, containerID)
			timer := time.NewTimer(managedGatewayReadyPoll)
			select {
			case <-deadlineCtx.Done():
				timer.Stop()
				if lastErr == "" {
					lastErr = fmt.Sprintf("managed gateway %q did not become ready before timeout", record.Name)
				}
				return fmt.Errorf("%s", lastErr)
			case <-timer.C:
				continue
			}
		}

		probes := []struct {
			name string
			host string
			port int
		}{
			{name: "service", host: probeHost, port: primaryPort},
		}
		if strings.EqualFold(record.Type, "MANAGED_SSH") && apiPort != nil && *apiPort > 0 {
			probes = append(probes, struct {
				name string
				host string
				port int
			}{
				name: "grpc",
				host: probeHost,
				port: *apiPort,
			})
		}

		allReady := true
		for _, probe := range probes {
			result := tcpProbe(deadlineCtx, probe.host, probe.port, time.Second)
			if result.Reachable {
				continue
			}
			if result.Error != nil {
				log.Printf("managed gateway readiness probe failed: container=%s target=%s:%d error=%s", containerName, probe.host, probe.port, strings.TrimSpace(*result.Error))
			} else {
				log.Printf("managed gateway readiness probe failed: container=%s target=%s:%d", containerName, probe.host, probe.port)
			}
			allReady = false
			if result.Error != nil {
				lastErr = fmt.Sprintf("%s endpoint %s:%d not ready: %s", probe.name, probe.host, probe.port, strings.TrimSpace(*result.Error))
			} else {
				lastErr = fmt.Sprintf("%s endpoint %s:%d not ready", probe.name, probe.host, probe.port)
			}
			break
		}
		if allReady {
			return nil
		}

		timer := time.NewTimer(managedGatewayReadyPoll)
		select {
		case <-deadlineCtx.Done():
			timer.Stop()
			if lastErr == "" {
				lastErr = fmt.Sprintf("managed gateway %q did not become ready before timeout", record.Name)
			}
			return fmt.Errorf("%s", lastErr)
		case <-timer.C:
		}
	}
}

func managedGatewayProbeHost(info managedContainerInfo, preferredNetworks []string) string {
	for _, networkName := range normalizedStrings(preferredNetworks) {
		if ip := strings.TrimSpace(info.NetworkIPs[networkName]); ip != "" {
			return ip
		}
	}
	return strings.TrimSpace(info.IPAddress)
}

func managedGatewayInstanceAddress(record gatewayRecord, info managedContainerInfo, fallbackPort int, preferredNetworks []string) (string, int) {
	port := fallbackPort
	if containerPort, ok := info.ContainerPorts[fallbackPort]; ok && containerPort > 0 {
		port = containerPort
	}
	for _, networkName := range normalizedStrings(preferredNetworks) {
		if ip := strings.TrimSpace(info.NetworkIPs[networkName]); ip != "" {
			return ip, port
		}
	}
	if strings.TrimSpace(info.IPAddress) != "" {
		return strings.TrimSpace(info.IPAddress), port
	}
	if publishedPort, ok := info.PublishedPorts[fallbackPort]; ok && publishedPort > 0 {
		host := strings.TrimSpace(record.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		return host, publishedPort
	}
	return inferPrimaryInstanceHost(record, info.Name), port
}
