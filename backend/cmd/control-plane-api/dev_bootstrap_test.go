package main

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestParseBootstrapOrchestratorKindDefaultsToPodman(t *testing.T) {
	if got := parseBootstrapOrchestratorKind(""); got != contracts.OrchestratorPodman {
		t.Fatalf("expected default podman kind, got %q", got)
	}
}

func TestParseBootstrapOrchestratorKindAcceptsKnownValues(t *testing.T) {
	if got := parseBootstrapOrchestratorKind("docker"); got != contracts.OrchestratorDocker {
		t.Fatalf("expected docker kind, got %q", got)
	}
	if got := parseBootstrapOrchestratorKind("kubernetes"); got != contracts.OrchestratorKubernetes {
		t.Fatalf("expected kubernetes kind, got %q", got)
	}
}

func TestParseBootstrapOrchestratorScopeDefaultsToGlobal(t *testing.T) {
	if got := parseBootstrapOrchestratorScope(""); got != contracts.OrchestratorScopeGlobal {
		t.Fatalf("expected default global scope, got %q", got)
	}
}

func TestParseBootstrapOrchestratorScopeAcceptsTenant(t *testing.T) {
	if got := parseBootstrapOrchestratorScope("tenant"); got != contracts.OrchestratorScopeTenant {
		t.Fatalf("expected tenant scope, got %q", got)
	}
}
