package handlers

import (
	"log"
	"net/http"

	"github.com/M1ngdaXie/realtime-collab/internal/hub"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)


var upgrader = websocket.Upgrader{
  	ReadBufferSize:  1024,
  	WriteBufferSize: 1024,
  	CheckOrigin: func(r *http.Request) bool {
  		// Allow all origins for development
  		// TODO: Restrict in production
  		return true
  	},
}

 type WebSocketHandler struct {
  	hub *hub.Hub
  }

  func NewWebSocketHandler(h *hub.Hub) *WebSocketHandler {
  	return &WebSocketHandler{hub: h}
  }

  func (wsh *WebSocketHandler) HandleWebSocket(c *gin.Context){
	roomID := c.Param("roomId")

	if(roomID == ""){
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomId is required"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
    
		// Create a new client
  	clientID := uuid.New().String()
  	client := hub.NewClient(clientID, conn, wsh.hub, roomID)

  	// Register client with hub
  	wsh.hub.Register <- client

  	// Start read and write pumps in separate goroutines
  	go client.WritePump()
  	go client.ReadPump()

  	log.Printf("WebSocket connection established for client %s in room %s", clientID, roomID)
  }

  
