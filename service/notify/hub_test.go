package notify

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var (
	upgrader = websocket.Upgrader{}
	hub      *Hub
	logger   = slog.New(slog.NewTextHandler(os.Stdout, nil))
)

func TestMain(m *testing.M) {
	hub = NewHub(logger)
	os.Exit(m.Run())
}

// Just a simple websocket handler that will throw the message its receive back
func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		var msg any
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		// Just echo it back
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

// Simulate a websocket connection
func newTestConn(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	u := "ws" + server.URL[len("http"):] // convert http://127.0.0.1 → ws://127.0.0.1
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	return conn
}

// Test broadcast logic
func TestHubBroadcast(t *testing.T) {
	// Create the test server
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()

	// Create client connections
	conn1 := newTestConn(t, server)
	defer conn1.Close()
	conn2 := newTestConn(t, server)
	defer conn2.Close()

	// Create clients
	client1 := NewClient(1, conn1)
	client2 := NewClient(2, conn2)

	// Subscribe clients to the hub
	hub.Subscribe(client1)
	hub.Subscribe(client2)

	// Broadcast the message
	msg := map[string]string{"msg": "hi everyone"}
	success := hub.Broadcast(msg)
	require.Equal(t, 2, success)

	// Verify both clients received the message
	var receive1, receive2 map[string]string

	err := conn1.ReadJSON(&receive1)
	require.NoError(t, err)
	require.Equal(t, msg["msg"], receive1["msg"])

	err = conn2.ReadJSON(&receive2)
	require.NoError(t, err)
	require.Equal(t, msg["msg"], receive2["msg"])

	// Unsubscribe clients
	hub.Unsubscribe(client1.ClientID, client1)
	hub.Unsubscribe(client2.ClientID, client2)
	require.Equal(t, 0, len(hub.clients))
}
