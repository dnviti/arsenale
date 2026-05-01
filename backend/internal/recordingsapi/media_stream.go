package recordingsapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/jackc/pgx/v5"
)

func (s Service) streamRecordingFile(w http.ResponseWriter, item recordingResponse) error {
	file, err := os.Open(item.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &requestError{status: http.StatusNotFound, message: "Recording file not found on disk"}
		}
		return fmt.Errorf("open recording stream: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat recording stream: %w", err)
	}
	contentType, ext := contentTypeForFormat(item.Format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="recording-%s.%s"`, item.ID, ext))
	if info.Size() > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}

	_, err = io.Copy(w, file)
	return err
}

func contentTypeForFormat(format string) (contentType string, ext string) {
	switch format {
	case "asciicast":
		return "application/x-asciicast", "cast"
	case "guac":
		return recordingContentTypeRaw, "guac"
	default:
		return recordingContentTypeRaw, format
	}
}

func (s Service) writeRecordingError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case err == nil:
		return
	case errors.Is(err, pgx.ErrNoRows):
		app.ErrorJSON(w, http.StatusNotFound, fallback)
	default:
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
	}
}
