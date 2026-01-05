package websocket

import (
	"log/slog"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *gorillaws.Conn, shopID, userID, username string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, sendBufferSize),
		shopID:   shopID,
		userID:   userID,
		username: username,
	}
}

// ReadPump handles incoming messages from the WebSocket connection
// This goroutine runs until the connection is closed
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Read loop - we discard client messages since this is a read-only stream
	// Clients send messages via POST /shops/messages instead
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
				slog.Warn("WebSocket connection closed unexpectedly",
					"error", err,
					"shop_id", c.shopID,
					"user_id", c.userID,
				)
			}
			break
		}
	}
}

// WritePump sends messages to the WebSocket connection
// This goroutine runs until the connection is closed
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(gorillaws.TextMessage, message); err != nil {
				slog.Warn("Failed to write WebSocket message",
					"error", err,
					"shop_id", c.shopID,
					"user_id", c.userID,
				)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				slog.Debug("Failed to send ping message",
					"error", err,
					"shop_id", c.shopID,
					"user_id", c.userID,
				)
				return
			}
		}
	}
}
