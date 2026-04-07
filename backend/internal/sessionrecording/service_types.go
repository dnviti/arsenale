package sessionrecording

import (
	"regexp"
	"time"
)

const (
	defaultGatewayDir = "default"
	defaultCastCols   = 80
	defaultCastRows   = 24
)

var safePathComponentPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type Reference struct {
	ID         string
	FilePath   string
	StartedAt  time.Time
	Width      int
	Height     int
	Format     string
	Protocol   string
	Connection string
}

type recordingPlan struct {
	HostPath   string
	HostDir    string
	GuacdPath  string
	GuacdDir   string
	GuacdName  string
	RecordedAt time.Time
}

type recordingRecord struct {
	ID             string
	UserID         string
	ConnectionID   string
	Protocol       string
	FilePath       string
	Status         string
	CreatedAt      time.Time
	ConnectionName *string
}
