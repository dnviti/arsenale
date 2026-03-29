package workloadspec

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestValidateForOCIRejectsKubernetesFields(t *testing.T) {
	spec := WorkloadSpec{
		Name:  "terminal-broker",
		Image: "ghcr.io/dnviti/arsenale/terminal-broker:dev",
		Kubernetes: KubernetesOptions{
			Namespace: "arsenale",
		},
	}

	result := spec.ValidateFor(contracts.OrchestratorDocker)
	if result.Valid {
		t.Fatalf("expected invalid result, got %+v", result)
	}
}

func TestValidateForKubernetesAcceptsMinimalSpec(t *testing.T) {
	spec := WorkloadSpec{
		Name:  "desktop-broker",
		Image: "ghcr.io/dnviti/arsenale/desktop-broker:dev",
		Ports: []Port{
			{Container: 8080, Protocol: "tcp"},
		},
		Kubernetes: KubernetesOptions{
			Namespace: "arsenale-runtime",
		},
	}

	result := spec.ValidateFor(contracts.OrchestratorKubernetes)
	if !result.Valid {
		t.Fatalf("expected valid result, got %+v", result)
	}
}
