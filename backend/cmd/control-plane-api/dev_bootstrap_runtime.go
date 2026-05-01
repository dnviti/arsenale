package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func runDevBootstrap(ctx context.Context, deps *apiDependencies) error {
	if deps == nil || deps.db == nil {
		return fmt.Errorf("bootstrap dependencies are unavailable")
	}

	options := loadDevBootstrapOptions()
	runtime := loadDevBootstrapRuntime()
	if err := ensureBootstrapSetup(ctx, deps, options); err != nil {
		return err
	}

	userID, err := lookupBootstrapUserID(ctx, deps, options.adminEmail)
	if err != nil {
		return err
	}
	if runtime.features.KeychainEnabled {
		if err := ensureBootstrapVaultUnlocked(ctx, deps, userID, options.adminPassword); err != nil {
			return err
		}
	}
	tenantID, err := ensureBootstrapTenant(ctx, deps, userID, options.tenantName)
	if err != nil {
		return err
	}
	if err := ensureBootstrapMembership(ctx, deps, tenantID, userID); err != nil {
		return err
	}
	if runtime.features.ConnectionsEnabled {
		if err := ensureBootstrapSSHKeyPair(ctx, deps, tenantID, userID); err != nil {
			return err
		}
	}

	specs := buildDevGatewaySpecs(options.certDir, runtime)
	if hasTunnelGateway(specs) {
		if err := syncTenantTunnelCA(ctx, deps, tenantID, options.certDir); err != nil {
			return err
		}
	}
	for _, spec := range specs {
		if err := upsertDevGateway(ctx, deps, tenantID, userID, spec); err != nil {
			return err
		}
	}
	if hasTunnelGateway(specs) {
		if err := ensureBootstrapOrchestratorConnection(ctx, deps, options); err != nil {
			return err
		}
	}

	if runtime.demoDatabasesEnabled && runtime.features.DatabaseProxyEnabled {
		if err := ensureDemoDatabaseConnections(ctx, deps, tenantID, userID); err != nil {
			return err
		}
	}

	if hasManagedSSHGateway(specs) {
		const maxManagedSSHKeyPushAttempts = 15
		const managedSSHKeyPushRetryDelay = 2 * time.Second
		keyPushSucceeded := false
		for attempt := 1; attempt <= maxManagedSSHKeyPushAttempts; attempt++ {
			pushResults, err := deps.gatewayService.PushSSHKeyToAllManagedGateways(ctx, tenantID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Managed SSH key push attempt %d/%d failed: %v\n", attempt, maxManagedSSHKeyPushAttempts, err)
			} else {
				allOK := len(pushResults) > 0
				for _, item := range pushResults {
					if item.OK {
						fmt.Printf("managed ssh key push ok: %s (%s)\n", item.Name, item.GatewayID)
						continue
					}
					allOK = false
					fmt.Fprintf(
						os.Stderr,
						"managed ssh key push pending: %s (%s): %s (attempt %d/%d)\n",
						item.Name,
						item.GatewayID,
						item.Error,
						attempt,
						maxManagedSSHKeyPushAttempts,
					)
				}
				if allOK {
					keyPushSucceeded = true
					break
				}
			}
			if attempt < maxManagedSSHKeyPushAttempts {
				time.Sleep(managedSSHKeyPushRetryDelay)
			}
		}
		if !keyPushSucceeded {
			return fmt.Errorf("managed SSH key push did not complete cleanly after %d attempts", maxManagedSSHKeyPushAttempts)
		}
	}

	fmt.Printf("development bootstrap complete for tenant %s\n", tenantID)
	for _, spec := range specs {
		fmt.Printf("  %s gateway: %s (%s)\n", spec.Type, spec.Name, spec.ID)
	}
	return nil
}

func loadDevBootstrapRuntime() devBootstrapRuntime {
	return devBootstrapRuntime{
		features:              runtimefeatures.FromEnv(),
		tunnelFixturesEnabled: requiredEnvBool("DEV_BOOTSTRAP_TUNNEL_FIXTURES_ENABLED", false),
		demoDatabasesEnabled:  requiredEnvBool("DEV_BOOTSTRAP_DEMO_DATABASES_ENABLED", false),
	}
}

func requiredEnvBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func hasManagedSSHGateway(specs []devGatewaySpec) bool {
	for _, spec := range specs {
		if spec.Type == "MANAGED_SSH" {
			return true
		}
	}
	return false
}

func hasTunnelGateway(specs []devGatewaySpec) bool {
	for _, spec := range specs {
		if spec.TunnelEnabled {
			return true
		}
	}
	return false
}

func loadDevBootstrapOptions() devBootstrapOptions {
	certDir := strings.TrimSpace(os.Getenv("DEV_TUNNEL_CERT_DIR"))
	if certDir == "" {
		certDir = "/certs"
	}
	return devBootstrapOptions{
		adminEmail:       requiredEnv("DEV_BOOTSTRAP_ADMIN_EMAIL", "admin@example.com"),
		adminPassword:    requiredEnv("DEV_BOOTSTRAP_ADMIN_PASSWORD", "ArsenaleTemp91Qx"),
		adminUsername:    requiredEnv("DEV_BOOTSTRAP_ADMIN_USERNAME", "admin"),
		tenantName:       requiredEnv("DEV_BOOTSTRAP_TENANT_NAME", "Development Environment"),
		certDir:          certDir,
		orchestratorName: requiredEnv("DEV_BOOTSTRAP_ORCHESTRATOR_NAME", "dev-podman"),
		orchestratorKind: parseBootstrapOrchestratorKind(
			requiredEnv("DEV_BOOTSTRAP_ORCHESTRATOR_KIND", string(contracts.OrchestratorPodman)),
		),
		orchestratorScope: parseBootstrapOrchestratorScope(
			requiredEnv("DEV_BOOTSTRAP_ORCHESTRATOR_SCOPE", string(contracts.OrchestratorScopeGlobal)),
		),
		orchestratorURL: requiredEnv("DEV_BOOTSTRAP_ORCHESTRATOR_ENDPOINT", "unix:///run/podman/podman.sock"),
	}
}

func requiredEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func requiredEnvInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBootstrapOrchestratorKind(value string) contracts.OrchestratorConnectionKind {
	switch contracts.OrchestratorConnectionKind(strings.ToLower(strings.TrimSpace(value))) {
	case contracts.OrchestratorDocker:
		return contracts.OrchestratorDocker
	case contracts.OrchestratorKubernetes:
		return contracts.OrchestratorKubernetes
	default:
		return contracts.OrchestratorPodman
	}
}

func parseBootstrapOrchestratorScope(value string) contracts.OrchestratorScope {
	switch contracts.OrchestratorScope(strings.ToLower(strings.TrimSpace(value))) {
	case contracts.OrchestratorScopeTenant:
		return contracts.OrchestratorScopeTenant
	default:
		return contracts.OrchestratorScopeGlobal
	}
}
