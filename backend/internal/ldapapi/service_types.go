package ldapapi

import "github.com/jackc/pgx/v5/pgxpool"

type Service struct {
	DB *pgxpool.Pool
}

type statusResponse struct {
	Enabled       bool   `json:"enabled"`
	ProviderName  string `json:"providerName"`
	ServerURL     string `json:"serverUrl"`
	BaseDN        string `json:"baseDn"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncCron      string `json:"syncCron"`
	AutoProvision bool   `json:"autoProvision"`
}

type LdapTestResult struct {
	Ok         bool   `json:"ok"`
	Message    string `json:"message"`
	UserCount  int    `json:"userCount,omitempty"`
	GroupCount int    `json:"groupCount,omitempty"`
}

type LdapSyncResult struct {
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Disabled int      `json:"disabled"`
	Errors   []string `json:"errors"`
}

type ldapConfig struct {
	Enabled               bool
	ProviderName          string
	ServerURL             string
	BaseDN                string
	BindDN                string
	BindPassword          string
	UserSearchFilter      string
	UserSearchBase        string
	DisplayNameAttr       string
	EmailAttr             string
	UIDAttr               string
	GroupBaseDN           string
	GroupSearchFilter     string
	GroupMemberAttr       string
	GroupNameAttr         string
	AllowedGroups         []string
	StartTLS              bool
	TLSRejectUnauthorized bool
	SyncEnabled           bool
	SyncCron              string
	AutoProvision         bool
	DefaultTenantID       string
}

type ldapUserEntry struct {
	DN             string
	UID            string
	Email          string
	DisplayName    string
	Groups         []string
	ProviderUserID string
}

func (c ldapConfig) isEnabled() bool {
	return c.Enabled && c.ServerURL != "" && c.BaseDN != ""
}
