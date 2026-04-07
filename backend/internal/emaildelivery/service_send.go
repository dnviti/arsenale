package emaildelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func Send(ctx context.Context, msg Message) error {
	status := StatusFromEnv()
	switch status.Provider {
	case "smtp":
		if !status.Configured {
			return nil
		}
		return sendSMTP(msg)
	case "sendgrid":
		if !status.Configured {
			return fmt.Errorf("sendgrid email provider is not configured")
		}
		return sendSendgrid(ctx, msg)
	case "ses":
		return sendSES(ctx, msg)
	case "resend":
		if !status.Configured {
			return fmt.Errorf("resend email provider is not configured")
		}
		return sendResend(ctx, msg)
	case "mailgun":
		if !status.Configured {
			return fmt.Errorf("mailgun email provider is not configured")
		}
		return sendMailgun(ctx, msg)
	default:
		return fmt.Errorf("unsupported EMAIL_PROVIDER %q", status.Provider)
	}
}

func sendJSONAPI(ctx context.Context, provider, endpoint, apiKey string, payload any, ok func(statusCode int) bool) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s email request: %w", provider, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s request: %w", provider, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	return doHTTP(req, provider, func(statusCode int, _ []byte) bool {
		return ok(statusCode)
	})
}

func doHTTP(req *http.Request, provider string, ok func(statusCode int, body []byte) bool) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send %s email: %w", provider, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("read %s response: %w", provider, err)
	}
	if ok(resp.StatusCode, body) {
		return nil
	}

	trimmed := strings.TrimSpace(string(bytes.TrimSpace(body)))
	if trimmed == "" {
		trimmed = resp.Status
	}
	return fmt.Errorf("%s email rejected request: %s", provider, trimmed)
}
