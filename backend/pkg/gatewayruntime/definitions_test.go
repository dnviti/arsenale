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

func TestTunnelLocalPortPrefersConfiguredPort(t *testing.T) {
	if got := TunnelLocalPort(TypeGuacd, 14822); got != 14822 {
		t.Fatalf("TunnelLocalPort = %d, want 14822", got)
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
	want := []int{14822, 4822}
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
