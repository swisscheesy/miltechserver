package controller

import (
	"log/slog"
	"miltechserver/api/websocket"
	"miltechserver/bootstrap"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

// WebSocket upgrader configuration
var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, this will validate against allowed origins
		// For now, allow all origins for development
		return true
	},
}

// ConfigureWebSocketUpgrader configures the WebSocket upgrader with allowed origins
func ConfigureWebSocketUpgrader(allowedOrigins string) {
	if allowedOrigins == "" {
		// Development mode - allow all origins
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		return
	}

	// Production mode - validate against whitelist
	origins := strings.Split(allowedOrigins, ",")
	originMap := make(map[string]bool)
	for _, origin := range origins {
		originMap[strings.TrimSpace(origin)] = true
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return originMap[origin]
	}
}

// StreamShopMessages handles WebSocket connections for real-time shop messages
func (controller *ShopsController) StreamShopMessages(c *gin.Context) {
	// 1. Get user from context (authenticated by middleware)
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok || user == nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized WebSocket connection attempt")
		return
	}

	// 2. Get shop_id from URL parameter
	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	// 3. Verify shop membership
	isMember, err := controller.ShopsService.IsUserShopMember(user, shopID)
	if err != nil {
		slog.Error("Failed to check shop membership",
			"error", err,
			"shop_id", shopID,
			"user_id", user.UserID,
		)
		c.JSON(500, gin.H{"message": "internal server error"})
		return
	}

	if !isMember {
		c.JSON(403, gin.H{"message": "not a shop member"})
		slog.Info("WebSocket connection denied - not a shop member",
			"shop_id", shopID,
			"user_id", user.UserID,
		)
		return
	}

	// 4. Get the Hub from the service
	hub := controller.ShopsService.GetHub()
	if hub == nil {
		c.JSON(503, gin.H{"message": "WebSocket service unavailable"})
		slog.Error("WebSocket Hub not initialized")
		return
	}

	// 5. Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed",
			"error", err,
			"shop_id", shopID,
			"user_id", user.UserID,
		)
		return
	}

	// 6. Create client and register with hub
	client := websocket.NewClient(hub, conn, shopID, user.UserID, user.Username)
	hub.Register(client)

	// 7. Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	slog.Info("WebSocket client connected successfully",
		"shop_id", shopID,
		"user_id", user.UserID,
		"username", user.Username,
	)
}
