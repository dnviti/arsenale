package contracts

import "time"

type CapabilityRisk string

const (
	CapabilityRiskLow      CapabilityRisk = "low"
	CapabilityRiskMedium   CapabilityRisk = "medium"
	CapabilityRiskHigh     CapabilityRisk = "high"
	CapabilityRiskCritical CapabilityRisk = "critical"
)

type CapabilityDefinition struct {
	ID               string         `json:"id"`
	Action           string         `json:"action"`
	ResourceType     string         `json:"resourceType"`
	Description      string         `json:"description"`
	Risk             CapabilityRisk `json:"risk"`
	RequiresApproval bool           `json:"requiresApproval"`
}

type ToolCallPlanRequest struct {
	Capability string       `json:"capability"`
	Authz      AuthzRequest `json:"authz"`
}

type ToolCallPlanResponse struct {
	Capability CapabilityDefinition `json:"capability"`
	Decision   AuthzDecision        `json:"decision"`
	DryRun     bool                 `json:"dryRun"`
}

type AgentRunRequest struct {
	TenantID              string   `json:"tenantId"`
	DefinitionID          string   `json:"definitionId"`
	Trigger               string   `json:"trigger"`
	Goals                 []string `json:"goals"`
	RequestedCapabilities []string `json:"requestedCapabilities"`
}

type AgentRunStatus string

const (
	AgentRunQueued    AgentRunStatus = "queued"
	AgentRunRunning   AgentRunStatus = "running"
	AgentRunSucceeded AgentRunStatus = "succeeded"
	AgentRunFailed    AgentRunStatus = "failed"
	AgentRunCanceled  AgentRunStatus = "canceled"
)

type AgentRun struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenantId"`
	DefinitionID     string         `json:"definitionId"`
	Trigger          string         `json:"trigger,omitempty"`
	Goals            []string       `json:"goals"`
	RequestedCaps    []string       `json:"requestedCapabilities"`
	Status           AgentRunStatus `json:"status"`
	RequiresApproval bool           `json:"requiresApproval"`
	RequestedAt      time.Time      `json:"requestedAt"`
	LastTransitionAt time.Time      `json:"lastTransitionAt"`
}
