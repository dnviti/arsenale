package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s Service) getFilePath(userID, fileName string) (string, error) {
	sanitized := strings.TrimSpace(fileName)
	if sanitized == "" || len(sanitized) > maxFileNameLength {
		return "", &requestError{status: 400, message: "Invalid file name"}
	}
	if strings.ContainsAny(sanitized, `/\`) || filepath.Base(sanitized) != sanitized {
		return "", &requestError{status: 400, message: "Invalid file name"}
	}
	filePath := filepath.Join(s.userDrivePath(userID), sanitized)
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &requestError{status: 404, message: "File not found"}
		}
		return "", fmt.Errorf("stat drive file: %w", err)
	}
	return filePath, nil
}

func (s Service) ensureUserDrive(userID string) (string, error) {
	dirPath := s.userDrivePath(userID)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", fmt.Errorf("create user drive: %w", err)
	}
	return dirPath, nil
}

func (s Service) userDrivePath(userID string) string {
	basePath := s.DriveBasePath
	if strings.TrimSpace(basePath) == "" {
		basePath = defaultDriveBasePath
	}
	return filepath.Join(basePath, sanitizeUserID(userID))
}

func sanitizeUserID(userID string) string {
	safe := unsafeUserIDPattern.ReplaceAllString(userID, "")
	if safe == "" {
		return "unknown"
	}
	return safe
}

func sanitizeUploadName(fileName string) string {
	name := strings.TrimSpace(filepath.Base(fileName))
	name = unsafeUploadNamePattern.ReplaceAllString(name, "_")
	if name == "" || name == "." || name == ".." {
		name = "upload.bin"
	}
	if len(name) > maxFileNameLength {
		name = name[:maxFileNameLength]
	}
	return name
}
