package importexportapi

import "testing"

func TestParseColumnMapping(t *testing.T) {
	t.Parallel()

	mapping, err := parseColumnMapping(`{"Name":"Display Name","HOST":"Address"}`)
	if err != nil {
		t.Fatalf("parseColumnMapping() error = %v", err)
	}
	if got := mapping.resolve("name", "name"); got != "display name" {
		t.Fatalf("name mapping = %q, want %q", got, "display name")
	}
	if got := mapping.resolve("host", "host"); got != "address" {
		t.Fatalf("host mapping = %q, want %q", got, "address")
	}
	if got := mapping.resolve("type", "type"); got != "type" {
		t.Fatalf("fallback mapping = %q, want %q", got, "type")
	}
}

func TestParseRDPFile(t *testing.T) {
	t.Parallel()

	parsed := parseRDPFile("full address:s:rdp.example.internal:3390\nusername:s:alice\n")
	if parsed.Hostname != "rdp.example.internal" {
		t.Fatalf("hostname = %q, want %q", parsed.Hostname, "rdp.example.internal")
	}
	if parsed.Port != 3390 {
		t.Fatalf("port = %d, want %d", parsed.Port, 3390)
	}
	if parsed.Username != "alice" {
		t.Fatalf("username = %q, want %q", parsed.Username, "alice")
	}
}

func TestParseMRemoteNGXML(t *testing.T) {
	t.Parallel()

	xml := `<Connections>
  <Connection Name="Root SSH" Hostname="ssh.example.internal" Protocol="SSH" Port="22" Username="root" />
  <Connection Name="Folder">
    <Connection Name="Nested RDP" Hostname="rdp.example.internal" Protocol="RDP" Panel="Windows" />
  </Connection>
</Connections>`

	parsed, err := parseMRemoteNGXML(xml)
	if err != nil {
		t.Fatalf("parseMRemoteNGXML() error = %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("len(parsed) = %d, want %d", len(parsed), 2)
	}
	if parsed[0].Protocol != "SSH" || parsed[0].Hostname != "ssh.example.internal" {
		t.Fatalf("first parsed connection = %+v", parsed[0])
	}
	if parsed[1].Protocol != "RDP" || parsed[1].Panel != "Windows" || parsed[1].Port != "3389" {
		t.Fatalf("second parsed connection = %+v", parsed[1])
	}
}
