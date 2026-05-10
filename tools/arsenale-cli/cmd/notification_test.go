package cmd

import (
	"encoding/json"
	"testing"
)

func TestNormalizeNotificationPreferencePayloadMapsPushToInApp(t *testing.T) {
	body, err := normalizeNotificationPreferencePayload([]byte(`{"push": false, "email": true}`))
	if err != nil {
		t.Fatalf("normalizeNotificationPreferencePayload() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := got["push"]; ok {
		t.Fatal("push alias was not removed")
	}
	if got["inApp"] != false {
		t.Fatalf("inApp = %v, want false", got["inApp"])
	}
	if got["email"] != true {
		t.Fatalf("email = %v, want true", got["email"])
	}
}

func TestNormalizeNotificationPreferencesPayloadWrapsGetResponseArray(t *testing.T) {
	body, err := normalizeNotificationPreferencesPayload([]byte(`[
		{"type": "CONNECTION_SHARED", "push": false, "email": true},
		{"type": "SECRET_EXPIRING", "inApp": true, "email": false}
	]`))
	if err != nil {
		t.Fatalf("normalizeNotificationPreferencesPayload() error = %v", err)
	}

	var got struct {
		Preferences []map[string]any `json:"preferences"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Preferences) != 2 {
		t.Fatalf("len(preferences) = %d, want 2", len(got.Preferences))
	}
	if got.Preferences[0]["inApp"] != false {
		t.Fatalf("first inApp = %v, want false", got.Preferences[0]["inApp"])
	}
	if _, ok := got.Preferences[0]["push"]; ok {
		t.Fatal("push alias was not removed from wrapped preference")
	}
}
