package files

import (
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDriveBasePath  = "/guacd-drive"
	defaultMaxUploadBytes = 10 * 1024 * 1024
	defaultUserQuotaBytes = 100 * 1024 * 1024
	multipartOverhead     = 1024 * 1024
	maxFileNameLength     = 255
)

var unsafeUserIDPattern = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var unsafeUploadNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

type Service struct {
	DB                *pgxpool.Pool
	DriveBasePath     string
	FileUploadMaxSize int64
	UserDriveQuota    int64
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type FileInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

type tenantFilePolicy struct {
	DLPDisableDownload bool
	DLPDisableUpload   bool
	FileUploadMaxBytes *int64
	UserDriveQuota     *int64
}
