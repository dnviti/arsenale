package oauthapi

import (
	"sync"
	"time"
)

type samlRequestEntry struct {
	RequestID  string `json:"requestId"`
	LinkUserID string `json:"linkUserId,omitempty"`
	ExpiresAt  int64  `json:"expiresAt"`
}

var (
	samlRequestMu    sync.Mutex
	samlRequestStore = map[string]samlRequestEntry{}
)

const samlRequestTTL = 5 * time.Minute
