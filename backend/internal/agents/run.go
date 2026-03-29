package agents

import (
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func ValidateRunRequest(req contracts.AgentRunRequest) error {
	if strings.TrimSpace(req.TenantID) == "" {
		return fmt.Errorf("tenantId is required")
	}
	if strings.TrimSpace(req.DefinitionID) == "" {
		return fmt.Errorf("definitionId is required")
	}
	if len(req.Goals) == 0 {
		return fmt.Errorf("at least one goal is required")
	}

	known := make(map[string]struct{}, len(catalog.Capabilities()))
	for _, capability := range catalog.Capabilities() {
		known[capability.ID] = struct{}{}
	}

	for _, capability := range req.RequestedCapabilities {
		if _, ok := known[capability]; !ok {
			return fmt.Errorf("unknown requested capability %q", capability)
		}
	}

	return nil
}
