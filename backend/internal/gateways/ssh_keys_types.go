package gateways

import (
	"encoding/json"
	"strings"
	"time"
)

type sshKeyPairResponse struct {
	ID                   string     `json:"id"`
	PublicKey            string     `json:"publicKey"`
	Fingerprint          string     `json:"fingerprint"`
	Algorithm            string     `json:"algorithm"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	AutoRotateEnabled    bool       `json:"autoRotateEnabled"`
	RotationIntervalDays int        `json:"rotationIntervalDays"`
	LastAutoRotatedAt    *time.Time `json:"lastAutoRotatedAt"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type sshKeyRotationStatus struct {
	AutoRotateEnabled    bool       `json:"autoRotateEnabled"`
	RotationIntervalDays int        `json:"rotationIntervalDays"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	LastAutoRotatedAt    *time.Time `json:"lastAutoRotatedAt"`
	NextRotationDate     *time.Time `json:"nextRotationDate"`
	DaysUntilRotation    *int       `json:"daysUntilRotation"`
	KeyExists            bool       `json:"keyExists"`
}

type sshKeyPairRecord struct {
	ID                   string
	TenantID             string
	EncryptedPrivateKey  string
	PrivateKeyIV         string
	PrivateKeyTag        string
	PublicKey            string
	Fingerprint          string
	Algorithm            string
	ExpiresAt            *time.Time
	AutoRotateEnabled    bool
	RotationIntervalDays int
	LastAutoRotatedAt    *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type rotationPolicyPayload struct {
	AutoRotateEnabled    *bool        `json:"autoRotateEnabled"`
	RotationIntervalDays *int         `json:"rotationIntervalDays"`
	ExpiresAt            optionalTime `json:"expiresAt"`
}

type optionalTime struct {
	Present bool
	Value   *time.Time
}

func (o *optionalTime) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	parsed = parsed.UTC()
	o.Value = &parsed
	return nil
}
