package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
)

const (
	tunnelClientCertPath = "./certs/tunnel-client-cert.pem"
	tunnelClientKeyPath  = "./certs/tunnel-client-key.pem"
	guacdServiceCertPath = "./certs/guacd-server-cert.pem"
	guacdServiceKeyPath  = "./certs/guacd-server-key.pem"
	guacdServiceCAPath   = "./certs/guacd-ca.pem"
)

type tunnelBundleRuntime struct {
	gatewayType        string
	serviceName        string
	image              string
	listenerEnv        string
	containerUID       int
	containerGID       int
	serviceTLSRequired bool
	extraEnvironment   []string
	volumes            []string
}

func writeTunnelComposeBundle(bundleDir string, bundle tunnelTokenBundle, envContent string) error {
	runtime, err := tunnelRuntimeForBundle(bundle)
	if err != nil {
		return err
	}

	if runtime.serviceTLSRequired {
		if err := writeBundleFile(filepath.Join(bundleDir, "certs", "guacd-server-cert.pem"), bundle.TunnelServiceCert, 0644); err != nil {
			return err
		}
		if err := writeBundleFile(filepath.Join(bundleDir, "certs", "guacd-server-key.pem"), bundle.TunnelServiceKey, 0600); err != nil {
			return err
		}
		if err := writeBundleFile(filepath.Join(bundleDir, "certs", "guacd-ca.pem"), bundle.TunnelServiceCACert, 0644); err != nil {
			return err
		}
	}

	composePath := filepath.Join(bundleDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(tunnelDockerCompose(runtime)), 0600); err != nil {
		return fmt.Errorf("write tunnel docker-compose.yml: %w", err)
	}
	if err := os.Chmod(composePath, 0600); err != nil {
		return fmt.Errorf("chmod tunnel docker-compose.yml: %w", err)
	}

	installPath := filepath.Join(bundleDir, "install.sh")
	if err := os.WriteFile(installPath, []byte(tunnelInstallScript(bundle, runtime, envContent, tunnelDockerCompose(runtime))), 0700); err != nil {
		return fmt.Errorf("write tunnel install script: %w", err)
	}
	if err := os.Chmod(installPath, 0700); err != nil {
		return fmt.Errorf("chmod tunnel install script: %w", err)
	}
	return nil
}

func tunnelRuntimeForBundle(bundle tunnelTokenBundle) (tunnelBundleRuntime, error) {
	gatewayType := gatewayruntime.NormalizeType(bundle.GatewayType)
	def, ok := gatewayruntime.Lookup(gatewayType)
	if !ok {
		return tunnelBundleRuntime{}, fmt.Errorf("unsupported gateway type %q in tunnel token response", bundle.GatewayType)
	}
	uid, gid := gatewayruntime.ContainerUser(gatewayType)
	runtime := tunnelBundleRuntime{
		gatewayType:        gatewayType,
		serviceName:        def.ComposeService,
		image:              def.StableImage,
		listenerEnv:        def.ListenerEnvVar,
		containerUID:       uid,
		containerGID:       gid,
		serviceTLSRequired: def.ServiceTLSRequired,
	}
	if runtime.serviceName == "" || runtime.image == "" {
		return tunnelBundleRuntime{}, fmt.Errorf("gateway type %q is missing compose runtime metadata", gatewayType)
	}
	if gatewayType == gatewayruntime.TypeGuacd {
		runtime.extraEnvironment = []string{
			`GUACD_SSL: "true"`,
			`GUACD_SSL_CERT: /certs/guacd-server-cert.pem`,
			`GUACD_SSL_KEY: /certs/guacd-server-key.pem`,
		}
		runtime.volumes = []string{
			"guacd-drive:/guacd-drive",
			"guacd-recordings:/recordings",
			guacdServiceCertPath + ":/certs/guacd-server-cert.pem:ro",
			guacdServiceKeyPath + ":/certs/guacd-server-key.pem:ro",
		}
	}
	return runtime, nil
}

func tunnelDockerCompose(runtime tunnelBundleRuntime) string {
	volumeLines := []string{
		tunnelClientCertPath + ":/tunnel-certs/client-cert.pem:ro",
		tunnelClientKeyPath + ":/tunnel-certs/client-key.pem:ro",
	}
	volumeLines = append(volumeLines, runtime.volumes...)

	lines := []string{
		"services:",
		"  " + runtime.serviceName + ":",
		"    image: " + runtime.image,
		"    pull_policy: always",
		`    user: "0:0"`,
		"    restart: unless-stopped",
		"    env_file:",
		"      - tunnel.env",
	}
	if len(runtime.extraEnvironment) > 0 {
		lines = append(lines, "    environment:")
		for _, line := range runtime.extraEnvironment {
			lines = append(lines, "      "+line)
		}
	}
	lines = append(lines, "    volumes:")
	for _, line := range volumeLines {
		lines = append(lines, "      - "+line)
	}

	if runtime.gatewayType == gatewayruntime.TypeGuacd {
		lines = append(lines, "", "volumes:", "  guacd-drive:", "  guacd-recordings:")
	}
	return strings.Join(lines, "\n") + "\n"
}

func tunnelInstallScript(bundle tunnelTokenBundle, runtime tunnelBundleRuntime, envContent string, dockerCompose string) string {
	privateFiles := []string{tunnelClientKeyPath}
	if runtime.serviceTLSRequired {
		privateFiles = append(privateFiles, guacdServiceKeyPath)
	}

	lines := []string{
		"#!/usr/bin/env sh",
		"set -eu",
		"umask 077",
		"mkdir -p ./certs",
	}
	lines = append(lines, writeHereDoc(tunnelClientCertPath, bundle.TunnelClientCert)...)
	lines = append(lines, writeHereDoc(tunnelClientKeyPath, bundle.TunnelClientKey)...)
	lines = append(lines, "chmod 644 "+tunnelClientCertPath, "chmod 600 "+tunnelClientKeyPath)
	if runtime.serviceTLSRequired {
		lines = append(lines, writeHereDoc(guacdServiceCertPath, bundle.TunnelServiceCert)...)
		lines = append(lines, writeHereDoc(guacdServiceKeyPath, bundle.TunnelServiceKey)...)
		lines = append(lines, writeHereDoc(guacdServiceCAPath, bundle.TunnelServiceCACert)...)
		lines = append(lines, "chmod 644 "+guacdServiceCertPath, "chmod 600 "+guacdServiceKeyPath, "chmod 644 "+guacdServiceCAPath)
	}
	lines = append(lines,
		`if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then`,
		`  compose_cmd="docker compose"`,
		`elif command -v podman-compose >/dev/null 2>&1; then`,
		`  compose_cmd="podman-compose"`,
		`else`,
		`  echo "docker compose or podman-compose is required" >&2`,
		`  exit 1`,
		`fi`,
	)
	if len(privateFiles) > 0 {
		lines = append(lines,
			`if [ "$compose_cmd" = "podman-compose" ] && command -v podman >/dev/null 2>&1; then`,
		)
		for _, path := range privateFiles {
			lines = append(lines,
				"  podman unshare chown "+strconv.Itoa(runtime.containerUID)+":"+strconv.Itoa(runtime.containerGID)+" "+path,
				"  podman unshare chmod 600 "+path,
			)
		}
		lines = append(lines, "fi")
	}
	if runtime.serviceTLSRequired {
		lines = append(lines,
			`if command -v openssl >/dev/null 2>&1; then`,
			"  openssl verify -CAfile "+guacdServiceCAPath+" "+guacdServiceCertPath+" >/dev/null",
			`fi`,
		)
	}
	lines = append(lines, writeHereDoc("tunnel.env", envContent)...)
	lines = append(lines, "chmod 600 tunnel.env")
	lines = append(lines, writeHereDoc("docker-compose.yml", dockerCompose)...)
	lines = append(lines, "$compose_cmd --env-file tunnel.env up -d")
	return strings.Join(lines, "\n") + "\n"
}

func writeHereDoc(path, value string) []string {
	return []string{
		"cat > " + path + " <<'EOF'",
		strings.TrimSpace(value),
		"EOF",
	}
}

func writeBundleFile(path, value string, mode os.FileMode) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("missing payload for %s", path)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}
