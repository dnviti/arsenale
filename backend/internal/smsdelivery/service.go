package smsdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type Message struct {
	To   string
	Body string
}

type Status struct {
	Provider   string
	Configured bool
}

func StatusFromEnv() Status {
	switch provider := strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER"))); provider {
	case "":
		return Status{Provider: "none", Configured: false}
	case "twilio":
		return Status{
			Provider:   provider,
			Configured: strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID")) != "" && loadSecretEnv("TWILIO_AUTH_TOKEN", "TWILIO_AUTH_TOKEN_FILE") != "" && strings.TrimSpace(os.Getenv("TWILIO_FROM_NUMBER")) != "",
		}
	case "vonage":
		return Status{
			Provider:   provider,
			Configured: strings.TrimSpace(os.Getenv("VONAGE_API_KEY")) != "" && loadSecretEnv("VONAGE_API_SECRET", "VONAGE_API_SECRET_FILE") != "" && strings.TrimSpace(os.Getenv("VONAGE_FROM_NUMBER")) != "",
		}
	case "sns":
		return Status{
			Provider:   provider,
			Configured: true,
		}
	default:
		return Status{
			Provider:   provider,
			Configured: false,
		}
	}
}

func Send(ctx context.Context, msg Message) error {
	status := StatusFromEnv()
	switch status.Provider {
	case "none":
		slog.Info("sms delivery unavailable; falling back to dev logging", "to", msg.To)
		return nil
	case "twilio":
		if !status.Configured {
			return errors.New("twilio sms provider is not fully configured")
		}
		return sendTwilio(ctx, msg)
	case "sns":
		return sendSNS(ctx, msg)
	case "vonage":
		if !status.Configured {
			return errors.New("vonage sms provider is not fully configured")
		}
		return sendVonage(ctx, msg)
	default:
		return fmt.Errorf("unsupported SMS_PROVIDER %q", status.Provider)
	}
}

func sendTwilio(ctx context.Context, msg Message) error {
	accountSID := strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID"))
	authToken := loadSecretEnv("TWILIO_AUTH_TOKEN", "TWILIO_AUTH_TOKEN_FILE")
	from := strings.TrimSpace(os.Getenv("TWILIO_FROM_NUMBER"))

	form := url.Values{
		"From": {from},
		"To":   {msg.To},
		"Body": {msg.Body},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", url.PathEscape(accountSID)),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("build twilio request: %w", err)
	}
	req.SetBasicAuth(accountSID, authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return doHTTPSend(req, "twilio", func(statusCode int, _ []byte) bool {
		return statusCode >= 200 && statusCode < 300
	})
}

func sendVonage(ctx context.Context, msg Message) error {
	form := url.Values{
		"api_key":    {strings.TrimSpace(os.Getenv("VONAGE_API_KEY"))},
		"api_secret": {loadSecretEnv("VONAGE_API_SECRET", "VONAGE_API_SECRET_FILE")},
		"from":       {strings.TrimSpace(os.Getenv("VONAGE_FROM_NUMBER"))},
		"to":         {msg.To},
		"text":       {msg.Body},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://rest.nexmo.com/sms/json",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("build vonage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return doHTTPSend(req, "vonage", func(statusCode int, body []byte) bool {
		if statusCode < 200 || statusCode >= 300 {
			return false
		}
		var payload struct {
			Messages []struct {
				Status string `json:"status"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return false
		}
		return len(payload.Messages) > 0 && payload.Messages[0].Status == "0"
	})
}

func sendSNS(ctx context.Context, msg Message) error {
	region := strings.TrimSpace(os.Getenv("AWS_SNS_REGION"))
	if region == "" {
		region = "us-east-1"
	}

	var loadOptions []func(*awsconfig.LoadOptions) error
	loadOptions = append(loadOptions, awsconfig.WithRegion(region))

	accessKeyID := strings.TrimSpace(os.Getenv("AWS_SNS_ACCESS_KEY_ID"))
	secretAccessKey := loadSecretEnv("AWS_SNS_SECRET_ACCESS_KEY", "AWS_SNS_SECRET_ACCESS_KEY_FILE")
	if accessKeyID != "" && secretAccessKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return fmt.Errorf("load sns config: %w", err)
	}

	client := sns.NewFromConfig(cfg)
	if _, err := client.Publish(ctx, &sns.PublishInput{
		PhoneNumber: &msg.To,
		Message:     &msg.Body,
	}); err != nil {
		return fmt.Errorf("send sns sms: %w", err)
	}
	return nil
}

func doHTTPSend(req *http.Request, provider string, ok func(statusCode int, body []byte) bool) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send %s sms: %w", provider, err)
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
	return fmt.Errorf("%s sms rejected request: %s", provider, trimmed)
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
