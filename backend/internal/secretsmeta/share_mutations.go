package secretsmeta

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type secretShareMutationResponse struct {
	ID         string `json:"id"`
	Permission string `json:"permission"`
	SharedWith string `json:"sharedWith"`
}

type secretShareTargetUser struct {
	ID    string
	Email string
}

func (s Service) HandleShare(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	body, err := readBodyBytes(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := parseShareSecretInput(body)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.ShareSecret(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), payload, requestIP(r))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUnshare(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.UnshareSecret(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), r.PathValue("userId"), requestIP(r))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateSharePermission(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	body, err := readBodyBytes(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := parseUpdateSecretSharePermissionInput(body)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateSecretSharePermission(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), r.PathValue("userId"), payload.Permission, requestIP(r))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) ShareSecret(ctx context.Context, userID, tenantID, secretID string, input shareSecretInput, ipAddress string) (secretShareMutationResponse, error) {
	access, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	if access.TeamID != nil && access.TeamRole != "TEAM_ADMIN" {
		return secretShareMutationResponse{}, &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only team admins can share team secrets"}
	}

	targetUser, err := s.resolveSecretShareTargetUser(ctx, input)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	if targetUser.ID == userID {
		return secretShareMutationResponse{}, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "Cannot share with yourself"}
	}
	if err := s.assertShareableTenantBoundary(ctx, userID, targetUser.ID); err != nil {
		return secretShareMutationResponse{}, err
	}

	targetKey, err := s.resolver().LoadUserMasterKey(ctx, targetUser.ID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	if len(targetKey) == 0 {
		return secretShareMutationResponse{}, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "Unable to share with this user at this time."}
	}
	defer zeroBytes(targetKey)

	detail, err := s.resolver().ResolveSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}

	ciphertext, iv, tag, err := s.resolver().EncryptWithKey(targetKey, string(detail.Data))
	if err != nil {
		return secretShareMutationResponse{}, err
	}

	var result secretShareMutationResponse
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "SharedSecret" (
	id,
	"secretId",
	"sharedWithUserId",
	"sharedByUserId",
	permission,
	"encryptedData",
	"dataIV",
	"dataTag",
	"createdAt"
)
VALUES ($1, $2, $3, $4, $5::"Permission", $6, $7, $8, NOW())
ON CONFLICT ("secretId", "sharedWithUserId")
DO UPDATE SET
	permission = EXCLUDED.permission,
	"sharedByUserId" = EXCLUDED."sharedByUserId",
	"encryptedData" = EXCLUDED."encryptedData",
	"dataIV" = EXCLUDED."dataIV",
	"dataTag" = EXCLUDED."dataTag"
RETURNING id, permission::text
`, uuid.NewString(), secretID, targetUser.ID, userID, input.Permission, ciphertext, iv, tag).Scan(&result.ID, &result.Permission); err != nil {
		return secretShareMutationResponse{}, fmt.Errorf("upsert shared secret: %w", err)
	}
	result.SharedWith = targetUser.Email

	actorName, err := s.lookupActorName(ctx, userID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	permissionLabel := "Read Only"
	if input.Permission == "FULL_ACCESS" {
		permissionLabel = "Full Access"
	}
	if err := s.insertNotification(ctx, targetUser.ID, "SECRET_SHARED", fmt.Sprintf(`%s shared secret "%s" with you (%s)`, actorName, detail.Name, permissionLabel), secretID); err != nil {
		return secretShareMutationResponse{}, err
	}
	_ = s.insertAuditLog(ctx, userID, "SECRET_SHARE", secretID, map[string]any{
		"sharedWith": targetUser.ID,
		"permission": input.Permission,
	}, ipAddress)

	return result, nil
}

func (s Service) UnshareSecret(ctx context.Context, userID, tenantID, secretID, targetUserID, ipAddress string) (map[string]bool, error) {
	access, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return nil, err
	}
	if access.TeamID != nil && access.TeamRole != "TEAM_ADMIN" {
		return nil, &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only team admins can revoke team secret shares"}
	}

	detail, err := s.resolver().LoadSecretSummary(ctx, secretID)
	if err != nil {
		return nil, err
	}

	if _, err := s.DB.Exec(ctx, `DELETE FROM "SharedSecret" WHERE "secretId" = $1 AND "sharedWithUserId" = $2`, secretID, strings.TrimSpace(targetUserID)); err != nil {
		return nil, fmt.Errorf("delete shared secret: %w", err)
	}

	actorName, err := s.lookupActorName(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.insertNotification(ctx, strings.TrimSpace(targetUserID), "SECRET_SHARE_REVOKED", fmt.Sprintf(`%s revoked your access to secret "%s"`, actorName, detail.Name), secretID); err != nil {
		return nil, err
	}
	_ = s.insertAuditLog(ctx, userID, "SECRET_UNSHARE", secretID, map[string]any{
		"targetUserId": strings.TrimSpace(targetUserID),
	}, ipAddress)

	return map[string]bool{"deleted": true}, nil
}

func (s Service) UpdateSecretSharePermission(ctx context.Context, userID, tenantID, secretID, targetUserID, permission, ipAddress string) (secretShareMutationResponse, error) {
	access, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	if access.TeamID != nil && access.TeamRole != "TEAM_ADMIN" {
		return secretShareMutationResponse{}, &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only team admins can update team secret shares"}
	}

	detail, err := s.resolver().LoadSecretSummary(ctx, secretID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}

	var result secretShareMutationResponse
	if err := s.DB.QueryRow(ctx, `
UPDATE "SharedSecret"
SET permission = $3::"Permission"
WHERE "secretId" = $1
  AND "sharedWithUserId" = $2
RETURNING id, permission::text
`, secretID, strings.TrimSpace(targetUserID), permission).Scan(&result.ID, &result.Permission); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return secretShareMutationResponse{}, &credentialresolver.RequestError{Status: http.StatusNotFound, Message: "Share not found"}
		}
		return secretShareMutationResponse{}, fmt.Errorf("update shared secret permission: %w", err)
	}

	if err := s.DB.QueryRow(ctx, `SELECT email FROM "User" WHERE id = $1`, strings.TrimSpace(targetUserID)).Scan(&result.SharedWith); err != nil {
		return secretShareMutationResponse{}, fmt.Errorf("load shared user email: %w", err)
	}

	actorName, err := s.lookupActorName(ctx, userID)
	if err != nil {
		return secretShareMutationResponse{}, err
	}
	permissionLabel := "Read Only"
	if permission == "FULL_ACCESS" {
		permissionLabel = "Full Access"
	}
	if err := s.insertNotification(ctx, strings.TrimSpace(targetUserID), "SHARE_PERMISSION_UPDATED", fmt.Sprintf(`%s changed your permission on secret "%s" to %s`, actorName, detail.Name, permissionLabel), secretID); err != nil {
		return secretShareMutationResponse{}, err
	}
	_ = s.insertAuditLog(ctx, userID, "SECRET_SHARE_UPDATE", secretID, map[string]any{
		"targetUserId": strings.TrimSpace(targetUserID),
		"permission":   permission,
	}, ipAddress)

	return result, nil
}

func (s Service) resolveSecretShareTargetUser(ctx context.Context, input shareSecretInput) (secretShareTargetUser, error) {
	email := normalizeOptionalString(input.Email)
	userID := normalizeOptionalString(input.UserID)

	var target secretShareTargetUser
	if userID != nil {
		if err := s.DB.QueryRow(ctx, `SELECT id, email FROM "User" WHERE id = $1`, *userID).Scan(&target.ID, &target.Email); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return secretShareTargetUser{}, &credentialresolver.RequestError{Status: http.StatusNotFound, Message: "User not found"}
			}
			return secretShareTargetUser{}, fmt.Errorf("load share target user: %w", err)
		}
		return target, nil
	}

	if err := s.DB.QueryRow(ctx, `SELECT id, email FROM "User" WHERE LOWER(email) = LOWER($1)`, *email).Scan(&target.ID, &target.Email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return secretShareTargetUser{}, &credentialresolver.RequestError{Status: http.StatusNotFound, Message: "User not found"}
		}
		return secretShareTargetUser{}, fmt.Errorf("load share target user: %w", err)
	}
	return target, nil
}

func (s Service) assertShareableTenantBoundary(ctx context.Context, actingUserID, targetUserID string) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_EXTERNAL_SHARING")), "true") {
		return nil
	}

	actingTenantIDs, err := s.loadAcceptedTenantIDs(ctx, actingUserID)
	if err != nil {
		return err
	}
	targetTenantIDs, err := s.loadAcceptedTenantIDs(ctx, targetUserID)
	if err != nil {
		return err
	}
	if len(actingTenantIDs) == 0 && len(targetTenantIDs) == 0 {
		return nil
	}

	for _, actingTenantID := range actingTenantIDs {
		for _, targetTenantID := range targetTenantIDs {
			if actingTenantID == targetTenantID {
				return nil
			}
		}
	}
	return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Cannot share connections with users outside your tenant"}
}

func (s Service) loadAcceptedTenantIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.DB.Query(ctx, `
SELECT "tenantId"
FROM "TenantMember"
WHERE "userId" = $1
  AND status = 'ACCEPTED'
  AND ("expiresAt" IS NULL OR "expiresAt" > NOW())
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list tenant memberships: %w", err)
	}
	defer rows.Close()

	tenantIDs := make([]string, 0)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, fmt.Errorf("scan tenant membership: %w", err)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant memberships: %w", err)
	}
	return tenantIDs, nil
}

func (s Service) lookupActorName(ctx context.Context, userID string) (string, error) {
	var actorName string
	if err := s.DB.QueryRow(ctx, `SELECT COALESCE(NULLIF(username, ''), email, 'Someone') FROM "User" WHERE id = $1`, userID).Scan(&actorName); err != nil {
		return "", fmt.Errorf("load actor identity: %w", err)
	}
	return actorName, nil
}

func (s Service) insertNotification(ctx context.Context, userID, notificationType, message, relatedID string) error {
	_, err := s.DB.Exec(ctx, `
INSERT INTO "Notification" (id, "userId", type, message, read, "relatedId", "createdAt")
VALUES ($1, $2, $3::"NotificationType", $4, false, NULLIF($5, ''), NOW())
`, uuid.NewString(), userID, notificationType, message, relatedID)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
