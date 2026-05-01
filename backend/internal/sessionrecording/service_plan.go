package sessionrecording

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func buildRecordingPlan(recordingRoot, userID, connectionID, protocol, ext, gatewayDir string, now time.Time) (recordingPlan, error) {
	recordingRoot = strings.TrimSpace(recordingRoot)
	if recordingRoot == "" {
		return recordingPlan{}, fmt.Errorf("recording path is not configured")
	}

	subdir := strings.TrimSpace(gatewayDir)
	if subdir == "" {
		subdir = defaultGatewayDir
	}

	components := []struct {
		label string
		value string
	}{
		{label: "userId", value: userID},
		{label: "connectionId", value: connectionID},
		{label: "protocol", value: protocol},
		{label: "ext", value: ext},
		{label: "gatewayDir", value: subdir},
	}
	for _, component := range components {
		if !isSafePathComponent(component.value) {
			return recordingPlan{}, fmt.Errorf("invalid recording path component (%s)", component.label)
		}
	}

	recordingRoot = filepath.Clean(recordingRoot)
	hostPath := filepath.Join(recordingRoot, subdir, userID, fmt.Sprintf("%s-%s-%d.%s", connectionID, strings.ToLower(protocol), now.UTC().UnixMilli(), ext))
	hostPath = filepath.Clean(hostPath)
	relative, err := filepath.Rel(recordingRoot, hostPath)
	if err != nil {
		return recordingPlan{}, fmt.Errorf("compute recording path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return recordingPlan{}, fmt.Errorf("recording path escapes allowed directory")
	}

	guacdPath := path.Join("/recordings", filepath.ToSlash(relative))
	return recordingPlan{
		HostPath:   hostPath,
		HostDir:    filepath.Dir(hostPath),
		GuacdPath:  guacdPath,
		GuacdDir:   path.Dir(guacdPath),
		GuacdName:  path.Base(guacdPath),
		RecordedAt: now.UTC(),
	}, nil
}

func writeAsciicastHeader(filePath string, header map[string]any) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create recording file: %w", err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(header); err != nil {
		return fmt.Errorf("write recording header: %w", err)
	}
	if err := os.Chmod(filePath, 0o666); err != nil {
		return fmt.Errorf("set recording file permissions: %w", err)
	}
	return nil
}
