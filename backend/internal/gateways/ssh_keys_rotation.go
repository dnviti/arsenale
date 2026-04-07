package gateways

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func computeSSHKeyRotationStatus(autoRotateEnabled bool, rotationIntervalDays int, expiresAt, lastAutoRotatedAt *time.Time, now time.Time, advanceDays int) sshKeyRotationStatus {
	status := sshKeyRotationStatus{
		AutoRotateEnabled:    autoRotateEnabled,
		RotationIntervalDays: rotationIntervalDays,
		ExpiresAt:            expiresAt,
		LastAutoRotatedAt:    lastAutoRotatedAt,
		KeyExists:            true,
	}
	if !autoRotateEnabled || expiresAt == nil {
		return status
	}

	nextRotationDate := expiresAt.UTC().Add(-time.Duration(advanceDays) * 24 * time.Hour)
	daysUntilRotation := int(nextRotationDate.Sub(now.UTC()).Hours() / 24)
	if nextRotationDate.After(now.UTC()) && nextRotationDate.Sub(now.UTC())%(24*time.Hour) != 0 {
		daysUntilRotation++
	}
	if daysUntilRotation < 0 {
		daysUntilRotation = 0
	}
	status.NextRotationDate = &nextRotationDate
	status.DaysUntilRotation = &daysUntilRotation
	return status
}

func validateRotationPolicyPayload(input rotationPolicyPayload) error {
	if input.RotationIntervalDays != nil && (*input.RotationIntervalDays < 1 || *input.RotationIntervalDays > 365) {
		return &requestError{status: http.StatusBadRequest, message: "rotationIntervalDays must be between 1 and 365"}
	}
	return nil
}

func sshKeyRotationAdvanceDays() int {
	value := strings.TrimSpace(os.Getenv("KEY_ROTATION_ADVANCE_DAYS"))
	if value == "" {
		return 7
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 7
	}
	return parsed
}

func errorsIsNotFound(err error) bool {
	var reqErr *requestError
	return errors.As(err, &reqErr) && reqErr.status == http.StatusNotFound
}
