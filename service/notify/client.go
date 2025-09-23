package notify

import "github.com/gorilla/websocket"

type Client struct {
	ClientID uint
	conn     *websocket.Conn
}

func NewClient(clientID uint, conn *websocket.Conn) *Client {
	return &Client{
		ClientID: clientID,
		conn:     conn,
	}
}

func (client *Client) Notify(message any) error {
	return client.conn.WriteJSON(message)
}
