package systemsettingsapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	loadRegistryOnce sync.Once
	loadRegistryErr  error
	registry         []SettingDef
	groups           []SettingGroup
)

func ensureRegistryLoaded() error {
	loadRegistryOnce.Do(func() {
		if err := json.Unmarshal([]byte(settingsRegistryJSON), &registry); err != nil {
			loadRegistryErr = fmt.Errorf("decode settings registry: %w", err)
			return
		}
		normalizeCallbackDefaults()
		if err := json.Unmarshal([]byte(settingGroupsJSON), &groups); err != nil {
			loadRegistryErr = fmt.Errorf("decode setting groups: %w", err)
			return
		}
	})
	return loadRegistryErr
}

func normalizeCallbackDefaults() {
	defaults := map[string]string{
		"GOOGLE_CALLBACK_URL":    publicEdgeCallbackURL("/api/auth/oauth/google/callback"),
		"MICROSOFT_CALLBACK_URL": publicEdgeCallbackURL("/api/auth/oauth/microsoft/callback"),
		"GITHUB_CALLBACK_URL":    publicEdgeCallbackURL("/api/auth/oauth/github/callback"),
		"OIDC_CALLBACK_URL":      publicEdgeCallbackURL("/api/auth/oauth/oidc/callback"),
		"SAML_CALLBACK_URL":      publicEdgeCallbackURL("/api/auth/saml/callback"),
	}
	for i := range registry {
		override, ok := defaults[registry[i].Key]
		if !ok {
			continue
		}
		current := strings.TrimSpace(fmt.Sprint(registry[i].Default))
		if current == "" || strings.Contains(current, "localhost:3000") || strings.Contains(current, "localhost:3001") {
			registry[i].Default = override
		}
	}
}

func publicEdgeCallbackURL(path string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("CLIENT_URL")), "/")
	if baseURL == "" {
		baseURL = "https://localhost:3000"
	}
	return baseURL + path
}

func loadedGroups() []SettingGroup {
	_ = ensureRegistryLoaded()
	out := make([]SettingGroup, len(groups))
	copy(out, groups)
	return out
}

func lookupDef(key string) (SettingDef, bool) {
	for _, def := range registry {
		if def.Key == key {
			return def, true
		}
	}
	return SettingDef{}, false
}
