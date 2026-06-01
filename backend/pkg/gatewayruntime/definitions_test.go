package gatewayruntime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDefinitionsCoverGatewayRuntimePorts(t *testing.T) {
	tests := []struct {
		gatewayType string
		managed     bool
		port        int
		listenerEnv string
	}{
		{TypeGuacd, true, 4822, "GUACD_PORT"},
		{TypeSSHBastion, false, 2222, "SSH_PORT"},
		{TypeManagedSSH, true, 2222, "SSH_PORT"},
		{TypeDBProxy, true, 5432, "DB_LISTEN_PORT"},
	}

	for _, tt := range tests {
		t.Run(tt.gatewayType, func(t *testing.T) {
			def, ok := Lookup(tt.gatewayType)
			if !ok {
				t.Fatal("definition not found")
			}
			if def.Managed != tt.managed {
				t.Fatalf("Managed = %v, want %v", def.Managed, tt.managed)
			}
			if def.PrimaryPort != tt.port {
				t.Fatalf("PrimaryPort = %d, want %d", def.PrimaryPort, tt.port)
			}
			if got := PrimaryPort(tt.gatewayType); got != tt.port {
				t.Fatalf("PrimaryPort(%q) = %d, want %d", tt.gatewayType, got, tt.port)
			}
			if got := ListenerEnvVar(tt.gatewayType); got != tt.listenerEnv {
				t.Fatalf("ListenerEnvVar(%q) = %q, want %q", tt.gatewayType, got, tt.listenerEnv)
			}
		})
	}
}

func TestTunnelLocalPortPrefersRuntimePort(t *testing.T) {
	if got := TunnelLocalPort(TypeGuacd, 14822); got != 4822 {
		t.Fatalf("TunnelLocalPort = %d, want 4822", got)
	}
	if got := TunnelLocalPort(TypeDBProxy, 0); got != 5432 {
		t.Fatalf("TunnelLocalPort fallback = %d, want 5432", got)
	}
	if got := TunnelLocalPort("OTHER", 15432); got != 15432 {
		t.Fatalf("TunnelLocalPort unknown type = %d, want 15432", got)
	}
}

func TestTunnelLocalPortCandidatesIncludeRuntimeFallback(t *testing.T) {
	got := TunnelLocalPortCandidates(TypeGuacd, 14822)
	want := []int{4822, 14822}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d = %d, want %d: %#v", i, got[i], want[i], got)
		}
	}

	got = TunnelLocalPortCandidates(TypeGuacd, 4822)
	if len(got) != 1 || got[0] != 4822 {
		t.Fatalf("default candidate list = %#v, want [4822]", got)
	}
}

func TestCatalogHasCompleteMetadata(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != len(definitions) {
		t.Fatalf("catalog size = %d, want %d", len(catalog), len(definitions))
	}
	for _, info := range catalog {
		if strings.TrimSpace(info.DisplayName) == "" {
			t.Errorf("%s: empty DisplayName", info.Type)
		}
		if strings.TrimSpace(info.Summary) == "" {
			t.Errorf("%s: empty Summary", info.Type)
		}
		if strings.TrimSpace(info.Description) == "" {
			t.Errorf("%s: empty Description", info.Type)
		}
		if len(info.Protocols) == 0 {
			t.Errorf("%s: no Protocols", info.Type)
		}
		if len(info.DeploymentModes) == 0 {
			t.Errorf("%s: no DeploymentModes", info.Type)
		}
		// Managed types must allow the managed group mode; self-hosted must not.
		hasGroup := false
		for _, m := range info.DeploymentModes {
			if m == DeploymentManagedGroup {
				hasGroup = true
			}
		}
		if info.Managed != hasGroup {
			t.Errorf("%s: Managed=%v but MANAGED_GROUP allowed=%v (must match)", info.Type, info.Managed, hasGroup)
		}
		wantModel := DeploymentModelSelfHosted
		if info.Managed {
			wantModel = DeploymentModelManaged
		}
		if info.DeploymentModel != wantModel {
			t.Errorf("%s: DeploymentModel=%q want %q", info.Type, info.DeploymentModel, wantModel)
		}
		// Only managed types are deployed by Arsenale, so only they advertise an image.
		if info.Managed && info.Image == "" {
			t.Errorf("%s: managed type missing image", info.Type)
		}
		if !info.Managed && info.Image != "" {
			t.Errorf("%s: self-hosted type should not advertise an image, got %q", info.Type, info.Image)
		}
	}

	if _, ok := Info(TypeGuacd); !ok {
		t.Fatal("Info(GUACD) not found")
	}
	if _, ok := Info("NOPE"); ok {
		t.Fatal("Info(unknown) should be false")
	}
	if DisplayName(TypeSSHBastion) != "SSH Bastion (Jump Host)" {
		t.Fatalf("DisplayName(SSH_BASTION) = %q", DisplayName(TypeSSHBastion))
	}
	if DisplayName("UNKNOWN_TYPE") != "UNKNOWN_TYPE" {
		t.Fatalf("DisplayName(unknown) = %q, want raw code", DisplayName("UNKNOWN_TYPE"))
	}
	// SSH_BASTION is the only type that takes credentials on the gateway.
	if info, _ := Info(TypeSSHBastion); !info.RequiresCredentials {
		t.Error("SSH_BASTION should require credentials")
	}
	if info, _ := Info(TypeManagedSSH); info.RequiresCredentials {
		t.Error("MANAGED_SSH should not require credentials")
	}
}

func TestLookupNormalizesType(t *testing.T) {
	def, ok := Lookup(" managed_ssh ")
	if !ok {
		t.Fatal("definition not found")
	}
	if def.Type != TypeManagedSSH {
		t.Fatalf("Type = %q, want %q", def.Type, TypeManagedSSH)
	}
}

func TestDefinitionsMatchStandaloneComposeFiles(t *testing.T) {
	seenDirs := map[string]bool{}
	for _, def := range All() {
		if seenDirs[def.StandaloneDirectory] {
			continue
		}
		seenDirs[def.StandaloneDirectory] = true

		composePath := filepath.Join("..", "..", "..", def.StandaloneDirectory, "compose.yml")
		contents, err := os.ReadFile(composePath)
		if err != nil {
			t.Fatalf("read %s: %v", composePath, err)
		}
		text := string(contents)
		if !strings.Contains(text, def.ComposeService+":") {
			t.Fatalf("%s does not define service %q", composePath, def.ComposeService)
		}
		wantPort := `TUNNEL_LOCAL_PORT: "${TUNNEL_LOCAL_PORT:-` + strconv.Itoa(def.PrimaryPort) + `}"`
		if !strings.Contains(text, wantPort) {
			t.Fatalf("%s missing %s", composePath, wantPort)
		}
		if def.ListenerEnvVar != "" {
			wantListenerPort := def.ListenerEnvVar + `: "${` + def.ListenerEnvVar + `:-` + strconv.Itoa(def.PrimaryPort) + `}"`
			if !strings.Contains(text, wantListenerPort) {
				t.Fatalf("%s missing %s", composePath, wantListenerPort)
			}
		}
	}
}
