package contracts

import "time"

type MemoryType string

const (
	MemoryWorking    MemoryType = "working"
	MemoryEpisodic   MemoryType = "episodic"
	MemorySemantic   MemoryType = "semantic"
	MemoryProcedural MemoryType = "procedural"
	MemoryArtifact   MemoryType = "artifact"
)

type MemoryScope string

const (
	MemoryScopeTenant    MemoryScope = "tenant"
	MemoryScopePrincipal MemoryScope = "principal"
	MemoryScopeAgent     MemoryScope = "agent"
	MemoryScopeRun       MemoryScope = "run"
	MemoryScopeWorkflow  MemoryScope = "workflow"
)

type MemoryNamespace struct {
	TenantID    string      `json:"tenantId"`
	Scope       MemoryScope `json:"scope"`
	PrincipalID string      `json:"principalId,omitempty"`
	AgentID     string      `json:"agentId,omitempty"`
	RunID       string      `json:"runId,omitempty"`
	WorkflowID  string      `json:"workflowId,omitempty"`
	Type        MemoryType  `json:"type"`
	Name        string      `json:"name"`
}

type MemoryNamespaceRecord struct {
	ID        string          `json:"id"`
	Key       string          `json:"key"`
	Namespace MemoryNamespace `json:"namespace"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type MemoryItem struct {
	ID           string            `json:"id"`
	NamespaceKey string            `json:"namespaceKey"`
	Content      string            `json:"content"`
	Summary      string            `json:"summary,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
}

type MemoryWriteRequest struct {
	Namespace MemoryNamespace   `json:"namespace"`
	Content   string            `json:"content"`
	Summary   string            `json:"summary,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
