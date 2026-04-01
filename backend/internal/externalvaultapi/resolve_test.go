package externalvaultapi

import "testing"

func TestMapSecretToCredentialsSupportsCommonFieldNames(t *testing.T) {
	creds, err := mapSecretToCredentials(map[string]string{
		"user":        "alice",
		"pass":        "secret",
		"domain":      "corp",
		"private_key": "PRIVATE",
		"passphrase":  "phrase",
	}, "secret/path")
	if err != nil {
		t.Fatalf("mapSecretToCredentials returned error: %v", err)
	}
	if creds.Username != "alice" || creds.Password != "secret" {
		t.Fatalf("unexpected username/password: %#v", creds)
	}
	if creds.Domain != "corp" || creds.PrivateKey != "PRIVATE" || creds.Passphrase != "phrase" {
		t.Fatalf("unexpected optional fields: %#v", creds)
	}
}

func TestMapSecretToCredentialsRejectsMissingUsernameAndPassword(t *testing.T) {
	if _, err := mapSecretToCredentials(map[string]string{"privateKey": "PRIVATE"}, "secret/path"); err == nil {
		t.Fatal("expected error for secret without username/password")
	}
}

func TestSplitAWSSecretPath(t *testing.T) {
	secretID, versionStage := splitAWSSecretPath("my-secret#AWSPREVIOUS")
	if secretID != "my-secret" || versionStage != "AWSPREVIOUS" {
		t.Fatalf("unexpected AWS secret path split: %q %q", secretID, versionStage)
	}
}

func TestSplitAzureSecretPath(t *testing.T) {
	name, version := splitAzureSecretPath("my-secret/version-id")
	if name != "my-secret" || version != "version-id" {
		t.Fatalf("unexpected Azure secret path split: %q %q", name, version)
	}
}

func TestSecretPathResourceDefaultsToLatestVersion(t *testing.T) {
	resource := secretPathResource("db-secret", "project-123")
	if resource != "projects/project-123/secrets/db-secret/versions/latest" {
		t.Fatalf("unexpected GCP secret resource: %q", resource)
	}
}
