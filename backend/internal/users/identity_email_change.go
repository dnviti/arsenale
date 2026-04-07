package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s Service) InitiateEmailChange(ctx context.Context, userID string, payload map[string]json.RawMessage) (emailChangeInitResult, error) {
	newEmail, err := parseNewEmailChangePayload(payload)
	if err != nil {
		return emailChangeInitResult{}, err
	}

	if s.DB == nil {
		return emailChangeInitResult{}, fmt.Errorf("postgres is not configured")
	}

	var existingUserID string
	err = s.DB.QueryRow(ctx, `SELECT id FROM "User" WHERE email = $1`, newEmail).Scan(&existingUserID)
	switch {
	case err == nil && existingUserID != userID:
		return emailChangeInitResult{}, &requestError{status: http.StatusConflict, message: "Email already in use"}
	case err != nil && err != pgx.ErrNoRows:
		return emailChangeInitResult{}, err
	}

	var currentEmail string
	var emailVerified bool
	if err := s.DB.QueryRow(ctx, `SELECT email, "emailVerified" FROM "User" WHERE id = $1`, userID).Scan(&currentEmail, &emailVerified); err != nil {
		return emailChangeInitResult{}, err
	}
	if strings.EqualFold(currentEmail, newEmail) {
		return emailChangeInitResult{}, &requestError{status: http.StatusBadRequest, message: "New email must be different from the current email."}
	}

	if emailVerificationConfigured() && emailVerified {
		otpOld, err := generateOTPCode()
		if err != nil {
			return emailChangeInitResult{}, err
		}
		otpNew, err := generateOTPCode()
		if err != nil {
			return emailChangeInitResult{}, err
		}

		if _, err := s.DB.Exec(
			ctx,
			`UPDATE "User"
			    SET "pendingEmail" = $2,
			        "emailChangeCodeOldHash" = $3,
			        "emailChangeCodeNewHash" = $4,
			        "emailChangeExpiry" = $5,
			        "updatedAt" = NOW()
			  WHERE id = $1`,
			userID,
			newEmail,
			hashOTP(otpOld),
			hashOTP(otpNew),
			time.Now().Add(emailChangeTTL),
		); err != nil {
			return emailChangeInitResult{}, err
		}

		if err := s.sendEmailChangeCode(ctx, currentEmail, otpOld, true); err != nil {
			return emailChangeInitResult{}, err
		}
		if err := s.sendEmailChangeCode(ctx, newEmail, otpNew, false); err != nil {
			return emailChangeInitResult{}, err
		}

		return emailChangeInitResult{Flow: "dual-otp"}, nil
	}

	if _, err := s.DB.Exec(ctx, `UPDATE "User" SET "pendingEmail" = $2, "updatedAt" = NOW() WHERE id = $1`, userID, newEmail); err != nil {
		return emailChangeInitResult{}, err
	}

	result, err := s.InitiateIdentity(ctx, userID, map[string]json.RawMessage{
		"purpose": json.RawMessage(`"email-change"`),
	})
	if err != nil {
		return emailChangeInitResult{}, err
	}

	return emailChangeInitResult{
		Flow:           "identity-verification",
		VerificationID: result.VerificationID,
		Method:         result.Method,
		Metadata:       result.Metadata,
	}, nil
}

func (s Service) ConfirmEmailChange(ctx context.Context, userID string, payload map[string]json.RawMessage, ipAddress string) (map[string]string, error) {
	confirmation, err := parseEmailChangeConfirmation(payload)
	if err != nil {
		return nil, err
	}
	if s.DB == nil {
		return nil, fmt.Errorf("postgres is not configured")
	}

	var (
		pendingEmail       *string
		emailChangeCodeOld *string
		emailChangeCodeNew *string
		emailChangeExpiry  *time.Time
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "pendingEmail", "emailChangeCodeOldHash", "emailChangeCodeNewHash", "emailChangeExpiry"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&pendingEmail, &emailChangeCodeOld, &emailChangeCodeNew, &emailChangeExpiry); err != nil {
		return nil, err
	}
	if pendingEmail == nil || strings.TrimSpace(*pendingEmail) == "" {
		return nil, &requestError{status: http.StatusBadRequest, message: "No pending email change."}
	}

	switch {
	case confirmation.UsesOTP:
		if emailChangeCodeOld == nil || emailChangeCodeNew == nil {
			return nil, &requestError{status: http.StatusBadRequest, message: "Invalid confirmation payload."}
		}
		if emailChangeExpiry != nil && emailChangeExpiry.Before(time.Now()) {
			if _, err := s.DB.Exec(
				ctx,
				`UPDATE "User"
				    SET "pendingEmail" = NULL,
				        "emailChangeCodeOldHash" = NULL,
				        "emailChangeCodeNewHash" = NULL,
				        "emailChangeExpiry" = NULL,
				        "updatedAt" = NOW()
				  WHERE id = $1`,
				userID,
			); err != nil {
				return nil, err
			}
			return nil, &requestError{status: http.StatusBadRequest, message: "Verification codes have expired. Please start again."}
		}
		if !timingSafeHexEqual(hashOTP(confirmation.CodeOld), *emailChangeCodeOld) ||
			!timingSafeHexEqual(hashOTP(confirmation.CodeNew), *emailChangeCodeNew) {
			return nil, &requestError{status: http.StatusBadRequest, message: "Invalid verification code(s)."}
		}
	case confirmation.VerificationID != "":
		if emailChangeCodeOld != nil || emailChangeCodeNew != nil || emailChangeExpiry != nil {
			return nil, &requestError{status: http.StatusBadRequest, message: "Invalid confirmation payload."}
		}
		if err := s.consumeVerificationSession(ctx, confirmation.VerificationID, userID, "email-change"); err != nil {
			return nil, err
		}
	default:
		return nil, &requestError{status: http.StatusBadRequest, message: "Invalid confirmation payload."}
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin confirm email change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var updatedEmail string
	if err := tx.QueryRow(
		ctx,
		`UPDATE "User"
		    SET email = "pendingEmail",
		        "emailVerified" = TRUE,
		        "pendingEmail" = NULL,
		        "emailChangeCodeOldHash" = NULL,
		        "emailChangeCodeNewHash" = NULL,
		        "emailChangeExpiry" = NULL,
		        "updatedAt" = NOW()
		  WHERE id = $1
		  RETURNING email`,
		userID,
	).Scan(&updatedEmail); err != nil {
		return nil, err
	}

	if err := insertAuditLog(ctx, tx, userID, "PROFILE_EMAIL_CHANGE", map[string]any{
		"newEmail": updatedEmail,
	}, ipAddress); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit confirm email change: %w", err)
	}

	return map[string]string{"email": updatedEmail}, nil
}
