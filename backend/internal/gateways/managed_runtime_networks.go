package gateways

import (
	"fmt"
	"strings"
)

func (s Service) managedGatewayNetworks(record gatewayRecord) []string {
	edgeNetwork := firstNonEmpty(s.EdgeNetwork, defaultEdgeNetwork)
	dbNetwork := firstNonEmpty(s.DBNetwork, defaultDBNetwork)
	guacdNetwork := firstNonEmpty(s.GuacdNetwork, defaultGuacdNetwork)
	gatewayNetwork := firstNonEmpty(s.GatewayNetwork, defaultGatewayNetwork)

	switch strings.ToUpper(strings.TrimSpace(record.Type)) {
	case "MANAGED_SSH":
		return normalizedStrings([]string{edgeNetwork, gatewayNetwork})
	case "GUACD":
		networks := []string{guacdNetwork}
		if record.TunnelEnabled {
			networks = append(networks, gatewayNetwork)
		}
		return normalizedStrings(networks)
	case "DB_PROXY":
		networks := []string{edgeNetwork, dbNetwork}
		if record.TunnelEnabled {
			networks = append(networks, gatewayNetwork)
		}
		return normalizedStrings(networks)
	default:
		return nil
	}
}

func (s Service) managedGatewayAttachNetworks(record gatewayRecord) []string {
	networks := make([]string, 0, 4)
	if egressNetwork := strings.TrimSpace(s.EgressNetwork); egressNetwork != "" {
		networks = append(networks, egressNetwork)
	}
	networks = append(networks, s.managedGatewayNetworks(record)...)
	return normalizedStrings(networks)
}

func (s Service) managedGatewayImageCandidates(gatewayType string) []string {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH":
		return normalizedStrings([]string{
			firstNonEmpty(s.SSHGatewayImage, localSSHGatewayImage),
			localSSHGatewayImage,
			remoteSSHGatewayImage,
		})
	case "GUACD":
		return normalizedStrings([]string{
			firstNonEmpty(s.GuacdImage, localGuacdImage),
			localGuacdImage,
			remoteGuacdImage,
		})
	case "DB_PROXY":
		return normalizedStrings([]string{
			firstNonEmpty(s.DBProxyImage, remoteDBProxyImage),
			localDBProxyImage,
			localDevDBProxyImage,
			remoteDBProxyImage,
		})
	default:
		return nil
	}
}

func (s Service) managedGatewayPrimaryPort(gatewayType string) int {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH":
		return 2222
	case "GUACD":
		return 4822
	case "DB_PROXY":
		return 5432
	default:
		return 0
	}
}

func (s Service) managedGatewayPublishedPorts(record gatewayRecord) ([]managedContainerPortBinding, error) {
	containerPort := s.managedGatewayPrimaryPort(record.Type)
	if containerPort <= 0 {
		return nil, fmt.Errorf("unsupported managed gateway type %q", record.Type)
	}

	publish := record.PublishPorts && !record.TunnelEnabled
	ports := []managedContainerPortBinding{{
		ContainerPort: containerPort,
		Publish:       publish,
	}}
	if publish {
		hostPort, err := findAvailableLoopbackPort()
		if err != nil {
			return nil, err
		}
		ports[0].HostPort = hostPort
	}
	return ports, nil
}
