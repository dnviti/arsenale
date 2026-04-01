package users

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/emaildelivery"
	"github.com/dnviti/arsenale/backend/internal/smsdelivery"
	"github.com/jackc/pgx/v5"
)

const smsOTPTTL = 5 * time.Minute

var verificationCodePattern = regexp.MustCompile(`^\d{6}$`)

func generateOTPCode() (string, error) {
	const max = 1_000_000
	const limit = ^uint32(0) - (^uint32(0) % max)
	for {
		var raw [4]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", fmt.Errorf("generate otp: %w", err)
		}
		value := uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
		if value < limit {
			return fmt.Sprintf("%06d", value%max), nil
		}
	}
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func timingSafeHexEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}

	bufA := []byte(strings.ToLower(strings.TrimSpace(a)))
	bufB := []byte(strings.ToLower(strings.TrimSpace(b)))
	if len(bufA) != len(bufB) {
		return false
	}
	return subtle.ConstantTimeCompare(bufA, bufB) == 1
}

func verifyTOTP(secret, code string, now time.Time) bool {
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return false
	}
	counter := now.UTC().Unix() / 30
	for _, offset := range []int64{-1, 0, 1} {
		if generateTOTPValue(key, counter+offset) == code {
			return true
		}
	}
	return false
}

func decodeTOTPSecret(secret string) ([]byte, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(secret))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(cleaned)
}

func generateTOTPValue(key []byte, counter int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := int(binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff)
	return fmt.Sprintf("%06d", value%1_000_000)
}

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return email
	}
	local := email[:at]
	if len(local) <= 2 {
		return local + "***" + email[at:]
	}
	return local[:2] + "***" + email[at:]
}

func maskPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 5 {
		return trimmed
	}
	return "+" + strings.Repeat("*", max(0, len(trimmed)-5)) + trimmed[len(trimmed)-4:]
}

func validateVerificationCode(code, fieldName string) error {
	if !verificationCodePattern.MatchString(strings.TrimSpace(code)) {
		return &requestError{status: http.StatusBadRequest, message: fieldName + " must be a 6-digit code"}
	}
	return nil
}

func (s Service) sendIdentityVerificationCode(ctx context.Context, to, code, purpose string) error {
	status := emaildelivery.StatusFromEnv()
	if !status.Configured {
		slog.Info("users dev identity verification email", "to", to, "code", code, "purpose", purpose)
		return nil
	}

	return emaildelivery.Send(ctx, emaildelivery.Message{
		To:      to,
		Subject: "Identity Verification Code - Arsenale",
		HTML: "<h2>Identity Verification</h2>" +
			"<p>Your verification code is: <strong>" + code + "</strong></p>" +
			"<p>This code is needed for: <strong>" + purpose + "</strong></p>" +
			"<p>The code expires in 15 minutes.</p>" +
			"<p>If you did not request this, please secure your account immediately.</p>",
		Text: "Your identity verification code is: " + code +
			"\n\nPurpose: " + purpose +
			"\nThis code expires in 15 minutes." +
			"\nIf you did not request this, please secure your account immediately.",
	})
}

func (s Service) sendEmailChangeCode(ctx context.Context, to, code string, isOldEmail bool) error {
	status := emaildelivery.StatusFromEnv()
	label := "Confirm your new email address"
	if isOldEmail {
		label = "Confirm that you want to change your email address"
	}

	if !status.Configured {
		slog.Info("users dev email-change code", "to", to, "code", code, "oldEmail", isOldEmail)
		return nil
	}

	return emaildelivery.Send(ctx, emaildelivery.Message{
		To:      to,
		Subject: "Email Change Verification - Arsenale",
		HTML: "<h2>Email Change Verification</h2>" +
			"<p>" + label + "</p>" +
			"<p>Your verification code is: <strong>" + code + "</strong></p>" +
			"<p>The code expires in 15 minutes.</p>" +
			"<p>If you did not request this, please secure your account immediately.</p>",
		Text: label +
			"\n\nYour verification code is: " + code +
			"\nThis code expires in 15 minutes." +
			"\nIf you did not request this, please secure your account immediately.",
	})
}

func (s Service) sendOTPToPhone(ctx context.Context, userID, phoneNumber string) error {
	if s.DB == nil {
		return fmt.Errorf("postgres is not configured")
	}

	code, err := generateOTPCode()
	if err != nil {
		return err
	}
	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "smsOtpHash" = $2,
		        "smsOtpExpiresAt" = $3,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
		hashOTP(code),
		time.Now().Add(smsOTPTTL),
	); err != nil {
		return fmt.Errorf("store sms otp: %w", err)
	}

	status := smsdelivery.StatusFromEnv()
	if err := smsdelivery.Send(ctx, smsdelivery.Message{
		To:   phoneNumber,
		Body: fmt.Sprintf("Your Arsenale verification code is: %s. It expires in 5 minutes.", code),
	}); err != nil {
		return err
	}

	if !status.Configured {
		slog.Info("users dev sms otp", "userId", userID, "phone", phoneNumber, "code", code)
	}
	return nil
}

func (s Service) verifySMSOTP(ctx context.Context, userID, code string) (bool, error) {
	if s.DB == nil {
		return false, fmt.Errorf("postgres is not configured")
	}

	var (
		storedHash *string
		expiresAt  *time.Time
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "smsOtpHash", "smsOtpExpiresAt"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&storedHash, &expiresAt); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("load sms otp: %w", err)
	}

	if storedHash == nil || expiresAt == nil {
		return false, nil
	}
	if expiresAt.Before(time.Now()) {
		_, _ = s.DB.Exec(
			ctx,
			`UPDATE "User"
			    SET "smsOtpHash" = NULL,
			        "smsOtpExpiresAt" = NULL,
			        "updatedAt" = NOW()
			  WHERE id = $1`,
			userID,
		)
		return false, nil
	}
	if !timingSafeHexEqual(hashOTP(code), *storedHash) {
		return false, nil
	}

	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "smsOtpHash" = NULL,
		        "smsOtpExpiresAt" = NULL,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
	); err != nil {
		return false, fmt.Errorf("clear sms otp: %w", err)
	}
	return true, nil
}
