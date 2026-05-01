package authz

import (
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type StaticPDP struct{}

func NewStaticPDP() StaticPDP {
	return StaticPDP{}
}

func (p StaticPDP) Evaluate(req contracts.AuthzRequest) contracts.AuthzDecision {
	if req.Subject.Type == "" || req.Subject.ID == "" {
		return contracts.AuthzDecision{
			Effect: contracts.AuthzDeny,
			Reason: "subject is required",
		}
	}
	if strings.TrimSpace(req.Action) == "" {
		return contracts.AuthzDecision{
			Effect: contracts.AuthzDeny,
			Reason: "action is required",
		}
	}
	if strings.TrimSpace(req.Resource.Type) == "" {
		return contracts.AuthzDecision{
			Effect: contracts.AuthzDeny,
			Reason: "resource.type is required",
		}
	}

	action := strings.ToLower(req.Action)
	if isReadOnlyAction(action) {
		return contracts.AuthzDecision{
			Effect: contracts.AuthzAllow,
			Reason: "read-only action permitted by static bootstrap policy",
			Obligations: []contracts.Obligation{
				{Type: contracts.ObligationReadOnly},
				{Type: contracts.ObligationMaskSecrets},
			},
		}
	}

	if req.Context["approved"] == "true" {
		return contracts.AuthzDecision{
			Effect: contracts.AuthzAllow,
			Reason: "approved elevated action permitted by static bootstrap policy",
			Obligations: []contracts.Obligation{
				{Type: contracts.ObligationRecordSession},
				{Type: contracts.ObligationMaskSecrets},
				{Type: contracts.ObligationTimeLimit, Parameters: map[string]string{"seconds": "900"}},
			},
		}
	}

	return contracts.AuthzDecision{
		Effect: contracts.AuthzDeny,
		Reason: "elevated action requires approval",
		Obligations: []contracts.Obligation{
			{Type: contracts.ObligationApproval},
		},
	}
}

func isReadOnlyAction(action string) bool {
	allowedSuffixes := []string{
		".read",
		".list",
		".search",
		".readonly",
	}
	for _, suffix := range allowedSuffixes {
		if strings.HasSuffix(action, suffix) {
			return true
		}
	}
	return false
}
