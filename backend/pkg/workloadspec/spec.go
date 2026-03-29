package workloadspec

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type Port struct {
	Name      string `json:"name,omitempty"`
	Container int    `json:"container"`
	Host      int    `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type Healthcheck struct {
	Command     []string `json:"command"`
	IntervalSec int      `json:"intervalSec"`
	TimeoutSec  int      `json:"timeoutSec"`
	Retries     int      `json:"retries"`
	StartPeriod int      `json:"startPeriodSec,omitempty"`
}

type Volume struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type ResourceRequirements struct {
	CPURequestMillicores int `json:"cpuRequestMillicores,omitempty"`
	CPULimitMillicores   int `json:"cpuLimitMillicores,omitempty"`
	MemoryRequestMiB     int `json:"memoryRequestMiB,omitempty"`
	MemoryLimitMiB       int `json:"memoryLimitMiB,omitempty"`
}

type OCIOptions struct {
	Network string `json:"network,omitempty"`
	User    string `json:"user,omitempty"`
}

type KubernetesOptions struct {
	Namespace      string `json:"namespace,omitempty"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	IngressClass   string `json:"ingressClass,omitempty"`
}

type WorkloadSpec struct {
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Env         map[string]string    `json:"env,omitempty"`
	Labels      map[string]string    `json:"labels,omitempty"`
	Ports       []Port               `json:"ports,omitempty"`
	Volumes     []Volume             `json:"volumes,omitempty"`
	Resources   ResourceRequirements `json:"resources,omitempty"`
	Healthcheck *Healthcheck         `json:"healthcheck,omitempty"`
	OCI         OCIOptions           `json:"oci,omitempty"`
	Kubernetes  KubernetesOptions    `json:"kubernetes,omitempty"`
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

var validEnvKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var validWorkloadName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func (s WorkloadSpec) ValidateFor(kind contracts.OrchestratorConnectionKind) ValidationResult {
	var errs []string
	var warnings []string

	if !validWorkloadName.MatchString(s.Name) {
		errs = append(errs, "name must be lowercase DNS-safe and 1-63 chars long")
	}
	if strings.TrimSpace(s.Image) == "" {
		errs = append(errs, "image is required")
	}

	for key := range s.Env {
		if !validEnvKey.MatchString(key) {
			errs = append(errs, fmt.Sprintf("env key %q must match %s", key, validEnvKey.String()))
		}
	}

	seenPorts := make(map[string]struct{}, len(s.Ports))
	for _, port := range s.Ports {
		if port.Container <= 0 || port.Container > 65535 {
			errs = append(errs, fmt.Sprintf("container port %d is invalid", port.Container))
		}
		if port.Host < 0 || port.Host > 65535 {
			errs = append(errs, fmt.Sprintf("host port %d is invalid", port.Host))
		}
		protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
		if protocol == "" {
			protocol = "tcp"
		}
		if !slices.Contains([]string{"tcp", "udp"}, protocol) {
			errs = append(errs, fmt.Sprintf("port protocol %q is unsupported", port.Protocol))
		}
		key := fmt.Sprintf("%d/%s", port.Container, protocol)
		if _, ok := seenPorts[key]; ok {
			errs = append(errs, fmt.Sprintf("duplicate container port mapping for %s", key))
		}
		seenPorts[key] = struct{}{}
	}

	for _, volume := range s.Volumes {
		if strings.TrimSpace(volume.Name) == "" {
			errs = append(errs, "volume name is required")
		}
		if strings.TrimSpace(volume.Source) == "" {
			errs = append(errs, fmt.Sprintf("volume %q source is required", volume.Name))
		}
		if !strings.HasPrefix(volume.MountPath, "/") {
			errs = append(errs, fmt.Sprintf("volume %q mountPath must be absolute", volume.Name))
		}
	}

	if s.Healthcheck != nil {
		if len(s.Healthcheck.Command) == 0 {
			errs = append(errs, "healthcheck command is required")
		}
		if s.Healthcheck.IntervalSec <= 0 {
			errs = append(errs, "healthcheck intervalSec must be greater than zero")
		}
		if s.Healthcheck.TimeoutSec <= 0 {
			errs = append(errs, "healthcheck timeoutSec must be greater than zero")
		}
		if s.Healthcheck.Retries <= 0 {
			errs = append(errs, "healthcheck retries must be greater than zero")
		}
	}

	if s.Resources.CPURequestMillicores > 0 && s.Resources.CPULimitMillicores > 0 && s.Resources.CPURequestMillicores > s.Resources.CPULimitMillicores {
		errs = append(errs, "cpu request cannot exceed cpu limit")
	}
	if s.Resources.MemoryRequestMiB > 0 && s.Resources.MemoryLimitMiB > 0 && s.Resources.MemoryRequestMiB > s.Resources.MemoryLimitMiB {
		errs = append(errs, "memory request cannot exceed memory limit")
	}

	switch kind {
	case contracts.OrchestratorDocker, contracts.OrchestratorPodman:
		if s.Kubernetes.Namespace != "" || s.Kubernetes.ServiceAccount != "" || s.Kubernetes.IngressClass != "" {
			errs = append(errs, "kubernetes-specific options are not valid for OCI runtimes")
		}
		if s.OCI.Network == "" {
			warnings = append(warnings, "oci.network is empty; default bridge/networking behavior will be used")
		}
	case contracts.OrchestratorKubernetes:
		if s.OCI.Network != "" || s.OCI.User != "" {
			errs = append(errs, "oci-specific options are not valid for kubernetes")
		}
		if s.Kubernetes.Namespace == "" {
			warnings = append(warnings, "kubernetes.namespace is empty; controller default namespace will be used")
		}
	default:
		errs = append(errs, fmt.Sprintf("unsupported orchestrator kind %q", kind))
	}

	return ValidationResult{
		Valid:    len(errs) == 0,
		Errors:   errs,
		Warnings: warnings,
	}
}
