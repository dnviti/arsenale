package connections

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) ShareConnection(ctx context.Context, claims authn.Claims, connectionID string, target shareTarget, permission string, ip *string) (shareMutationResponse, error) {
	access, err := s.resolveShareableConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return shareMutationResponse{}, err
	}

	targetUser, err := s.resolveShareTargetUser(ctx, target)
	if err != nil {
		return shareMutationResponse{}, err
	}
	if targetUser.ID == claims.UserID {
		return shareMutationResponse{}, &requestError{status: http.StatusBadRequest, message: "Cannot share with yourself"}
	}
	if err := s.assertShareableTenantBoundary(ctx, claims.UserID, targetUser.ID); err != nil {
		return shareMutationResponse{}, err
	}

	targetKey, err := s.getVaultKey(ctx, targetUser.ID)
	if err != nil {
		return shareMutationResponse{}, err
	}
	if len(targetKey) == 0 {
		return shareMutationResponse{}, &requestError{status: http.StatusBadRequest, message: "Unable to share with this user at this time."}
	}
	defer zeroBytes(targetKey)

	var encryptedUsername any
	var usernameIV any
	var usernameTag any
	var encryptedPassword any
	var passwordIV any
	var passwordTag any
	var encryptedDomain any
	var domainIV any
	var domainTag any

	if access.Connection.CredentialSecretID == nil {
		sourceKey, err := s.loadSharingSourceKey(ctx, claims.UserID, access.Connection)
		if err != nil {
			return shareMutationResponse{}, err
		}
		defer zeroBytes(sourceKey)

		sourceCreds, err := s.loadSharableCredentials(ctx, access.Connection.ID)
		if err != nil {
			return shareMutationResponse{}, err
		}
		encUsername, encPassword, encDomainField, err := reencryptSharedCredentials(sourceKey, targetKey, sourceCreds)
		if err != nil {
			return shareMutationResponse{}, err
		}
		encryptedUsername, usernameIV, usernameTag = encUsername.Ciphertext, encUsername.IV, encUsername.Tag
		encryptedPassword, passwordIV, passwordTag = encPassword.Ciphertext, encPassword.IV, encPassword.Tag
		encryptedDomain, domainIV, domainTag = nullCiphertext(encDomainField), nullIV(encDomainField), nullTag(encDomainField)
	}

	var result shareMutationResponse
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "SharedConnection" (
	id,
	"connectionId",
	"sharedWithUserId",
	"sharedByUserId",
	permission,
	"encryptedUsername",
	"usernameIV",
	"usernameTag",
	"encryptedPassword",
	"passwordIV",
	"passwordTag",
	"encryptedDomain",
	"domainIV",
	"domainTag",
	"createdAt"
)
VALUES ($1, $2, $3, $4, $5::"Permission", $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
ON CONFLICT ("connectionId", "sharedWithUserId")
DO UPDATE SET
	permission = EXCLUDED.permission,
	"sharedByUserId" = EXCLUDED."sharedByUserId",
	"encryptedUsername" = EXCLUDED."encryptedUsername",
	"usernameIV" = EXCLUDED."usernameIV",
	"usernameTag" = EXCLUDED."usernameTag",
	"encryptedPassword" = EXCLUDED."encryptedPassword",
	"passwordIV" = EXCLUDED."passwordIV",
	"passwordTag" = EXCLUDED."passwordTag",
	"encryptedDomain" = EXCLUDED."encryptedDomain",
	"domainIV" = EXCLUDED."domainIV",
	"domainTag" = EXCLUDED."domainTag"
RETURNING id, permission::text
`, uuid.NewString(), access.Connection.ID, targetUser.ID, claims.UserID, permission, encryptedUsername, usernameIV, usernameTag, encryptedPassword, passwordIV, passwordTag, encryptedDomain, domainIV, domainTag).Scan(&result.ID, &result.Permission); err != nil {
		return shareMutationResponse{}, fmt.Errorf("upsert shared connection: %w", err)
	}
	result.SharedWith = targetUser.Email

	actorName, err := s.lookupActorName(ctx, claims.UserID)
	if err != nil {
		return shareMutationResponse{}, err
	}
	permissionLabel := "Read Only"
	if permission == "FULL_ACCESS" {
		permissionLabel = "Full Access"
	}
	if err := s.insertNotification(ctx, targetUser.ID, "CONNECTION_SHARED", fmt.Sprintf(`%s shared "%s" with you (%s)`, actorName, access.Connection.Name, permissionLabel), access.Connection.ID); err != nil {
		return shareMutationResponse{}, err
	}
	if err := s.insertAuditLog(ctx, claims.UserID, "SHARE_CONNECTION", access.Connection.ID, map[string]any{
		"sharedWith": targetUser.ID,
		"permission": permission,
	}, ip); err != nil {
		return shareMutationResponse{}, err
	}
	return result, nil
}

func (s Service) BatchShareConnections(ctx context.Context, claims authn.Claims, payload batchSharePayload, ip *string) (batchShareResponse, error) {
	if _, err := s.resolveShareTargetUser(ctx, payload.Target); err != nil {
		return batchShareResponse{}, err
	}

	result := batchShareResponse{Errors: make([]batchShareResultReason, 0)}
	for _, connectionID := range payload.ConnectionIDs {
		if _, err := s.ShareConnection(ctx, claims, connectionID, payload.Target, payload.Permission, ip); err != nil {
			result.Failed++
			reason := err.Error()
			var reqErr *requestError
			if errors.As(err, &reqErr) {
				reason = reqErr.message
			}
			result.Errors = append(result.Errors, batchShareResultReason{
				ConnectionID: connectionID,
				Reason:       reason,
			})
			continue
		}
		result.Shared++
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "BATCH_SHARE", "", map[string]any{
		"connectionCount": len(payload.ConnectionIDs),
		"shared":          result.Shared,
		"failed":          result.Failed,
		"permission":      payload.Permission,
		"folderName":      normalizeOptionalStringPtrValue(payload.FolderName),
	}, ip); err != nil {
		return batchShareResponse{}, err
	}
	return result, nil
}

func (s Service) UnshareConnection(ctx context.Context, claims authn.Claims, connectionID, targetUserID string, ip *string) error {
	access, err := s.resolveShareableConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(strings.TrimSpace(targetUserID)); err != nil {
		return &requestError{status: http.StatusBadRequest, message: "invalid userId"}
	}

	if _, err := s.DB.Exec(ctx, `DELETE FROM "SharedConnection" WHERE "connectionId" = $1 AND "sharedWithUserId" = $2`, access.Connection.ID, targetUserID); err != nil {
		return fmt.Errorf("delete shared connection: %w", err)
	}

	actorName, err := s.lookupActorName(ctx, claims.UserID)
	if err != nil {
		return err
	}
	if err := s.insertNotification(ctx, targetUserID, "SHARE_REVOKED", fmt.Sprintf(`%s revoked your access to "%s"`, actorName, access.Connection.Name), access.Connection.ID); err != nil {
		return err
	}
	return s.insertAuditLog(ctx, claims.UserID, "UNSHARE_CONNECTION", access.Connection.ID, map[string]any{
		"targetUserId": targetUserID,
	}, ip)
}

func (s Service) UpdateSharePermission(ctx context.Context, claims authn.Claims, connectionID, targetUserID, permission string, ip *string) (shareMutationResponse, error) {
	access, err := s.resolveShareableConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return shareMutationResponse{}, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(targetUserID)); err != nil {
		return shareMutationResponse{}, &requestError{status: http.StatusBadRequest, message: "invalid userId"}
	}

	var result shareMutationResponse
	if err := s.DB.QueryRow(ctx, `
UPDATE "SharedConnection"
SET permission = $3::"Permission"
WHERE "connectionId" = $1 AND "sharedWithUserId" = $2
RETURNING id, permission::text
`, access.Connection.ID, targetUserID, permission).Scan(&result.ID, &result.Permission); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shareMutationResponse{}, &requestError{status: http.StatusNotFound, message: "Share not found"}
		}
		return shareMutationResponse{}, fmt.Errorf("update shared connection permission: %w", err)
	}

	if err := s.DB.QueryRow(ctx, `SELECT email FROM "User" WHERE id = $1`, targetUserID).Scan(&result.SharedWith); err != nil {
		return shareMutationResponse{}, fmt.Errorf("load shared user email: %w", err)
	}

	actorName, err := s.lookupActorName(ctx, claims.UserID)
	if err != nil {
		return shareMutationResponse{}, err
	}
	permissionLabel := "Read Only"
	if permission == "FULL_ACCESS" {
		permissionLabel = "Full Access"
	}
	if err := s.insertNotification(ctx, targetUserID, "SHARE_PERMISSION_UPDATED", fmt.Sprintf(`%s changed your permission on "%s" to %s`, actorName, access.Connection.Name, permissionLabel), access.Connection.ID); err != nil {
		return shareMutationResponse{}, err
	}
	if err := s.insertAuditLog(ctx, claims.UserID, "UPDATE_SHARE_PERMISSION", access.Connection.ID, map[string]any{
		"targetUserId": targetUserID,
		"permission":   permission,
	}, ip); err != nil {
		return shareMutationResponse{}, err
	}
	return result, nil
}

func (s Service) ListShares(ctx context.Context, claims authn.Claims, connectionID string) ([]shareListEntry, error) {
	access, err := s.resolveShareableConnection(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(ctx, `
SELECT sc.id, u.id, u.email, sc.permission::text, sc."createdAt"
FROM "SharedConnection" sc
JOIN "User" u ON u.id = sc."sharedWithUserId"
WHERE sc."connectionId" = $1
ORDER BY sc."createdAt" ASC
`, access.Connection.ID)
	if err != nil {
		return nil, fmt.Errorf("list shared connections: %w", err)
	}
	defer rows.Close()

	result := make([]shareListEntry, 0)
	for rows.Next() {
		var item shareListEntry
		if err := rows.Scan(&item.ID, &item.UserID, &item.Email, &item.Permission, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan shared connection: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shared connections: %w", err)
	}
	return result, nil
}
