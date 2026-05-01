package syncprofiles

type syncTestResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type discoveredDevice struct {
	ExternalID  string         `json:"externalId"`
	Name        string         `json:"name"`
	Host        string         `json:"host"`
	Port        int            `json:"port"`
	Protocol    string         `json:"protocol"`
	SiteName    string         `json:"siteName,omitempty"`
	RackName    string         `json:"rackName,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata"`
}

type syncPlan struct {
	ToCreate []discoveredDevice   `json:"toCreate"`
	ToUpdate []syncPlanUpdateItem `json:"toUpdate"`
	ToSkip   []syncPlanSkipItem   `json:"toSkip"`
	Errors   []syncPlanErrorItem  `json:"errors"`
}

type syncPlanUpdateItem struct {
	Device       discoveredDevice `json:"device"`
	ConnectionID string           `json:"connectionId"`
	Changes      []string         `json:"changes"`
}

type syncPlanSkipItem struct {
	Device discoveredDevice `json:"device"`
	Reason string           `json:"reason"`
}

type syncPlanErrorItem struct {
	Device discoveredDevice `json:"device"`
	Error  string           `json:"error"`
}

type syncPreviewResponse struct {
	Plan syncPlan `json:"plan"`
}

type syncResultError struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Error      string `json:"error"`
}

type syncResultResponse struct {
	Created int               `json:"created"`
	Updated int               `json:"updated"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Errors  []syncResultError `json:"errors"`
}

type triggerSyncResponse struct {
	Plan   syncPlan            `json:"plan"`
	Result *syncResultResponse `json:"result,omitempty"`
}

type syncProfileRuntime struct {
	Profile           syncProfileResponse
	EncryptedAPIToken string
	APITokenIV        string
	APITokenTag       string
}

type netBoxPaginatedResponse[T any] struct {
	Count   int     `json:"count"`
	Next    *string `json:"next"`
	Results []T     `json:"results"`
}

type netBoxIP struct {
	Address string `json:"address"`
	Family  struct {
		Value int `json:"value"`
	} `json:"family"`
}

type netBoxPlatform struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type netBoxNamed struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type netBoxRack struct {
	Name string `json:"name"`
}

type netBoxStatus struct {
	Value string `json:"value"`
}

type netBoxDevice struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Display      string          `json:"display"`
	PrimaryIP4   *netBoxIP       `json:"primary_ip4"`
	PrimaryIP6   *netBoxIP       `json:"primary_ip6"`
	Platform     *netBoxPlatform `json:"platform"`
	Site         *netBoxNamed    `json:"site"`
	Rack         *netBoxRack     `json:"rack"`
	Location     *netBoxRack     `json:"location"`
	Status       *netBoxStatus   `json:"status"`
	Description  string          `json:"description"`
	CustomFields map[string]any  `json:"custom_fields"`
}

type netBoxVM struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Display      string          `json:"display"`
	PrimaryIP4   *netBoxIP       `json:"primary_ip4"`
	PrimaryIP6   *netBoxIP       `json:"primary_ip6"`
	Platform     *netBoxPlatform `json:"platform"`
	Site         *netBoxNamed    `json:"site"`
	Cluster      *netBoxRack     `json:"cluster"`
	Status       *netBoxStatus   `json:"status"`
	Description  string          `json:"description"`
	CustomFields map[string]any  `json:"custom_fields"`
}
