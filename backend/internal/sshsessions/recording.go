package sshsessions

import (
	"context"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/sessionrecording"
)

func (s Service) maybeStartSessionRecording(ctx context.Context, tenantID, userID, connectionID, protocol, gatewayDir string) (*sessionrecording.Reference, error) {
	if !s.RecordingEnabled {
		return nil, nil
	}

	enabled, err := sessionrecording.TenantRecordingEnabled(ctx, s.DB, tenantID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}

	ref, err := sessionrecording.StartAsciicastRecording(ctx, s.DB, s.recordingRoot(), userID, connectionID, protocol, gatewayDir, "", 80, 24)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (s Service) deleteSessionRecording(ctx context.Context, ref sessionrecording.Reference) error {
	return sessionrecording.DeleteRecording(ctx, s.DB, ref)
}

func recordingGatewayDir(gatewayID, instanceID string) string {
	if strings.TrimSpace(instanceID) != "" {
		return strings.TrimSpace(instanceID)
	}
	if strings.TrimSpace(gatewayID) != "" {
		return strings.TrimSpace(gatewayID)
	}
	return "default"
}

func recordingID(ref *sessionrecording.Reference) string {
	if ref == nil {
		return ""
	}
	return strings.TrimSpace(ref.ID)
}

func recordingMetadata(ref sessionrecording.Reference) map[string]any {
	return sessionrecording.MetadataFromReference(ref)
}

func recordingTokenMetadata(ref sessionrecording.Reference) map[string]string {
	return sessionrecording.MetadataStringsFromReference(ref)
}

func mergeStringMaps(base, extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return base
	}
	if base == nil {
		base = map[string]string{}
	}
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func (s Service) recordingRoot() string {
	root := strings.TrimSpace(s.RecordingPath)
	if root == "" {
		return "/recordings"
	}
	return root
}
