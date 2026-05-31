package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
	"github.com/spf13/cobra"
)

// runGwTypes prints the gateway type catalog so operators can understand what
// each type deploys before creating one. The catalog is read locally from the
// shared gatewayruntime definitions, so it needs no login and is always
// consistent with this CLI build.
func runGwTypes(_ *cobra.Command, _ []string) {
	catalog := gatewayruntime.Catalog()

	p := printer()
	if p.Format == "json" || p.Format == "yaml" {
		data, err := json.Marshal(map[string]any{"types": catalog})
		if err != nil {
			fatal("%v", err)
		}
		if err := p.Print(data, nil); err != nil {
			fatal("%v", err)
		}
		return
	}

	out := os.Stdout
	for i, ti := range catalog {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "%s — %s   [%s]\n", ti.Type, ti.DisplayName, ti.DeploymentModel)
		fmt.Fprintf(out, "  %s\n", ti.Description)
		fmt.Fprintf(out, "  Protocols: %s · Default port: %d · Deployment modes: %s\n",
			strings.Join(ti.Protocols, ", "), ti.DefaultPort, strings.Join(ti.DeploymentModes, ", "))
		if ti.Image != "" {
			fmt.Fprintf(out, "  Image: %s\n", ti.Image)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, `Create one with: arsenale gateway create --from-file gw.yaml  (set "type" to a code above)`)
}

// gatewayCreateLongHelp builds the `gateway create` help text from the live
// catalog so the listed types never drift from the definitions.
func gatewayCreateLongHelp() string {
	var b strings.Builder
	b.WriteString("Create a gateway from a JSON/YAML file:\n")
	b.WriteString("  arsenale gateway create --from-file gw.yaml\n\n")
	b.WriteString("The file must set \"type\" to one of:\n")

	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, ti := range gatewayruntime.Catalog() {
		fmt.Fprintf(w, "  %s\t%s\t(%s)\n", ti.Type, ti.DisplayName, ti.DeploymentModel)
	}
	_ = w.Flush()

	b.WriteString("\nRun \"arsenale gateway types\" for full descriptions of what each one deploys.")
	return b.String()
}

// gatewayDeploymentSummary returns a plain-language summary of what a tunnel
// bundle deploys, shown after the bundle is written.
func gatewayDeploymentSummary(bundle tunnelTokenBundle) string {
	info, ok := gatewayruntime.Info(bundle.GatewayType)
	if !ok {
		return ""
	}

	localPort := bundle.TunnelLocalPort
	if localPort <= 0 {
		localPort = gatewayruntime.TunnelLocalPort(bundle.GatewayType, 0)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\nYou are deploying: %s  [%s]\n", info.DisplayName, info.DeploymentModel)
	fmt.Fprintf(&b, "  %s\n", info.Summary)
	service := gatewayruntime.ComposeService(bundle.GatewayType)
	parts := []string{}
	if service != "" {
		parts = append(parts, "compose service "+service)
	}
	if info.Image != "" {
		parts = append(parts, "image "+info.Image)
	}
	if localPort > 0 {
		parts = append(parts, fmt.Sprintf("local port %d", localPort))
	}
	if len(parts) > 0 {
		fmt.Fprintf(&b, "  Deploys: %s\n", strings.Join(parts, " · "))
	}
	return b.String()
}
