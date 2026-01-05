package websocket

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"

	gorillaws "github.com/gorilla/websocket"
)

// WebSocket timing constants
const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10 // 54 seconds

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB

	// Send buffer size (number of messages)
	sendBufferSize = 256
)

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *gorillaws.Conn
	send     chan []byte
	shopID   string
	userID   string
	username string
}

// BroadcastMessage represents a message to be broadcast to shop members
type BroadcastMessage struct {
	ShopID  string
	Message *ShopMessageWithUsername
}

// ShopMessageWithUsername extends the base ShopMessages model with user context
type ShopMessageWithUsername struct {
	*model.ShopMessages
	Username string `json:"username"`
}
