package authz

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestReadOnlyActionAllowed(t *testing.T) {
	decision := NewStaticPDP().Evaluate(contracts.AuthzRequest{
		Subject: contracts.PrincipalRef{Type: contracts.PrincipalUser, ID: "usr_1"},
		Action:  "connection.read",
		Resource: contracts.ResourceRef{
			Type: "connection",
			ID:   "conn_1",
		},
	})

	if decision.Effect != contracts.AuthzAllow {
		t.Fatalf("expected allow, got %+v", decision)
	}
}

func TestElevatedActionDeniedWithoutApproval(t *testing.T) {
	decision := NewStaticPDP().Evaluate(contracts.AuthzRequest{
		Subject: contracts.PrincipalRef{Type: contracts.PrincipalAgentRun, ID: "run_1"},
		Action:  "workload.deploy",
		Resource: contracts.ResourceRef{
			Type: "workload",
			ID:   "desktop-broker",
		},
	})

	if decision.Effect != contracts.AuthzDeny {
		t.Fatalf("expected deny, got %+v", decision)
	}
}
