package main

import (
	"testing"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
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

func TestBuildDevGatewaySpecsIncludesLocalConnectionGateways(t *testing.T) {
	specs := buildDevGatewaySpecs("/certs", devBootstrapRuntime{
		features: runtimefeatures.Manifest{
			ConnectionsEnabled: true,
		},
	})

	if len(specs) != 2 {
		t.Fatalf("expected 2 local connection gateways, got %d", len(specs))
	}
	if specs[0].Host != "ssh-gateway" || specs[0].DeploymentMode != "SINGLE_INSTANCE" || specs[0].TunnelEnabled {
		t.Fatalf("unexpected local managed SSH gateway spec: %+v", specs[0])
	}
	if specs[1].Host != "guacd" || specs[1].DeploymentMode != "SINGLE_INSTANCE" || specs[1].TunnelEnabled {
		t.Fatalf("unexpected local GUACD gateway spec: %+v", specs[1])
	}
}

func TestBuildDevGatewaySpecsAddsTunnelFixturesWhenEnabled(t *testing.T) {
	specs := buildDevGatewaySpecs("/certs", devBootstrapRuntime{
		features: runtimefeatures.Manifest{
			ConnectionsEnabled:   true,
			DatabaseProxyEnabled: true,
			ZeroTrustEnabled:     true,
		},
		tunnelFixturesEnabled: true,
	})

	if len(specs) != 5 {
		t.Fatalf("expected local plus tunnel gateway specs, got %d", len(specs))
	}
	if !specs[2].TunnelEnabled || specs[2].DeploymentMode != "MANAGED_GROUP" {
		t.Fatalf("unexpected tunnel managed SSH spec: %+v", specs[2])
	}
	if !specs[4].TunnelEnabled || specs[4].Type != "DB_PROXY" {
		t.Fatalf("unexpected tunnel DB proxy spec: %+v", specs[4])
	}
}
