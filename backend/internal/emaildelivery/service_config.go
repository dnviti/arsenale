package emaildelivery

import (
	"os"
	"strings"
)

func StatusFromEnv() Status {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_PROVIDER")))
	if provider == "" {
		provider = "smtp"
	}

	status := Status{
		Provider: provider,
		From:     emailFrom(),
	}

	switch provider {
	case "smtp":
		status.Configured = strings.TrimSpace(os.Getenv("SMTP_HOST")) != ""
	case "sendgrid":
		status.Configured = loadSecretEnv("SENDGRID_API_KEY", "SENDGRID_API_KEY_FILE") != ""
	case "ses":
		status.Configured = true
	case "resend":
		status.Configured = loadSecretEnv("RESEND_API_KEY", "RESEND_API_KEY_FILE") != ""
	case "mailgun":
		status.Configured = loadSecretEnv("MAILGUN_API_KEY", "MAILGUN_API_KEY_FILE") != "" && strings.TrimSpace(os.Getenv("MAILGUN_DOMAIN")) != ""
	default:
		status.Configured = false
	}

	return status
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

func emailFrom() string {
	if value := strings.TrimSpace(os.Getenv("SMTP_FROM")); value != "" {
		return value
	}
	return "noreply@localhost"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func awsString(value string) *string {
	return &value
}
