package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathPrefersFlagThenEnvThenLocalThenDefault(t *testing.T) {
	home := t.TempDir()
	workdir := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(workdir)
	t.Setenv(arsenaleConfigEnv, filepath.Join(t.TempDir(), "env.yaml"))

	flagPath := filepath.Join(t.TempDir(), "flag.yaml")
	if got := resolveConfigPath(flagPath); got != flagPath {
		t.Fatalf("resolveConfigPath(flag) = %q, want %q", got, flagPath)
	}

	envPath := os.Getenv(arsenaleConfigEnv)
	if got := resolveConfigPath(""); got != envPath {
		t.Fatalf("resolveConfigPath(env) = %q, want %q", got, envPath)
	}

	t.Setenv(arsenaleConfigEnv, "")
	localPath := filepath.Join(workdir, localConfigFile)
	if err := os.WriteFile(localPath, []byte("server_url: https://local.example\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if got := resolveConfigPath(""); got != localPath {
		t.Fatalf("resolveConfigPath(local) = %q, want %q", got, localPath)
	}
	if err := os.Remove(localPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	wantDefault := filepath.Join(home, ".arsenale", "config.yaml")
	if got := resolveConfigPath(""); got != wantDefault {
		t.Fatalf("resolveConfigPath(default) = %q, want %q", got, wantDefault)
	}
}

func TestSaveAndLoadConfigFromExplicitPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent", "arsenale.yaml")
	cfg := &CLIConfig{
		ServerURL:    "https://arsenale.example",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenExpiry:  persistentCLITokenExpiry,
		TenantID:     "tenant-1",
		CacheTTL:     "10m",
		ConfigPath:   path,
	}

	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %v, want 0600", got)
	}

	loaded, err := loadConfigFromPath(path)
	if err != nil {
		t.Fatalf("loadConfigFromPath() error = %v", err)
	}
	if loaded.ConfigPath != path {
		t.Fatalf("ConfigPath = %q, want %q", loaded.ConfigPath, path)
	}
	if loaded.ServerURL != cfg.ServerURL || loaded.RefreshToken != cfg.RefreshToken || loaded.TokenExpiry != cfg.TokenExpiry {
		t.Fatalf("loaded config = %+v, want matching saved config", loaded)
	}
}

func TestAdoptConfigRefreshedByAnotherProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arsenale.yaml")
	diskCfg := &CLIConfig{
		ServerURL:    "https://arsenale.example",
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		TokenExpiry:  persistentCLITokenExpiry,
		TenantID:     "tenant-1",
		CacheTTL:     "15m",
		ConfigPath:   path,
	}
	if err := saveConfig(diskCfg); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	cfg := &CLIConfig{
		ServerURL:    "https://arsenale.example",
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ConfigPath:   path,
	}
	adopted, err := adoptConfigRefreshedByAnotherProcess(cfg)
	if err != nil {
		t.Fatalf("adoptConfigRefreshedByAnotherProcess() error = %v", err)
	}
	if !adopted {
		t.Fatal("expected config refresh to be adopted")
	}
	if cfg.AccessToken != "new-access" || cfg.RefreshToken != "new-refresh" || cfg.TenantID != "tenant-1" {
		t.Fatalf("adopted config = %+v", cfg)
	}
}
