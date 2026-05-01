package orchestration

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestValidateConnectionDockerEndpoint(t *testing.T) {
	result := ValidateConnection(contracts.OrchestratorConnection{
		Name:     "edge-host-a",
		Kind:     contracts.OrchestratorDocker,
		Scope:    contracts.OrchestratorScopeGlobal,
		Endpoint: "unix:///var/run/docker.sock",
	})

	if !result.Valid {
		t.Fatalf("expected valid result, got %+v", result)
	}
}

func TestValidateConnectionRejectsBadKubernetesEndpoint(t *testing.T) {
	result := ValidateConnection(contracts.OrchestratorConnection{
		Name:     "cluster-a",
		Kind:     contracts.OrchestratorKubernetes,
		Scope:    contracts.OrchestratorScopeGlobal,
		Endpoint: "http://cluster.internal",
	})

	if result.Valid {
		t.Fatalf("expected invalid result, got %+v", result)
	}
}
