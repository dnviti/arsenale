package sessionrecording

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func AppendAsciicastOutput(filePath string, startedAt time.Time, output string) error {
	return AppendAsciicastOutputAt(filePath, startedAt, time.Now().UTC(), output)
}

func AppendAsciicastOutputAt(filePath string, startedAt, eventAt time.Time, output string) error {
	if strings.TrimSpace(filePath) == "" || output == "" {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open recording file for append: %w", err)
	}
	defer file.Close()

	elapsed := eventAt.UTC().Sub(startedAt.UTC()).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	event, err := json.Marshal([]any{elapsed, "o", output})
	if err != nil {
		return fmt.Errorf("marshal asciicast event: %w", err)
	}
	if _, err := file.Write(append(event, '\n')); err != nil {
		return fmt.Errorf("append asciicast event: %w", err)
	}
	return nil
}
