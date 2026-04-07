package files

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	files, err := s.ListFiles(claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, files)
}

func (s Service) HandleDownload(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	policy, err := s.loadTenantPolicy(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if policy.DLPDisableDownload {
		app.ErrorJSON(w, http.StatusForbidden, "File download is disabled by organization policy")
		return
	}

	name := strings.TrimSpace(r.PathValue("name"))
	filePath, err := s.getFilePath(claims.UserID, name)
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			app.ErrorJSON(w, http.StatusNotFound, "File not found")
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	http.ServeContent(w, r, name, stat.ModTime(), file)
}

func (s Service) HandleUpload(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	policy, err := s.loadTenantPolicy(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if policy.DLPDisableUpload {
		app.ErrorJSON(w, http.StatusForbidden, "File upload is disabled by organization policy")
		return
	}

	maxUploadBytes := s.maxUploadBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+multipartOverhead)
	if err := r.ParseMultipartForm(maxUploadBytes + multipartOverhead); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			app.ErrorJSON(w, http.StatusRequestEntityTooLarge, "File exceeds maximum upload size")
			return
		}
		app.ErrorJSON(w, http.StatusBadRequest, "Invalid multipart form data")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	src, header, err := r.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			app.ErrorJSON(w, http.StatusBadRequest, "No file uploaded")
			return
		}
		app.ErrorJSON(w, http.StatusBadRequest, "Invalid file upload")
		return
	}
	defer src.Close()

	safeName := sanitizeUploadName(header.Filename)
	maxTenantBytes := policy.FileUploadMaxBytes
	if maxTenantBytes != nil && header.Size > *maxTenantBytes {
		app.ErrorJSON(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("File exceeds organization limit of %dMB", bytesToMB(*maxTenantBytes)))
		return
	}

	currentUsage, err := s.currentUsage(claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if quota := s.effectiveQuota(policy); quota > 0 && currentUsage+max(header.Size, 0) > quota {
		app.ErrorJSON(w, http.StatusRequestEntityTooLarge, quotaExceededMessage(currentUsage, quota))
		return
	}

	drivePath, err := s.ensureUserDrive(claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	targetPath := filepath.Join(drivePath, safeName)
	dst, err := os.Create(targetPath)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(targetPath)
		if copyErr != nil {
			app.ErrorJSON(w, http.StatusServiceUnavailable, copyErr.Error())
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, closeErr.Error())
		return
	}

	if maxTenantBytes != nil && written > *maxTenantBytes {
		_ = os.Remove(targetPath)
		app.ErrorJSON(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("File exceeds organization limit of %dMB", bytesToMB(*maxTenantBytes)))
		return
	}

	if quota := s.effectiveQuota(policy); quota > 0 && currentUsage+written > quota {
		_ = os.Remove(targetPath)
		app.ErrorJSON(w, http.StatusRequestEntityTooLarge, quotaExceededMessage(currentUsage, quota))
		return
	}

	files, err := s.ListFiles(claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusCreated, files)
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.DeleteFile(claims.UserID, strings.TrimSpace(r.PathValue("name"))); err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
