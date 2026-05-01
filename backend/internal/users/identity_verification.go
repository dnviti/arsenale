package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

	// The first configured method wins so repeat prompts stay predictable for the user.
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
