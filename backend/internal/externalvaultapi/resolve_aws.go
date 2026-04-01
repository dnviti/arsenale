package externalvaultapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type awsAuthPayload struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	Region          string `json:"region"`
}

func (s Service) readAWSSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	var auth awsAuthPayload
	if err := s.decodeProviderAuth(record, &auth); err != nil {
		return nil, err
	}

	region := strings.TrimSpace(auth.Region)
	if region == "" {
		region = "us-east-1"
	}

	loaders := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if record.AuthMethod == "IAM_ACCESS_KEY" {
		if strings.TrimSpace(auth.AccessKeyID) == "" || strings.TrimSpace(auth.SecretAccessKey) == "" {
			return nil, &ResolveError{Status: 400, Message: "AWS auth payload must contain accessKeyId and secretAccessKey"}
		}
		loaders = append(loaders, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(auth.AccessKeyID),
			strings.TrimSpace(auth.SecretAccessKey),
			strings.TrimSpace(auth.SessionToken),
		)))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loaders...)
	if err != nil {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("load AWS config: %v", err)}
	}

	client := secretsmanager.NewFromConfig(cfg)
	secretID, versionStage := splitAWSSecretPath(secretPath)
	input := &secretsmanager.GetSecretValueInput{SecretId: &secretID}
	if versionStage != "" {
		input.VersionStage = &versionStage
	}

	resp, err := client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("AWS Secrets Manager API error: %v", err)}
	}
	if resp.SecretString == nil {
		if resp.SecretBinary != nil {
			return parseSecretResponse(resp.SecretBinary), nil
		}
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Secret %q has no string value (binary secrets are not supported)", secretID)}
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(*resp.SecretString), &parsed); err == nil {
		return parsed, nil
	}
	return map[string]string{"value": *resp.SecretString}, nil
}

func splitAWSSecretPath(secretPath string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(secretPath), "#", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
