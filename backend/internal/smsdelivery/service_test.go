package smsdelivery

import "testing"

func TestStatusFromEnv(t *testing.T) {
	t.Run("none when provider unset", func(t *testing.T) {
		t.Setenv("SMS_PROVIDER", "")
		status := StatusFromEnv()
		if status.Provider != "none" || status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("twilio configured", func(t *testing.T) {
		t.Setenv("SMS_PROVIDER", "twilio")
		t.Setenv("TWILIO_ACCOUNT_SID", "AC123")
		t.Setenv("TWILIO_AUTH_TOKEN", "secret")
		t.Setenv("TWILIO_FROM_NUMBER", "+15551234567")
		status := StatusFromEnv()
		if status.Provider != "twilio" || !status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("twilio incomplete", func(t *testing.T) {
		t.Setenv("SMS_PROVIDER", "twilio")
		t.Setenv("TWILIO_ACCOUNT_SID", "AC123")
		status := StatusFromEnv()
		if status.Provider != "twilio" || status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("vonage configured", func(t *testing.T) {
		t.Setenv("SMS_PROVIDER", "vonage")
		t.Setenv("VONAGE_API_KEY", "key")
		t.Setenv("VONAGE_API_SECRET", "secret")
		t.Setenv("VONAGE_FROM_NUMBER", "Arsenale")
		status := StatusFromEnv()
		if status.Provider != "vonage" || !status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})

	t.Run("sns defaults to configured", func(t *testing.T) {
		t.Setenv("SMS_PROVIDER", "sns")
		status := StatusFromEnv()
		if status.Provider != "sns" || !status.Configured {
			t.Fatalf("unexpected status: %+v", status)
		}
	})
}
