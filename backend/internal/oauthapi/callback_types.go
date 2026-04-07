package oauthapi

type oauthProfile struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
}

type oauthProviderTokens struct {
	AccessToken  string
	RefreshToken string
}

type oauthLoginResult struct {
	UserID            string
	Email             string
	Username          *string
	AvatarData        *string
	NeedsVaultSetup   bool
	AllowlistDecision ipAllowlistDecision
}

type ipAllowlistDecision struct {
	Flagged bool
	Blocked bool
}

type tenantAllowlist struct {
	Enabled bool
	Mode    string
	Entries []string
}
