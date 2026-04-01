package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CLIConfig holds the Arsenale CLI configuration.
type CLIConfig struct {
	ServerURL    string `yaml:"server_url"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	TokenExpiry  string `yaml:"token_expiry,omitempty"`
	TenantID     string `yaml:"tenant_id,omitempty"`
	CacheTTL     string `yaml:"cache_ttl,omitempty"`
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: cannot determine home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".arsenale")
}

func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func loadConfig() *CLIConfig {
	cfg := &CLIConfig{
		ServerURL: "http://localhost:3001",
		CacheTTL:  "5m",
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to parse config file:", err)
	}

	return cfg
}

func saveConfig(cfg *CLIConfig) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(configPath(), data, 0600)
}

func (c *CLIConfig) isTokenValid() bool {
	if c.AccessToken == "" || c.TokenExpiry == "" {
		return false
	}
	expiry, err := time.Parse(time.RFC3339, c.TokenExpiry)
	if err != nil {
		return false
	}
	return time.Now().Before(expiry.Add(-30 * time.Second))
}

// resolveTenantID returns the tenant ID from flag, config, or fetches it.
func (c *CLIConfig) resolveTenantID() string {
	if c.TenantID != "" {
		return c.TenantID
	}

	// Try to fetch from API
	body, status, err := apiGet("/api/tenants/mine", c)
	if err != nil || status != 200 {
		return ""
	}

	var tenant struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &tenant); err != nil {
		return ""
	}

	// Cache it
	c.TenantID = tenant.ID
	_ = saveConfig(c)
	return tenant.ID
}
