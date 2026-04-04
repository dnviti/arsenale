package notifications

import (
	"testing"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
)

func TestAvailableNotificationTypesFiltersRecordingReadyWhenRecordingsDisabled(t *testing.T) {
	types := availableNotificationTypes(runtimefeatures.Manifest{RecordingsEnabled: false})
	for _, prefType := range types {
		if prefType == "RECORDING_READY" {
			t.Fatalf("RECORDING_READY should be hidden when recordings are disabled")
		}
	}
}

func TestNotificationTypeEnabledAllowsRecordingReadyWhenRecordingsEnabled(t *testing.T) {
	if !notificationTypeEnabled(runtimefeatures.Manifest{RecordingsEnabled: true}, "RECORDING_READY") {
		t.Fatalf("expected RECORDING_READY to remain available when recordings are enabled")
	}
}

func TestNotificationTypeEnabledRejectsUnknownTypes(t *testing.T) {
	if notificationTypeEnabled(runtimefeatures.Manifest{RecordingsEnabled: true}, "NOPE") {
		t.Fatalf("unexpected unknown notification type to be accepted")
	}
}
