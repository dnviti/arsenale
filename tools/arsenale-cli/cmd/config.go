package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	persistentCLITokenExpiry = "9999-12-31T23:59:59Z"
	arsenaleConfigEnv        = "ARSENALE_CONFIG"
	localConfigFile          = "config.yaml"
)

// CLIConfig holds the Arsenale CLI configuration.
type CLIConfig struct {
	ServerURL    string `yaml:"server_url"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	TokenExpiry  string `yaml:"token_expiry,omitempty"`
	TenantID     string `yaml:"tenant_id,omitempty"`
	CacheTTL     string `yaml:"cache_ttl,omitempty"`
	ConfigPath   string `yaml:"-"`
}

func configDir() string {
	return filepath.Dir(configPath())
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: cannot determine home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".arsenale", "config.yaml")
}

func configPath() string {
	return resolveConfigPath(configFlag)
}

func resolveConfigPath(flagValue string) string {
	if path := strings.TrimSpace(flagValue); path != "" {
		return path
	}
	if path := strings.TrimSpace(os.Getenv(arsenaleConfigEnv)); path != "" {
		return path
	}
	if path := existingLocalConfigPath(); path != "" {
		return path
	}
	return defaultConfigPath()
}

func existingLocalConfigPath() string {
	info, err := os.Stat(localConfigFile)
	if err != nil || info.IsDir() {
		return ""
	}
	path, err := filepath.Abs(localConfigFile)
	if err != nil {
		return localConfigFile
	}
	return path
}

func loadConfig() *CLIConfig {
	cfg, err := loadConfigFromPath(configPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to parse config file:", err)
	}
	return cfg
}

func loadConfigFromPath(path string) (*CLIConfig, error) {
	cfg := &CLIConfig{
		ServerURL:  "http://localhost:3001",
		CacheTTL:   "5m",
		ConfigPath: path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		cfg.ConfigPath = path
		return cfg, err
	}
	cfg.ConfigPath = path

	return cfg, nil
}

func saveConfig(cfg *CLIConfig) error {
	path := cfg.ConfigPath
	if strings.TrimSpace(path) == "" {
		path = configPath()
	}
	if err := saveConfigToPath(cfg, path); err != nil {
		return err
	}
	cfg.ConfigPath = path
	return nil
}

func saveConfigToPath(cfg *CLIConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
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
