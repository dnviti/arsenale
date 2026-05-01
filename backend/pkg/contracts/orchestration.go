package contracts

type OrchestratorConnectionKind string

const (
	OrchestratorDocker     OrchestratorConnectionKind = "docker"
	OrchestratorPodman     OrchestratorConnectionKind = "podman"
	OrchestratorKubernetes OrchestratorConnectionKind = "kubernetes"
)

type OrchestratorScope string

const (
	OrchestratorScopeGlobal OrchestratorScope = "global"
	OrchestratorScopeTenant OrchestratorScope = "tenant"
)

type OrchestratorConnection struct {
	ID           string                     `json:"id,omitempty"`
	Name         string                     `json:"name"`
	Kind         OrchestratorConnectionKind `json:"kind"`
	Scope        OrchestratorScope          `json:"scope"`
	Endpoint     string                     `json:"endpoint"`
	Namespace    string                     `json:"namespace,omitempty"`
	Labels       map[string]string          `json:"labels,omitempty"`
	Capabilities []string                   `json:"capabilities,omitempty"`
}
