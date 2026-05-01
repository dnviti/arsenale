package memory

import (
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func ValidateNamespace(ns contracts.MemoryNamespace) error {
	if strings.TrimSpace(ns.TenantID) == "" {
		return fmt.Errorf("tenantId is required")
	}
	if strings.TrimSpace(ns.Name) == "" {
		return fmt.Errorf("name is required")
	}

	switch ns.Scope {
	case contracts.MemoryScopeTenant:
	case contracts.MemoryScopePrincipal:
		if ns.PrincipalID == "" {
			return fmt.Errorf("principalId is required for principal scope")
		}
	case contracts.MemoryScopeAgent:
		if ns.AgentID == "" {
			return fmt.Errorf("agentId is required for agent scope")
		}
	case contracts.MemoryScopeRun:
		if ns.RunID == "" {
			return fmt.Errorf("runId is required for run scope")
		}
	case contracts.MemoryScopeWorkflow:
		if ns.WorkflowID == "" {
			return fmt.Errorf("workflowId is required for workflow scope")
		}
	default:
		return fmt.Errorf("unsupported scope %q", ns.Scope)
	}

	switch ns.Type {
	case contracts.MemoryWorking, contracts.MemoryEpisodic, contracts.MemorySemantic, contracts.MemoryProcedural, contracts.MemoryArtifact:
	default:
		return fmt.Errorf("unsupported memory type %q", ns.Type)
	}

	return nil
}

func NamespaceKey(ns contracts.MemoryNamespace) string {
	parts := []string{
		"tenant=" + ns.TenantID,
		"scope=" + string(ns.Scope),
		"type=" + string(ns.Type),
		"name=" + ns.Name,
	}
	if ns.PrincipalID != "" {
		parts = append(parts, "principal="+ns.PrincipalID)
	}
	if ns.AgentID != "" {
		parts = append(parts, "agent="+ns.AgentID)
	}
	if ns.RunID != "" {
		parts = append(parts, "run="+ns.RunID)
	}
	if ns.WorkflowID != "" {
		parts = append(parts, "workflow="+ns.WorkflowID)
	}
	return strings.Join(parts, "/")
}
