package contracts

type PrincipalType string

const (
	PrincipalUser           PrincipalType = "user"
	PrincipalServiceAccount PrincipalType = "service_account"
	PrincipalAgentDef       PrincipalType = "agent_definition"
	PrincipalAgentInstance  PrincipalType = "agent_instance"
	PrincipalAgentRun       PrincipalType = "agent_run"
	PrincipalSystem         PrincipalType = "system"
)

type PrincipalRef struct {
	Type PrincipalType `json:"type"`
	ID   string        `json:"id"`
}

type ResourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type DelegationRef struct {
	Type PrincipalType `json:"type"`
	ID   string        `json:"id"`
}

type AuthzRequest struct {
	Subject         PrincipalRef      `json:"subject"`
	Action          string            `json:"action"`
	Resource        ResourceRef       `json:"resource"`
	Context         map[string]string `json:"context,omitempty"`
	DelegationChain []DelegationRef   `json:"delegationChain,omitempty"`
}

type AuthzEffect string

const (
	AuthzAllow AuthzEffect = "allow"
	AuthzDeny  AuthzEffect = "deny"
)

type ObligationType string

const (
	ObligationReadOnly      ObligationType = "read_only"
	ObligationMaskSecrets   ObligationType = "mask_secrets"
	ObligationRecordSession ObligationType = "record_session"
	ObligationApproval      ObligationType = "require_approval"
	ObligationSandboxOnly   ObligationType = "sandbox_only"
	ObligationMaxRows       ObligationType = "max_rows"
	ObligationTimeLimit     ObligationType = "time_limit"
)

type Obligation struct {
	Type       ObligationType    `json:"type"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type AuthzDecision struct {
	Effect      AuthzEffect  `json:"effect"`
	Reason      string       `json:"reason"`
	Obligations []Obligation `json:"obligations,omitempty"`
}
