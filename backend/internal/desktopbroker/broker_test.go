package desktopbroker

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type recordingStore struct {
	mu                sync.Mutex
	finalized         int
	tokenHash         string
	recordingID       string
	readyTokenHash    string
	readyConnectionID string
}

func (s *recordingStore) FinalizeDesktopSession(_ context.Context, tokenHash, recordingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finalized++
	s.tokenHash = tokenHash
	s.recordingID = recordingID
	return nil
}

func (s *recordingStore) GetDesktopSessionState(context.Context, string) (DesktopSessionState, error) {
	return DesktopSessionState{}, nil
}

func (s *recordingStore) GetDesktopSessionStateBySessionID(context.Context, string) (DesktopSessionState, error) {
	return DesktopSessionState{}, nil
}

func (s *recordingStore) RecordDesktopConnectionReady(_ context.Context, tokenHash, connectionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readyTokenHash = tokenHash
	s.readyConnectionID = connectionID
	return nil
}

func TestBrokerAcceptsNodeCompatibleTokenAndFlushesBufferedClientMessages(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake guacd: %v", err)
	}
	defer listener.Close()

	selectSeen := make(chan []string, 1)
	connectSeen := make(chan []string, 1)
	bufferedSeen := make(chan []string, 1)

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()

		decoder := &Decoder{}
		var pending [][]string
		readInstruction := func() []string {
			if len(pending) > 0 {
				instruction := pending[0]
				pending = pending[1:]
				return instruction
			}
			buffer := make([]byte, 4096)
			for {
				n, readErr := conn.Read(buffer)
				if readErr != nil {
					return nil
				}
				instructions, feedErr := decoder.Feed(buffer[:n])
				if feedErr != nil {
					return nil
				}
				if len(instructions) > 0 {
					pending = append(pending, instructions[1:]...)
					return instructions[0]
				}
			}
		}

		if instruction := readInstruction(); instruction != nil {
			selectSeen <- instruction
		}
		_, _ = conn.Write([]byte(EncodeInstruction("args", "VERSION_1_1_0", "hostname", "port", "username", "password")))

		for {
			instruction := readInstruction()
			if instruction == nil {
				return
			}
			if instruction[0] == "connect" {
				connectSeen <- instruction
				break
			}
		}

		_, _ = conn.Write([]byte(EncodeInstruction("ready", "abc123")))
		for {
			instruction := readInstruction()
			if instruction == nil {
				return
			}
			if instruction[0] == "sync" {
				bufferedSeen <- instruction
				return
			}
		}
	}()

	store := &recordingStore{}
	token := ConnectionToken{}
	token.Connection.Type = "rdp"
	token.Connection.GuacdHost = strings.Split(listener.Addr().String(), ":")[0]
	token.Connection.GuacdPort = listener.Addr().(*net.TCPAddr).Port
	token.Connection.Settings = map[string]any{
		"hostname": "10.0.0.25",
		"port":     "3389",
		"username": "alice",
		"password": "secret",
	}
	token.Metadata = map[string]any{"recordingId": "rec-1"}

	broker := NewBroker(BrokerConfig{
		GuacamoleSecret: "broker-secret",
		SessionStore:    store,
	})

	server := httptest.NewServer(http.HandlerFunc(broker.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/?token=" + mustEncryptToken(t, "broker-secret", token)
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial broker websocket: %v", err)
	}
	defer wsConn.Close()

	if err := wsConn.WriteMessage(websocket.TextMessage, []byte(EncodeInstruction("sync", "0"))); err != nil {
		t.Fatalf("write buffered client instruction: %v", err)
	}

	_, payload, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("read ready message: %v", err)
	}
	if string(payload) != EncodeInstruction("ready", "abc123") {
		t.Fatalf("unexpected ready payload: %q", string(payload))
	}

	select {
	case instruction := <-selectSeen:
		if len(instruction) != 2 || instruction[0] != "select" || instruction[1] != "rdp" {
			t.Fatalf("unexpected select instruction: %#v", instruction)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for select instruction")
	}

	select {
	case instruction := <-connectSeen:
		if instruction[0] != "connect" || instruction[2] != "10.0.0.25" {
			t.Fatalf("unexpected connect instruction: %#v", instruction)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for connect instruction")
	}

	select {
	case instruction := <-bufferedSeen:
		if instruction[0] != "sync" || instruction[1] != "0" {
			t.Fatalf("unexpected buffered instruction: %#v", instruction)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for buffered instruction")
	}

	store.mu.Lock()
	readyConnectionID := store.readyConnectionID
	readyTokenHash := store.readyTokenHash
	store.mu.Unlock()
	if readyConnectionID != "abc123" {
		t.Fatalf("ready connection id = %q, want %q", readyConnectionID, "abc123")
	}
	if readyTokenHash == "" {
		t.Fatal("expected broker to persist ready token hash")
	}
}

func TestConnectGuacdUsesTokenCA(t *testing.T) {
	t.Parallel()

	caPEM, serverCert := testGuacdCertificate(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{serverCert}})
	if err != nil {
		t.Fatalf("listen tls guacd: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		if tlsConn, ok := conn.(*tls.Conn); ok {
			_ = tlsConn.Handshake()
		}
		close(accepted)
	}()

	broker := NewBroker(BrokerConfig{GuacdTLS: true})
	conn, err := broker.connectGuacd(strings.Split(listener.Addr().String(), ":")[0], listener.Addr().(*net.TCPAddr).Port, string(caPEM))
	if err != nil {
		t.Fatalf("connectGuacd returned error: %v", err)
	}
	_ = conn.Close()

	select {
	case <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for guacd tls accept")
	}
}

func TestBrokerRejectsExpiredObserverToken(t *testing.T) {
	t.Parallel()

	store := &recordingStore{}
	token := sampleConnectionToken()
	token.Connection.Join = "owner-connection-123"
	token.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	token.Metadata = map[string]any{MetadataKeyObserveSessionID: "sess-observe"}

	broker := NewBroker(BrokerConfig{
		GuacamoleSecret: "broker-secret",
		SessionStore:    store,
	})

	server := httptest.NewServer(http.HandlerFunc(broker.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/?token=" + mustEncryptToken(t, "broker-secret", token)
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial broker websocket: %v", err)
	}
	defer wsConn.Close()

	_, payload, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("read invalid-token message: %v", err)
	}
	if string(payload) != EncodeInstruction("error", "Token validation failed", "INVALID_TOKEN") {
		t.Fatalf("unexpected expired-token payload: %q", string(payload))
	}
}

func testGuacdCertificate(t *testing.T) ([]byte, tls.Certificate) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	now := time.Now().UTC()
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caKey.Public(), caKey)
	if err != nil {
		t.Fatalf("create ca cert: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "arsenale-guacd"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, serverKey.Public(), caKey)
	if err != nil {
		t.Fatalf("create server cert: %v", err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyDER})
	cert, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("parse server key pair: %v", err)
	}
	return caPEM, cert
}
