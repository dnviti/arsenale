package cmd

import (
	"strings"
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
)

func TestGatewayCreateLongHelpListsEveryType(t *testing.T) {
	help := gatewayCreateLongHelp()
	for _, ti := range gatewayruntime.Catalog() {
		if !strings.Contains(help, ti.Type) {
			t.Errorf("create help missing type code %q", ti.Type)
		}
		if !strings.Contains(help, ti.DisplayName) {
			t.Errorf("create help missing display name %q", ti.DisplayName)
		}
	}
	if !strings.Contains(help, "arsenale gateway types") {
		t.Error("create help should point at the types command")
	}
}

func TestGatewayDeploymentSummaryDescribesDeployment(t *testing.T) {
	summary := gatewayDeploymentSummary(tunnelTokenBundle{GatewayType: "GUACD", TunnelLocalPort: 4822})
	for _, want := range []string{"Remote Desktop Gateway (Guacamole)", "Arsenale-managed", "compose service guacd", "local port 4822"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary missing %q; got:\n%s", want, summary)
		}
	}

	// Unknown type yields no summary rather than a confusing partial one.
	if got := gatewayDeploymentSummary(tunnelTokenBundle{GatewayType: "NOPE"}); got != "" {
		t.Errorf("expected empty summary for unknown type, got %q", got)
	}
}
