package emaildelivery

import (
	"context"
	"fmt"
	"os"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

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
