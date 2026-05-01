package terminalbroker

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/sessionrecording"
	"github.com/gorilla/websocket"
)

func recordingReference(metadata map[string]string) *sessionrecording.Reference {
	ref, ok := sessionrecording.ReferenceFromMetadataStrings(metadata)
	if !ok {
		return nil
	}
	return &ref
}

func sendSocketErrorAndClose(conn *websocket.Conn, code, message string) {
	_ = sendWebsocketMessage(conn, serverMessage{Type: "error", Code: code, Message: message})
	_ = conn.Close()
}

func sendWebsocketMessage(conn *websocket.Conn, message serverMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func closeWebSocketConnection(conn *websocket.Conn, code int, text string) {
	if conn == nil {
		return
	}
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, text),
		time.Now().Add(time.Second),
	)
	_ = conn.Close()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
