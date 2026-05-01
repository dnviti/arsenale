package secretsmeta

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type restoreVersionResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CurrentVersion int       `json:"currentVersion"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (s Service) HandleRestoreVersion(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		app.ErrorJSON(w, http.StatusBadRequest, "Invalid version number")
		return
	}

	item, err := s.RestoreVersion(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), version)
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	_ = s.insertAuditLog(r.Context(), claims.UserID, "SECRET_VERSION_RESTORE", r.PathValue("id"), map[string]any{
		"restoredVersion": version,
	}, requestIP(r))
	app.WriteJSON(w, http.StatusOK, item)
}

func (s Service) RestoreVersion(ctx context.Context, userID, tenantID, secretID string, version int) (restoreVersionResponse, error) {
	if _, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID); err != nil {
		return restoreVersionResponse{}, err
	}
	if s.DB == nil {
		return restoreVersionResponse{}, fmt.Errorf("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return restoreVersionResponse{}, fmt.Errorf("begin restore secret version: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		name           string
		currentVersion int
	)
	if err := tx.QueryRow(
		ctx,
		`SELECT name, "currentVersion"
		   FROM "VaultSecret"
		  WHERE id = $1
		  FOR UPDATE`,
		secretID,
	).Scan(&name, &currentVersion); err != nil {
		return restoreVersionResponse{}, fmt.Errorf("load current secret version: %w", err)
	}

	var (
		encryptedData string
		dataIV        string
		dataTag       string
	)
	if err := tx.QueryRow(
		ctx,
		`SELECT "encryptedData", "dataIV", "dataTag"
		   FROM "VaultSecretVersion"
		  WHERE "secretId" = $1
		    AND version = $2`,
		secretID,
		version,
	).Scan(&encryptedData, &dataIV, &dataTag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return restoreVersionResponse{}, &credentialresolver.RequestError{Status: http.StatusNotFound, Message: "Version not found"}
		}
		return restoreVersionResponse{}, fmt.Errorf("load secret version: %w", err)
	}

	newVersion := currentVersion + 1
	var result restoreVersionResponse
	if err := tx.QueryRow(
		ctx,
		`UPDATE "VaultSecret"
		    SET "encryptedData" = $2,
		        "dataIV" = $3,
		        "dataTag" = $4,
		        "currentVersion" = $5,
		        "updatedAt" = NOW()
		  WHERE id = $1
		RETURNING id, name, "currentVersion", "updatedAt"`,
		secretID,
		encryptedData,
		dataIV,
		dataTag,
		newVersion,
	).Scan(&result.ID, &result.Name, &result.CurrentVersion, &result.UpdatedAt); err != nil {
		return restoreVersionResponse{}, fmt.Errorf("restore secret version: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO "VaultSecretVersion" (
			id,
			"secretId",
			version,
			"encryptedData",
			"dataIV",
			"dataTag",
			"changedBy",
			"changeNote"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.NewString(),
		secretID,
		newVersion,
		encryptedData,
		dataIV,
		dataTag,
		userID,
		fmt.Sprintf("Restored from version %d", version),
	); err != nil {
		return restoreVersionResponse{}, fmt.Errorf("record restored secret version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return restoreVersionResponse{}, fmt.Errorf("commit restore secret version: %w", err)
	}

	return result, nil
}
