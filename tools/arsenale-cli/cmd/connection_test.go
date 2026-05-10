package cmd

import (
	"encoding/json"
	"testing"
)

func TestNormalizeConnectionListOutputFlattensGroupedTableRows(t *testing.T) {
	input := []byte(`{
		"own":[{"id":"own-1","name":"Alpha","type":"SSH","host":"ssh.example","port":22}],
		"shared":[{"id":"shared-1","name":"Beta","type":"RDP","host":"rdp.example","port":3389}],
		"team":[{"id":"team-1","name":"Gamma","type":"VNC","host":"vnc.example","port":5900}]
	}`)

	displayBody, tableBody, err := normalizeConnectionListOutput(input, "")
	if err != nil {
		t.Fatalf("normalizeConnectionListOutput() error = %v", err)
	}

	var groups connectionListGroups
	if err := json.Unmarshal(displayBody, &groups); err != nil {
		t.Fatalf("unmarshal grouped display body: %v", err)
	}
	if len(groups.Own) != 1 || len(groups.Shared) != 1 || len(groups.Team) != 1 {
		t.Fatalf("group counts = own:%d shared:%d team:%d", len(groups.Own), len(groups.Shared), len(groups.Team))
	}

	var rows []map[string]any
	if err := json.Unmarshal(tableBody, &rows); err != nil {
		t.Fatalf("unmarshal table body: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0]["scope"] != "private" || rows[1]["scope"] != "shared" || rows[2]["scope"] != "team" {
		t.Fatalf("scopes = %v, %v, %v", rows[0]["scope"], rows[1]["scope"], rows[2]["scope"])
	}
}

func TestNormalizeConnectionListOutputFiltersGroups(t *testing.T) {
	input := []byte(`{
		"own":[{"id":"own-1","name":"Alpha","type":"SSH","host":"ssh.example","port":22}],
		"shared":[{"id":"shared-1","name":"Beta","type":"RDP","host":"rdp.example","port":3389}],
		"team":[]
	}`)

	displayBody, tableBody, err := normalizeConnectionListOutput(input, "beta")
	if err != nil {
		t.Fatalf("normalizeConnectionListOutput() error = %v", err)
	}

	var groups connectionListGroups
	if err := json.Unmarshal(displayBody, &groups); err != nil {
		t.Fatalf("unmarshal grouped display body: %v", err)
	}
	if len(groups.Own) != 0 || len(groups.Shared) != 1 || len(groups.Team) != 0 {
		t.Fatalf("filtered group counts = own:%d shared:%d team:%d", len(groups.Own), len(groups.Shared), len(groups.Team))
	}

	var rows []map[string]any
	if err := json.Unmarshal(tableBody, &rows); err != nil {
		t.Fatalf("unmarshal table body: %v", err)
	}
	if len(rows) != 1 || rows[0]["id"] != "shared-1" {
		t.Fatalf("filtered rows = %+v", rows)
	}
}

func TestNormalizeConnectionPermissionAcceptsLegacyAliases(t *testing.T) {
	tests := map[string]string{
		"read":        "READ_ONLY",
		"read-only":   "READ_ONLY",
		"READ_ONLY":   "READ_ONLY",
		"write":       "FULL_ACCESS",
		"full":        "FULL_ACCESS",
		"FULL_ACCESS": "FULL_ACCESS",
	}
	for input, want := range tests {
		got, err := normalizeConnectionPermission(input)
		if err != nil {
			t.Fatalf("normalizeConnectionPermission(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeConnectionPermission(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildConnectionExportPayloadUsesCurrentAPIFields(t *testing.T) {
	t.Cleanup(func() {
		connExportFormat = ""
		connExportIDs = nil
		connExportIncludeCredentials = false
		connExportFolderID = ""
	})
	connExportFormat = "csv"
	connExportIDs = []string{"conn-1", "conn-2"}
	connExportIncludeCredentials = true
	connExportFolderID = "folder-1"

	payload, err := buildConnectionExportPayload()
	if err != nil {
		t.Fatalf("buildConnectionExportPayload() error = %v", err)
	}
	if payload["format"] != "CSV" {
		t.Fatalf("format = %v, want CSV", payload["format"])
	}
	if payload["includeCredentials"] != true {
		t.Fatalf("includeCredentials = %v, want true", payload["includeCredentials"])
	}
	if payload["folderId"] != "folder-1" {
		t.Fatalf("folderId = %v, want folder-1", payload["folderId"])
	}
	ids, ok := payload["connectionIds"].([]string)
	if !ok || len(ids) != 2 || ids[0] != "conn-1" || ids[1] != "conn-2" {
		t.Fatalf("connectionIds = %#v", payload["connectionIds"])
	}
	if _, exists := payload["ids"]; exists {
		t.Fatal("legacy ids field should not be present")
	}
}
