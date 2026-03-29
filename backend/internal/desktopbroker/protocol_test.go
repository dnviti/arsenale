package desktopbroker

import "testing"

func TestDecoderFeed(t *testing.T) {
	t.Parallel()

	decoder := &Decoder{}
	first, err := decoder.Feed([]byte("4.args,13.VERSION_1_1_0,8.hostname"))
	if err != nil {
		t.Fatalf("feed first chunk: %v", err)
	}
	if len(first) != 0 {
		t.Fatalf("expected no complete instructions, got %d", len(first))
	}

	second, err := decoder.Feed([]byte(";"))
	if err != nil {
		t.Fatalf("feed second chunk: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("expected one instruction, got %d", len(second))
	}
	if second[0][0] != "args" || second[0][2] != "hostname" {
		t.Fatalf("unexpected instruction: %#v", second[0])
	}
}

func TestBuildHandshakeMessages(t *testing.T) {
	t.Parallel()

	settings := CompiledSettings{
		Selector: "rdp",
		Values: map[string]string{
			"hostname": "10.0.0.10",
			"port":     "3389",
			"username": "alice",
			"password": "secret",
		},
		Width:  "1280",
		Height: "720",
		DPI:    "96",
		Audio:  []string{"audio/L16"},
		Image:  []string{"image/png", "image/jpeg"},
	}

	messages, err := BuildHandshakeMessages(settings, []string{"VERSION_1_1_0", "hostname", "port", "username", "password"})
	if err != nil {
		t.Fatalf("build handshake: %v", err)
	}
	if len(messages) != 5 {
		t.Fatalf("expected 5 handshake messages, got %d", len(messages))
	}
	if messages[len(messages)-1] != EncodeInstruction("connect", "VERSION_1_1_0", "10.0.0.10", "3389", "alice", "secret") {
		t.Fatalf("unexpected connect message: %q", messages[len(messages)-1])
	}
}
