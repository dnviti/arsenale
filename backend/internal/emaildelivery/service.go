package emaildelivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

type Status struct {
	Provider   string
	Configured bool
	From       string
}

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

func sendSMTP(msg Message) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := emailFrom()
	addr := host + ":" + port

	raw, err := buildSMTPMessage(from, msg)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := loadSecretEnv("SMTP_PASS", "SMTP_PASS_FILE")
	var auth smtp.Auth
	if user != "" || pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	if port == "465" {
		return sendImplicitTLSSMTP(addr, host, auth, from, msg.To, raw)
	}
	return smtp.SendMail(addr, auth, from, []string{msg.To}, raw)
}

func sendImplicitTLSSMTP(addr, host string, auth smtp.Auth, from, to string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("dial smtp tls: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp finalize body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func buildSMTPMessage(from string, msg Message) ([]byte, error) {
	text := msg.Text
	if text == "" {
		text = msg.Subject
	}

	var body string
	headers := []string{
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
	}

	switch {
	case msg.HTML != "" && text != "":
		boundary := "arsenale-email-boundary"
		headers = append(headers, `Content-Type: multipart/alternative; boundary="`+boundary+`"`)
		body = "--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + text + "\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n\r\n" + msg.HTML + "\r\n" +
			"--" + boundary + "--\r\n"
	case msg.HTML != "":
		headers = append(headers, "Content-Type: text/html; charset=UTF-8")
		body = msg.HTML
	default:
		headers = append(headers, "Content-Type: text/plain; charset=UTF-8")
		body = text
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body), nil
}

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

func sendSES(ctx context.Context, msg Message) error {
	region := strings.TrimSpace(os.Getenv("AWS_SES_REGION"))
	if region == "" {
		region = "us-east-1"
	}

	var loadOptions []func(*awsconfig.LoadOptions) error
	loadOptions = append(loadOptions, awsconfig.WithRegion(region))

	accessKeyID := strings.TrimSpace(os.Getenv("AWS_SES_ACCESS_KEY_ID"))
	secretAccessKey := loadSecretEnv("AWS_SES_SECRET_ACCESS_KEY", "AWS_SES_SECRET_ACCESS_KEY_FILE")
	if accessKeyID != "" && secretAccessKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return fmt.Errorf("load ses config: %w", err)
	}

	client := ses.NewFromConfig(cfg)
	input := &ses.SendEmailInput{
		Source: awsString(emailFrom()),
		Destination: &sestypes.Destination{
			ToAddresses: []string{msg.To},
		},
		Message: &sestypes.Message{
			Subject: &sestypes.Content{Data: awsString(msg.Subject)},
			Body: &sestypes.Body{
				Html: &sestypes.Content{Data: awsString(msg.HTML)},
				Text: &sestypes.Content{Data: awsString(firstNonEmpty(msg.Text, msg.Subject))},
			},
		},
	}
	if _, err := client.SendEmail(ctx, input); err != nil {
		return fmt.Errorf("send ses email: %w", err)
	}
	return nil
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
