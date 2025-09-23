package notify

import (
	"fmt"
	"log/slog"
	"sync"
)

type Hub struct {
	clients map[uint]*Client
	logger  *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[uint]*Client),
		logger:  logger,
	}
}

func (hub *Hub) Subscribe(client *Client) {
	// Add the client to the map using client ID
	hub.clients[client.ClientID] = client
}

func (hub *Hub) Unsubscribe(clientID uint, client *Client) {
	// Remove the client out of the map
	delete(hub.clients, clientID)

	// Close the websocket connection to clean up resource
	client.conn.Close()
}

func (hub *Hub) Broadcast(message any) int {
	// We receive the message, and write that message (JSON) to all clients
	// using the existing websocket connection
	// Since we broadcast to a lot of different, we won't return even if some failed
	// Use goroutine to speed up processs

	var (
		wg      = sync.WaitGroup{}
		success = 0
		mutex   = sync.Mutex{}
	)

	for _, clt := range hub.clients {
		wg.Add(1)
		go func(client *Client) {
			defer wg.Done()

			// Send message
			if err := client.Notify(message); err != nil {
				hub.logger.Error("Error sending message to client", "id", client.ClientID, "error", err)
			}

			// Increase the number of success
			mutex.Lock()
			defer mutex.Unlock()
			success++
		}(clt)
	}

	wg.Wait()

	return success
}

func (hub *Hub) Publish(clientID uint, message any) error {
	client, ok := hub.clients[clientID]
	if !ok {
		return fmt.Errorf("client ID not registed in hub")
	}
	return client.Notify(message)
}

func (hub *Hub) IsUserOnline(clientID uint) bool {
	_, isOnline := hub.clients[clientID]
	return isOnline
}
