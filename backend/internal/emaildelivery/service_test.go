package emaildelivery

import "testing"

func TestStatusFromEnv(t *testing.T) {
	t.Run("smtp dev mode without host", func(t *testing.T) {
		t.Setenv("EMAIL_PROVIDER", "")
		status := StatusFromEnv()
		if status.Provider != "smtp" || status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("smtp configured", func(t *testing.T) {
		t.Setenv("EMAIL_PROVIDER", "smtp")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_FROM", "noreply@example.com")
		status := StatusFromEnv()
		if status.Provider != "smtp" || !status.Configured || status.From != "noreply@example.com" {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("sendgrid configured", func(t *testing.T) {
		t.Setenv("EMAIL_PROVIDER", "sendgrid")
		t.Setenv("SENDGRID_API_KEY", "secret")
		status := StatusFromEnv()
		if status.Provider != "sendgrid" || !status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("mailgun incomplete", func(t *testing.T) {
		t.Setenv("EMAIL_PROVIDER", "mailgun")
		t.Setenv("MAILGUN_API_KEY", "secret")
		status := StatusFromEnv()
		if status.Provider != "mailgun" || status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("ses considered configured", func(t *testing.T) {
		t.Setenv("EMAIL_PROVIDER", "ses")
		status := StatusFromEnv()
		if status.Provider != "ses" || !status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})
}
