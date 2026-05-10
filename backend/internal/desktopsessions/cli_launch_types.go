package desktopsessions

import "time"

type desktopLaunchRequest struct {
	Protocol     string `json:"protocol"`
	ConnectionID string `json:"connectionId"`
}

type desktopLaunchResponse struct {
	Protocol     string    `json:"protocol"`
	ConnectionID string    `json:"connectionId"`
	LaunchURL    string    `json:"launchUrl"`
	ExpiresAt    time.Time `json:"expiresAt"`
	ExpiresIn    int       `json:"expiresIn"`
}

type desktopLaunchRedeemRequest struct {
	Grant string `json:"grant"`
}

type desktopLaunchRedeemResponse struct {
	Protocol              string      `json:"protocol"`
	ConnectionID          string      `json:"connectionId"`
	SessionID             string      `json:"sessionId"`
	Token                 string      `json:"token"`
	WebSocketPath         string      `json:"webSocketPath"`
	ControlToken          string      `json:"controlToken"`
	ControlTokenExpiresAt time.Time   `json:"controlTokenExpiresAt"`
	EnableDrive           bool        `json:"enableDrive,omitempty"`
	RecordingID           string      `json:"recordingId,omitempty"`
	DLPPolicy             resolvedDLP `json:"dlpPolicy"`
	ResolvedUsername      string      `json:"resolvedUsername,omitempty"`
	ResolvedDomain        string      `json:"resolvedDomain,omitempty"`
}

type desktopViewerControlRequest struct {
	ControlToken string `json:"controlToken"`
}

type desktopLaunchGrantRecord struct {
	ID           string
	TenantID     string
	UserID       string
	ConnectionID string
	Protocol     string
	SecretHash   string
	ExpiresAt    time.Time
	Consumed     bool
}

type desktopViewerControlRecord struct {
	ID        string
	TenantID  string
	UserID    string
	SessionID string
	Protocol  string
}
