package sessionrecording

import (
	"strconv"
	"strings"
	"time"
)

func MetadataFromReference(ref Reference) map[string]any {
	return map[string]any{
		"id":        ref.ID,
		"filePath":  ref.FilePath,
		"startedAt": ref.StartedAt.UTC().Format(time.RFC3339Nano),
		"width":     ref.Width,
		"height":    ref.Height,
		"format":    ref.Format,
		"protocol":  ref.Protocol,
	}
}

func MetadataStringsFromReference(ref Reference) map[string]string {
	return map[string]string{
		"recordingId":        strings.TrimSpace(ref.ID),
		"recordingPath":      strings.TrimSpace(ref.FilePath),
		"recordingStartedAt": ref.StartedAt.UTC().Format(time.RFC3339Nano),
		"recordingWidth":     strconv.Itoa(ref.Width),
		"recordingHeight":    strconv.Itoa(ref.Height),
		"recordingFormat":    strings.TrimSpace(ref.Format),
		"recordingProtocol":  strings.TrimSpace(ref.Protocol),
	}
}

func ReferenceFromMetadata(metadata map[string]any) (Reference, bool) {
	raw, ok := metadata["recording"]
	if !ok {
		return Reference{}, false
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return Reference{}, false
	}

	startedAt, ok := parseTimeValue(payload["startedAt"])
	if !ok {
		return Reference{}, false
	}

	ref := Reference{
		ID:        stringify(payload["id"]),
		FilePath:  stringify(payload["filePath"]),
		StartedAt: startedAt,
		Width:     intValue(payload["width"], defaultCastCols),
		Height:    intValue(payload["height"], defaultCastRows),
		Format:    defaultString(stringify(payload["format"]), "asciicast"),
		Protocol:  strings.ToUpper(defaultString(stringify(payload["protocol"]), "")),
	}
	if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.FilePath) == "" {
		return Reference{}, false
	}
	return ref, true
}

func ReferenceFromMetadataStrings(metadata map[string]string) (Reference, bool) {
	id := strings.TrimSpace(metadata["recordingId"])
	filePath := strings.TrimSpace(metadata["recordingPath"])
	startedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(metadata["recordingStartedAt"]))
	if id == "" || filePath == "" || err != nil {
		return Reference{}, false
	}

	return Reference{
		ID:        id,
		FilePath:  filePath,
		StartedAt: startedAt.UTC(),
		Width:     parseIntDefault(metadata["recordingWidth"], defaultCastCols),
		Height:    parseIntDefault(metadata["recordingHeight"], defaultCastRows),
		Format:    defaultString(strings.TrimSpace(metadata["recordingFormat"]), "asciicast"),
		Protocol:  strings.ToUpper(strings.TrimSpace(metadata["recordingProtocol"])),
	}, true
}
