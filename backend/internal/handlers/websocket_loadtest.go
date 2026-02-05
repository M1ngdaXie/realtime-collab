package handlers

import (
	"log"
	"net/http"

	"github.com/M1ngdaXie/realtime-collab/internal/hub"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// loadTestUpgrader allows any origin for load testing
var loadTestUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for load testing
	},
}

// HandleWebSocketLoadTest is a backdoor WebSocket handler that skips all authentication.
// WARNING: Only use this for local load testing! Do not expose in production.
//
// Usage: ws://localhost:8080/ws/loadtest/:roomId
// Optional query param: ?user=customUserName
func (wsh *WebSocketHandler) HandleWebSocketLoadTest(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomId is required"})
		return
	}

	// Generate or use provided user identifier
	userName := c.Query("user")
	if userName == "" {
		userName = "LoadTestUser-" + uuid.New().String()[:8]
	}

	// Upgrade connection - NO AUTH CHECK
	conn, err := loadTestUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client with full edit permissions (no userID, anonymous)
	clientID := uuid.New().String()
	client := hub.NewClient(
		clientID,
		nil,      // userID - nil for anonymous
		userName, // userName
		nil,      // userAvatar
		"edit",   // permission - full access for load testing
		conn,
		wsh.hub,
		roomID,
	)

	// Register client
	wsh.hub.Register <- client

	// Start goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Printf("[LOADTEST] Client connected: %s (user: %s) to room: %s", clientID, userName, roomID)
}
