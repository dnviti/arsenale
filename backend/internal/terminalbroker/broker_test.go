package terminalbroker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestCloseWebSocketConnectionSendsCloseFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		closeWebSocketConnection(conn, websocket.CloseNormalClosure, "")
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer client.Close()

	if _, _, err := client.ReadMessage(); err == nil {
		t.Fatal("expected close error from websocket read")
	} else if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Fatalf("expected normal close error, got %v", err)
	}
}
