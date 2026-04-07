package emaildelivery

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func sendSendgrid(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": []map[string]string{
					{"email": msg.To},
				},
			},
		},
		"from": map[string]string{
			"email": emailFrom(),
		},
		"subject": msg.Subject,
		"content": []map[string]string{
			{"type": "text/plain", "value": firstNonEmpty(msg.Text, msg.Subject)},
		},
	}
	if msg.HTML != "" {
		payload["content"] = append(payload["content"].([]map[string]string), map[string]string{"type": "text/html", "value": msg.HTML})
	}
	return sendJSONAPI(ctx, "sendgrid", "https://api.sendgrid.com/v3/mail/send", loadSecretEnv("SENDGRID_API_KEY", "SENDGRID_API_KEY_FILE"), payload, func(statusCode int) bool {
		return statusCode == http.StatusAccepted
	})
}

func sendResend(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"from":    emailFrom(),
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"html":    msg.HTML,
		"text":    msg.Text,
	}
	return sendJSONAPI(ctx, "resend", "https://api.resend.com/emails", loadSecretEnv("RESEND_API_KEY", "RESEND_API_KEY_FILE"), payload, func(statusCode int) bool {
		return statusCode >= 200 && statusCode < 300
	})
}

func sendMailgun(ctx context.Context, msg Message) error {
	domain := strings.TrimSpace(os.Getenv("MAILGUN_DOMAIN"))
	baseURL := "https://api.mailgun.net"
	if strings.EqualFold(strings.TrimSpace(os.Getenv("MAILGUN_REGION")), "eu") {
		baseURL = "https://api.eu.mailgun.net"
	}

	form := url.Values{
		"from":    {emailFrom()},
		"to":      {msg.To},
		"subject": {msg.Subject},
		"html":    {msg.HTML},
		"text":    {firstNonEmpty(msg.Text, msg.Subject)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v3/"+url.PathEscape(domain)+"/messages", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build mailgun request: %w", err)
	}
	req.SetBasicAuth("api", loadSecretEnv("MAILGUN_API_KEY", "MAILGUN_API_KEY_FILE"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doHTTP(req, "mailgun", func(statusCode int, _ []byte) bool {
		return statusCode >= 200 && statusCode < 300
	})
}
