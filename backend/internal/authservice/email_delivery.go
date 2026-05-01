package authservice

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/emaildelivery"
)

const (
	emailVerifyTTL    = 24 * 60 * 60
	resendCooldownSec = 60
	passwordResetTTL  = 60 * 60
)

func (s Service) clientURL() string {
	if value := strings.TrimSpace(s.ClientURL); value != "" {
		return strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(os.Getenv("CLIENT_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "https://localhost:3000"
}

func emailFlowConfigured() bool {
	provider := strings.TrimSpace(strings.ToLower(os.Getenv("EMAIL_PROVIDER")))
	if provider == "" {
		provider = "smtp"
	}

	switch provider {
	case "smtp":
		return strings.TrimSpace(os.Getenv("SMTP_HOST")) != ""
	case "sendgrid":
		return loadSecretEnv("SENDGRID_API_KEY", "SENDGRID_API_KEY_FILE") != ""
	case "ses":
		return strings.TrimSpace(os.Getenv("AWS_SES_ACCESS_KEY_ID")) != "" &&
			loadSecretEnv("AWS_SES_SECRET_ACCESS_KEY", "AWS_SES_SECRET_ACCESS_KEY_FILE") != ""
	case "resend":
		return loadSecretEnv("RESEND_API_KEY", "RESEND_API_KEY_FILE") != ""
	case "mailgun":
		return loadSecretEnv("MAILGUN_API_KEY", "MAILGUN_API_KEY_FILE") != "" &&
			strings.TrimSpace(os.Getenv("MAILGUN_DOMAIN")) != ""
	default:
		return false
	}
}

func loadSecretEnv(name, fileName string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	if path := strings.TrimSpace(os.Getenv(fileName)); path != "" {
		payload, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(payload))
		}
	}
	return ""
}

func (s Service) logVerificationEmail(to, token string) {
	slog.Info("email verification link (dev mode)", "to", to, "verifyUrl", s.clientURL()+"/api/auth/verify-email?token="+token)
}

func (s Service) logPasswordResetEmail(to, token string) {
	slog.Info("password reset link (dev mode)", "to", to, "resetUrl", s.clientURL()+"/reset-password?token="+token)
}

func (s Service) sendVerificationEmail(ctx context.Context, to, token string) error {
	status := emaildelivery.StatusFromEnv()
	verifyURL := s.clientURL() + "/api/auth/verify-email?token=" + token
	if !status.Configured {
		s.logVerificationEmail(to, token)
		return nil
	}
	return emaildelivery.Send(ctx, emaildelivery.Message{
		To:      to,
		Subject: "Verify your email - Arsenale",
		HTML: "<h2>Email Verification</h2>" +
			"<p>Click the link below to verify your email address:</p>" +
			`<p><a href="` + verifyURL + `">` + verifyURL + `</a></p>` +
			"<p>This link expires in 24 hours.</p>" +
			"<p>If you did not create an account, you can ignore this email.</p>",
		Text: "Verify your email: " + verifyURL + "\n\nThis link expires in 24 hours. If you did not create an account, ignore this email.",
	})
}

func (s Service) sendPasswordResetEmail(ctx context.Context, to, token string) error {
	status := emaildelivery.StatusFromEnv()
	resetURL := s.clientURL() + "/reset-password?token=" + token
	if !status.Configured {
		s.logPasswordResetEmail(to, token)
		return nil
	}
	return emaildelivery.Send(ctx, emaildelivery.Message{
		To:      to,
		Subject: "Password Reset - Arsenale",
		HTML: "<h2>Password Reset Request</h2>" +
			"<p>You requested a password reset. Click the link below to set a new password:</p>" +
			`<p><a href="` + resetURL + `">` + resetURL + `</a></p>` +
			"<p>This link expires in 1 hour.</p>" +
			"<p>If you did not request this, you can safely ignore this email. Your password will not be changed.</p>",
		Text: "Password Reset: " + resetURL + "\n\nThis link expires in 1 hour. If you did not request this, ignore this email.",
	})
}
