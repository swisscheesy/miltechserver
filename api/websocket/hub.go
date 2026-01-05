package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Hub maintains active WebSocket connections for all shops
type Hub struct {
	// Map: shop_id → set of clients
	shops map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to shop members
	broadcast chan *BroadcastMessage

	// Mutex for thread-safe map access
	mu sync.RWMutex

	// Shutdown channel
	shutdown chan struct{}
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		shops:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		shutdown:   make(chan struct{}),
	}
}

// Run starts the hub's main loop (should be run in a goroutine)
func (h *Hub) Run() {
	slog.Info("WebSocket Hub started")
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.broadcastMessage(msg)

		case <-h.shutdown:
			slog.Info("WebSocket Hub shutting down")
			h.closeAllConnections()
			return
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.shops[client.shopID]; !ok {
		h.shops[client.shopID] = make(map[*Client]bool)
	}
	h.shops[client.shopID][client] = true

	slog.Info("WebSocket client registered",
		"shop_id", client.shopID,
		"user_id", client.userID,
		"username", client.username,
		"shop_client_count", len(h.shops[client.shopID]),
	)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.shops[client.shopID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			// Remove shop entry if no more clients
			if len(clients) == 0 {
				delete(h.shops, client.shopID)
			}

			slog.Info("WebSocket client unregistered",
				"shop_id", client.shopID,
				"user_id", client.userID,
				"username", client.username,
				"remaining_clients", len(clients),
			)
		}
	}
}

// broadcastMessage sends a message to all clients in a shop
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	h.mu.RLock()
	clients := h.shops[msg.ShopID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		slog.Debug("No connected clients for broadcast",
			"shop_id", msg.ShopID,
			"message_id", msg.Message.ID,
		)
		return
	}

	messageJSON, err := json.Marshal(msg.Message)
	if err != nil {
		slog.Error("Failed to marshal WebSocket message",
			"error", err,
			"shop_id", msg.ShopID,
			"message_id", msg.Message.ID,
		)
		return
	}

	slog.Debug("Broadcasting message to clients",
		"shop_id", msg.ShopID,
		"message_id", msg.Message.ID,
		"client_count", len(clients),
	)

	// Broadcast to all clients in the shop
	for client := range clients {
		select {
		case client.send <- messageJSON:
			// Message sent successfully
		default:
			// Client's send buffer is full, disconnect them
			slog.Warn("Client send buffer full, disconnecting",
				"shop_id", client.shopID,
				"user_id", client.userID,
			)
			h.mu.Lock()
			delete(clients, client)
			close(client.send)
			h.mu.Unlock()
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// BroadcastMessage sends a shop message to all connected clients in that shop
func (h *Hub) BroadcastMessage(shopID string, message *ShopMessageWithUsername) {
	select {
	case h.broadcast <- &BroadcastMessage{
		ShopID:  shopID,
		Message: message,
	}:
		// Message queued successfully
	default:
		slog.Error("Broadcast channel full, message dropped",
			"shop_id", shopID,
			"message_id", message.ID,
		)
	}
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	close(h.shutdown)
}

// closeAllConnections closes all active WebSocket connections
func (h *Hub) closeAllConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for shopID, clients := range h.shops {
		for client := range clients {
			close(client.send)
		}
		delete(h.shops, shopID)
	}
	slog.Info("All WebSocket connections closed")
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalClients := 0
	for _, clients := range h.shops {
		totalClients += len(clients)
	}

	return map[string]interface{}{
		"active_connections": totalClients,
		"active_shops":       len(h.shops),
	}
}
