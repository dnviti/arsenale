package gateways

import (
	"context"
	"net/http"
	"strings"
)

func (s Service) managedGatewayRuntime(ctx context.Context) (*dockerSocketClient, string, error) {
	requested := strings.ToLower(strings.TrimSpace(s.OrchestratorType))
	switch requested {
	case "none":
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	case "docker":
		client, err := newDockerSocketClient("docker", s.DockerSocketPath)
		if err != nil {
			return nil, "", err
		}
		if err := client.ping(ctx); err != nil {
			return nil, "", &requestError{status: http.StatusServiceUnavailable, message: unavailableOrchestratorMsg}
		}
		return client, "DOCKER", nil
	case "podman":
		client, err := newDockerSocketClient("podman", s.PodmanSocketPath)
		if err != nil {
			return nil, "", err
		}
		if err := client.ping(ctx); err != nil {
			return nil, "", &requestError{status: http.StatusServiceUnavailable, message: unavailableOrchestratorMsg}
		}
		return client, "PODMAN", nil
	case "", "auto":
		candidates := []struct {
			kind       string
			socketPath string
		}{
			{kind: "podman", socketPath: s.PodmanSocketPath},
			{kind: "docker", socketPath: s.DockerSocketPath},
		}
		for _, candidate := range candidates {
			client, err := newDockerSocketClient(candidate.kind, candidate.socketPath)
			if err != nil {
				continue
			}
			if err := client.ping(ctx); err == nil {
				return client, strings.ToUpper(candidate.kind), nil
			}
		}
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	default:
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	}
}
