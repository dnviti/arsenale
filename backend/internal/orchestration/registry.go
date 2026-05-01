package orchestration

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

var validConnectionName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func ValidateConnection(conn contracts.OrchestratorConnection) ValidationResult {
	var errs []string
	var warnings []string

	if !validConnectionName.MatchString(conn.Name) {
		errs = append(errs, "name must be lowercase DNS-safe and 1-63 chars long")
	}

	switch conn.Scope {
	case contracts.OrchestratorScopeGlobal, contracts.OrchestratorScopeTenant:
	default:
		errs = append(errs, "scope must be global or tenant")
	}

	switch conn.Kind {
	case contracts.OrchestratorDocker, contracts.OrchestratorPodman:
		if !isSupportedOCIEndpoint(conn.Endpoint) {
			errs = append(errs, "docker/podman endpoints must use unix://, tcp://, http://, https://, or ssh://")
		}
		if conn.Namespace != "" {
			warnings = append(warnings, "namespace is ignored for docker and podman connections")
		}
	case contracts.OrchestratorKubernetes:
		if !isSupportedKubernetesEndpoint(conn.Endpoint) {
			errs = append(errs, "kubernetes endpoint must be https://... or in-cluster")
		}
	default:
		errs = append(errs, fmt.Sprintf("unsupported orchestrator kind %q", conn.Kind))
	}

	return ValidationResult{
		Valid:    len(errs) == 0,
		Errors:   errs,
		Warnings: warnings,
	}
}

func isSupportedOCIEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "unix", "tcp", "http", "https", "ssh":
		return true
	default:
		return false
	}
}

func isSupportedKubernetesEndpoint(raw string) bool {
	if raw == "in-cluster" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "https") && u.Host != ""
}
