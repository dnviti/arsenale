package gateways

import (
	"strings"
	"testing"
)

func TestNormalizeCreateTemplatePayloadManagedDefaults(t *testing.T) {
	payload := createTemplatePayload{
		Name: "Managed SSH",
		Type: "managed_ssh",
	}

	normalized, err := normalizeCreateTemplatePayload(payload)
	if err != nil {
		t.Fatalf("normalizeCreateTemplatePayload returned error: %v", err)
	}
	if normalized.Type != "MANAGED_SSH" {
		t.Fatalf("expected type MANAGED_SSH, got %q", normalized.Type)
	}
	if normalized.Host != "" {
		t.Fatalf("expected managed template host to be empty, got %q", normalized.Host)
	}
	if normalized.Port != 2222 {
		t.Fatalf("expected default managed SSH port 2222, got %d", normalized.Port)
	}
}

func TestNormalizeCreateTemplatePayloadRequiresPortForBastion(t *testing.T) {
	_, err := normalizeCreateTemplatePayload(createTemplatePayload{
		Name: "Bastion",
		Type: "SSH_BASTION",
	})
	if err == nil {
		t.Fatal("expected normalizeCreateTemplatePayload to reject missing bastion port")
	}
}

func TestValidateUpdateTemplatePayloadRejectsInvalidReplicaRange(t *testing.T) {
	minReplicas := 5
	maxReplicas := 3
	err := validateUpdateTemplatePayload(updateTemplatePayload{
		MinReplicas: optionalInt{Present: true, Value: &minReplicas},
		MaxReplicas: optionalInt{Present: true, Value: &maxReplicas},
	})
	if err == nil {
		t.Fatal("expected validateUpdateTemplatePayload to reject minReplicas > maxReplicas")
	}
}

func TestBuildTemplateDeploymentName(t *testing.T) {
	name := buildTemplateDeploymentName("12345678-tenant", "Managed SSH")

	if !strings.HasPrefix(name, "12345678-Managed SSH-") {
		t.Fatalf("unexpected template deployment name prefix: %q", name)
	}
	if len(name) != len("12345678-Managed SSH-")+6 {
		t.Fatalf("expected 6-char suffix in deployment name, got %q", name)
	}
}

func TestInsertGatewayTemplateSQLSetsTimestamps(t *testing.T) {
	if !strings.Contains(insertGatewayTemplateSQL, `"createdAt"`) {
		t.Fatal(`expected insertGatewayTemplateSQL to populate "createdAt"`)
	}
	if !strings.Contains(insertGatewayTemplateSQL, `"updatedAt"`) {
		t.Fatal(`expected insertGatewayTemplateSQL to populate "updatedAt"`)
	}
	if strings.Count(insertGatewayTemplateSQL, "NOW()") < 2 {
		t.Fatal(`expected insertGatewayTemplateSQL to set both timestamps with NOW()`)
	}
}

func TestInsertGatewayFromTemplateSQLSetsTimestamps(t *testing.T) {
	if !strings.Contains(insertGatewayFromTemplateSQL, `"createdAt"`) {
		t.Fatal(`expected insertGatewayFromTemplateSQL to populate "createdAt"`)
	}
	if !strings.Contains(insertGatewayFromTemplateSQL, `"updatedAt"`) {
		t.Fatal(`expected insertGatewayFromTemplateSQL to populate "updatedAt"`)
	}
	if strings.Count(insertGatewayFromTemplateSQL, "NOW()") < 2 {
		t.Fatal(`expected insertGatewayFromTemplateSQL to set both timestamps with NOW()`)
	}
}
