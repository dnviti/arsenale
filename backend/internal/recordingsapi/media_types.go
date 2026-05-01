package recordingsapi

import "time"

const (
	maxAnalyzeBytes         = 10 * 1024 * 1024
	guacencPollInterval     = 2 * time.Second
	guacencSubmitTimeout    = 10 * time.Second
	guacencStatusTimeout    = 5 * time.Second
	defaultGuacencTimeout   = 120 * time.Second
	defaultVideoWidth       = 1024
	defaultVideoHeight      = 768
	recordingContentTypeRaw = "application/octet-stream"
)

type recordingAnalysisResponse struct {
	FileSize       int            `json:"fileSize"`
	Truncated      bool           `json:"truncated"`
	Instructions   map[string]int `json:"instructions"`
	SyncCount      int            `json:"syncCount"`
	DisplayWidth   int            `json:"displayWidth"`
	DisplayHeight  int            `json:"displayHeight"`
	HasLayer0Image bool           `json:"hasLayer0Image"`
}
