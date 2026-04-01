package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func (s Service) InitiatePasswordChange(ctx context.Context, userID string) (passwordChangeInitResult, error) {
	result, err := s.initiateVerification(ctx, userID, "password-change")
	if err != nil {
		return passwordChangeInitResult{}, err
	}
	if result.Method == "password" {
		return passwordChangeInitResult{SkipVerification: true}, nil
	}
	return passwordChangeInitResult{
		SkipVerification: false,
		VerificationID:   result.VerificationID,
		Method:           result.Method,
		Metadata:         result.Metadata,
	}, nil
}

func (s Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string, verificationID *string, ipAddress string) (passwordChangeResult, error) {
	if s.DB == nil {
		return passwordChangeResult{}, fmt.Errorf("postgres is not configured")
	}
	if err := validatePassword(newPassword); err != nil {
		return passwordChangeResult{}, err
	}

	var (
		passwordHash      *string
		vaultSalt         *string
		encryptedVaultKey *string
		vaultKeyIV        *string
		vaultKeyTag       *string
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&passwordHash, &vaultSalt, &encryptedVaultKey, &vaultKeyIV, &vaultKeyTag); err != nil {
		return passwordChangeResult{}, err
	}

	if passwordHash == nil || *passwordHash == "" {
		return passwordChangeResult{}, &requestError{status: http.StatusBadRequest, message: "Cannot change password for OAuth-only accounts."}
	}
	if vaultSalt == nil || encryptedVaultKey == nil || vaultKeyIV == nil || vaultKeyTag == nil ||
		*vaultSalt == "" || *encryptedVaultKey == "" || *vaultKeyIV == "" || *vaultKeyTag == "" {
		return passwordChangeResult{}, &requestError{status: http.StatusBadRequest, message: "Vault is not set up."}
	}

	var masterKey []byte
	if verificationID != nil && *verificationID != "" {
		if err := s.consumeVerificationSession(ctx, *verificationID, userID, "password-change"); err != nil {
			return passwordChangeResult{}, err
		}
		sessionKey, err := s.getVaultMasterKey(ctx, userID)
		if err != nil {
			return passwordChangeResult{}, err
		}
		if len(sessionKey) == 0 {
			return passwordChangeResult{}, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
		}
		masterKey = sessionKey
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(oldPassword)); err != nil {
			return passwordChangeResult{}, &requestError{status: http.StatusUnauthorized, message: "Current password is incorrect"}
		}
		oldDerivedKey := deriveKeyFromPassword(oldPassword, *vaultSalt)
		defer zeroBytes(oldDerivedKey)

		decrypted, err := decryptMasterKey(encryptedField{
			Ciphertext: *encryptedVaultKey,
			IV:         *vaultKeyIV,
			Tag:        *vaultKeyTag,
		}, oldDerivedKey)
		if err != nil {
			return passwordChangeResult{}, fmt.Errorf("decrypt master key: %w", err)
		}
		masterKey = decrypted
	}
	defer zeroBytes(masterKey)

	if err := assertPasswordNotBreached(ctx, newPassword); err != nil {
		return passwordChangeResult{}, err
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptRounds)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("hash password: %w", err)
	}
	newVaultSalt := generateSalt()
	newDerivedKey := deriveKeyFromPassword(newPassword, newVaultSalt)
	defer zeroBytes(newDerivedKey)
	newEncryptedVault, err := encryptMasterKey(masterKey, newDerivedKey)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("encrypt master key: %w", err)
	}

	newRecoveryKey, err := generateRecoveryKey()
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("generate recovery key: %w", err)
	}
	newRecoverySalt := generateSalt()
	recoveryDerivedKey := deriveKeyFromPassword(newRecoveryKey, newRecoverySalt)
	defer zeroBytes(recoveryDerivedKey)
	recoveryEncrypted, err := encryptMasterKey(masterKey, recoveryDerivedKey)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("encrypt recovery key: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("begin change password: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(
		ctx,
		`UPDATE "User"
		    SET "passwordHash" = $2,
		        "vaultSalt" = $3,
		        "encryptedVaultKey" = $4,
		        "vaultKeyIV" = $5,
		        "vaultKeyTag" = $6,
		        "encryptedVaultRecoveryKey" = $7,
		        "vaultRecoveryKeyIV" = $8,
		        "vaultRecoveryKeyTag" = $9,
		        "vaultRecoveryKeySalt" = $10,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
		string(newPasswordHash),
		newVaultSalt,
		newEncryptedVault.Ciphertext,
		newEncryptedVault.IV,
		newEncryptedVault.Tag,
		recoveryEncrypted.Ciphertext,
		recoveryEncrypted.IV,
		recoveryEncrypted.Tag,
		newRecoverySalt,
	); err != nil {
		return passwordChangeResult{}, fmt.Errorf("update password: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "userId" = $1`, userID); err != nil {
		return passwordChangeResult{}, fmt.Errorf("delete refresh tokens: %w", err)
	}

	if err := insertAuditLog(ctx, tx, userID, "PASSWORD_CHANGE", map[string]any{}, ipAddress); err != nil {
		return passwordChangeResult{}, err
	}
	if err := insertAuditLog(ctx, tx, userID, "VAULT_RECOVERY_KEY_GENERATED", map[string]any{}, ipAddress); err != nil {
		return passwordChangeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return passwordChangeResult{}, fmt.Errorf("commit change password: %w", err)
	}

	if s.Redis != nil {
		_ = s.Redis.Del(ctx, "vault:user:"+userID).Err()
	}

	return passwordChangeResult{Success: true, RecoveryKey: newRecoveryKey}, nil
}

func (s Service) InitiateIdentity(ctx context.Context, userID string, payload map[string]json.RawMessage) (identityInitResult, error) {
	purpose, err := parseIdentityPurpose(payload)
	if err != nil {
		return identityInitResult{}, err
	}

	return s.initiateVerification(ctx, userID, purpose)
}

func (s Service) initiateVerification(ctx context.Context, userID, purpose string) (identityInitResult, error) {
	method, err := s.primaryVerificationMethod(ctx, userID)
	if err != nil {
		return identityInitResult{}, err
	}
	if method == "" {
		return identityInitResult{}, errNoVerificationMethod
	}
	if s.Redis == nil {
		return identityInitResult{}, fmt.Errorf("redis is not configured")
	}

	verificationID := uuid.NewString()
	expiresAt := time.Now().Add(verificationSessionTTL)
	session := verificationSession{
		UserID:      userID,
		Method:      method,
		Purpose:     purpose,
		Confirmed:   false,
		ConfirmedAt: nil,
		Attempts:    0,
		ExpiresAt:   expiresAt.UnixMilli(),
	}

	var metadata map[string]interface{}
	switch method {
	case "email":
		email, err := s.loadVerificationEmail(ctx, userID)
		if err != nil {
			return identityInitResult{}, err
		}
		code, err := generateOTPCode()
		if err != nil {
			return identityInitResult{}, err
		}
		session.EmailOtpHash = hashOTP(code)
		if err := s.sendIdentityVerificationCode(ctx, email, code, purpose); err != nil {
			return identityInitResult{}, err
		}
		metadata = map[string]interface{}{"maskedEmail": maskEmail(email)}
	case "sms":
		phoneNumber, err := s.loadVerificationPhone(ctx, userID)
		if err != nil {
			return identityInitResult{}, err
		}
		if err := s.sendOTPToPhone(ctx, userID, phoneNumber); err != nil {
			return identityInitResult{}, err
		}
		metadata = map[string]interface{}{"maskedPhone": maskPhone(phoneNumber)}
	case "webauthn":
		options, err := s.prepareIdentityWebAuthnOptions(ctx, userID)
		if err != nil {
			return identityInitResult{}, err
		}
		session.WebAuthnOption = options
		metadata = map[string]interface{}{"options": options}
	case "totp", "password":
	default:
		return identityInitResult{}, &requestError{status: http.StatusBadRequest, message: "Unsupported verification method."}
	}

	if err := s.putVerificationSession(ctx, verificationID, session, verificationSessionTTL); err != nil {
		return identityInitResult{}, err
	}

	return identityInitResult{
		VerificationID: verificationID,
		Method:         method,
		Metadata:       metadata,
	}, nil
}

func (s Service) ConfirmIdentity(ctx context.Context, userID string, payload map[string]json.RawMessage) (bool, error) {
	confirmation, err := parseIdentityConfirmation(payload)
	if err != nil {
		return false, err
	}

	session, found, err := s.getVerificationSession(ctx, confirmation.VerificationID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, &requestError{status: http.StatusBadRequest, message: "Verification session not found or expired."}
	}
	if session.UserID != userID {
		return false, &requestError{status: http.StatusForbidden, message: "Verification session mismatch."}
	}
	if session.ExpiresAt < time.Now().UnixMilli() {
		_ = s.deleteVerificationSession(ctx, confirmation.VerificationID)
		return false, &requestError{status: http.StatusBadRequest, message: "Verification session expired."}
	}
	if session.Confirmed {
		return false, &requestError{status: http.StatusBadRequest, message: "Verification already confirmed."}
	}

	session.Attempts++
	if session.Attempts > verificationMaxAttempts {
		_ = s.deleteVerificationSession(ctx, confirmation.VerificationID)
		return false, &requestError{status: http.StatusTooManyRequests, message: "Too many verification attempts. Please start a new verification."}
	}

	valid, err := s.verifyIdentityChallenge(ctx, userID, session, confirmation)
	if err != nil {
		return false, err
	}
	if !valid {
		remaining := time.Until(time.UnixMilli(session.ExpiresAt))
		if remaining <= 0 {
			remaining = time.Second
		}
		if err := s.putVerificationSession(ctx, confirmation.VerificationID, session, remaining); err != nil {
			return false, err
		}
		return false, nil
	}

	now := time.Now()
	confirmedAt := now.UnixMilli()
	session.Confirmed = true
	session.ConfirmedAt = &confirmedAt
	session.ExpiresAt = now.Add(verificationConsumeWindow).UnixMilli()
	if err := s.putVerificationSession(ctx, confirmation.VerificationID, session, verificationConsumeWindow); err != nil {
		return false, err
	}

	return true, nil
}

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

func (s Service) loadVerificationEmail(ctx context.Context, userID string) (string, error) {
	if s.DB == nil {
		return "", fmt.Errorf("postgres is not configured")
	}
	var email string
	if err := s.DB.QueryRow(ctx, `SELECT email FROM "User" WHERE id = $1`, userID).Scan(&email); err != nil {
		return "", err
	}
	return email, nil
}

func (s Service) loadVerificationPhone(ctx context.Context, userID string) (string, error) {
	if s.DB == nil {
		return "", fmt.Errorf("postgres is not configured")
	}
	var phoneNumber string
	if err := s.DB.QueryRow(
		ctx,
		`SELECT COALESCE("phoneNumber", '')
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&phoneNumber); err != nil {
		return "", err
	}
	if strings.TrimSpace(phoneNumber) == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "SMS is not configured properly."}
	}
	return phoneNumber, nil
}

func (s Service) verifyIdentityChallenge(ctx context.Context, userID string, session verificationSession, confirmation identityConfirmationPayload) (bool, error) {
	switch session.Method {
	case "email":
		if confirmation.Code == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "Verification code is required."}
		}
		if session.EmailOtpHash == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "Verification session is missing an email code."}
		}
		return timingSafeHexEqual(hashOTP(confirmation.Code), session.EmailOtpHash), nil
	case "totp":
		if confirmation.Code == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "TOTP code is required."}
		}
		user, err := s.loadVerificationTOTPUser(ctx, userID)
		if err != nil {
			return false, err
		}
		secret, err := s.resolveVerificationTOTPSecret(ctx, userID, user)
		if err != nil {
			return false, err
		}
		if secret == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "TOTP is not configured properly."}
		}
		return verifyTOTP(secret, confirmation.Code, time.Now()), nil
	case "sms":
		if confirmation.Code == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "SMS code is required."}
		}
		return s.verifySMSOTP(ctx, userID, confirmation.Code)
	case "webauthn":
		if session.WebAuthnOption == nil {
			return false, &requestError{status: http.StatusBadRequest, message: "Verification session is missing WebAuthn options."}
		}
		return s.verifyIdentityWebAuthn(ctx, userID, session.WebAuthnOption, confirmation.Credential)
	case "password":
		if confirmation.Password == "" {
			return false, &requestError{status: http.StatusBadRequest, message: "Password is required."}
		}
		return s.verifyPassword(ctx, userID, confirmation.Password)
	default:
		return false, &requestError{status: http.StatusBadRequest, message: "Unsupported verification method."}
	}
}

type verificationTOTPUser struct {
	EncryptedTOTPSecret *string
	TOTPSecretIV        *string
	TOTPSecretTag       *string
	TOTPSecret          *string
}

func (s Service) loadVerificationTOTPUser(ctx context.Context, userID string) (verificationTOTPUser, error) {
	if s.DB == nil {
		return verificationTOTPUser{}, fmt.Errorf("postgres is not configured")
	}

	var user verificationTOTPUser
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "encryptedTotpSecret", "totpSecretIV", "totpSecretTag", "totpSecret"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&user.EncryptedTOTPSecret, &user.TOTPSecretIV, &user.TOTPSecretTag, &user.TOTPSecret); err != nil {
		return verificationTOTPUser{}, err
	}
	return user, nil
}

func (s Service) resolveVerificationTOTPSecret(ctx context.Context, userID string, user verificationTOTPUser) (string, error) {
	if user.TOTPSecret != nil && strings.TrimSpace(*user.TOTPSecret) != "" {
		return strings.TrimSpace(*user.TOTPSecret), nil
	}
	if user.EncryptedTOTPSecret == nil || user.TOTPSecretIV == nil || user.TOTPSecretTag == nil ||
		strings.TrimSpace(*user.EncryptedTOTPSecret) == "" || strings.TrimSpace(*user.TOTPSecretIV) == "" || strings.TrimSpace(*user.TOTPSecretTag) == "" {
		return "", nil
	}

	masterKey, err := s.getVaultMasterKey(ctx, userID)
	if err != nil {
		return "", err
	}
	if len(masterKey) == 0 {
		return "", nil
	}
	defer zeroBytes(masterKey)

	secret, err := decryptEncryptedField(masterKey, encryptedField{
		Ciphertext: *user.EncryptedTOTPSecret,
		IV:         *user.TOTPSecretIV,
		Tag:        *user.TOTPSecretTag,
	})
	if err != nil {
		return "", fmt.Errorf("decrypt totp secret: %w", err)
	}
	return secret, nil
}

func (s Service) primaryVerificationMethod(ctx context.Context, userID string) (string, error) {
	if s.DB == nil {
		return "", fmt.Errorf("postgres is not configured")
	}

	var (
		emailVerified bool
		totpEnabled   bool
		smsMfaEnabled bool
		phoneVerified bool
		webauthn      bool
		hasPassword   bool
	)

	err := s.DB.QueryRow(
		ctx,
		`SELECT
			"emailVerified",
			"totpEnabled",
			"smsMfaEnabled",
			"phoneVerified",
			"webauthnEnabled",
			"passwordHash" IS NOT NULL
		FROM "User"
		WHERE id = $1`,
		userID,
	).Scan(&emailVerified, &totpEnabled, &smsMfaEnabled, &phoneVerified, &webauthn, &hasPassword)
	if err != nil {
		return "", err
	}

	switch {
	case emailVerificationConfigured() && emailVerified:
		return "email", nil
	case totpEnabled:
		return "totp", nil
	case smsMfaEnabled && phoneVerified:
		return "sms", nil
	case webauthn:
		return "webauthn", nil
	case hasPassword:
		return "password", nil
	default:
		return "", nil
	}
}
