package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/config"
	"github.com/M1ngdaXie/realtime-collab/internal/hub"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// connectionSem limits concurrent WebSocket connection handshakes
// to prevent overwhelming the database during connection storms
var connectionSem = make(chan struct{}, 200)

type WebSocketHandler struct {
	hub   *hub.Hub
	store store.Store
	cfg   *config.Config
}

func NewWebSocketHandler(h *hub.Hub, s store.Store, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub:   h,
		store: s,
		cfg:   cfg,
	}
}

func (wsh *WebSocketHandler) getUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, allowed := range wsh.cfg.AllowedOrigins {
				if allowed == origin {
					return true
				}
			}
			return false
		},
	}
}

func (wsh *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Acquire semaphore to limit concurrent connection handshakes
	select {
	case connectionSem <- struct{}{}:
		defer func() { <-connectionSem }()
	case <-time.After(10 * time.Second):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server busy, retry later"})
		return
	}

	roomID := c.Param("roomId")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomId is required"})
		return
	}

	// Parse document ID
	documentID, err := uuid.Parse(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Try to authenticate via JWT token or share token
	var userID *uuid.UUID
	var userName string
	var userAvatar *string
	authenticated := false

	// Check for JWT token in query parameter
	jwtToken := c.Query("token")
	if jwtToken != "" {
		// Direct JWT validation - fast path (~1ms)
		claims, err := auth.ValidateJWT(jwtToken, wsh.cfg.JWTSecret)
		if err == nil {
			// Extract user data from JWT claims
			uid, parseErr := uuid.Parse(claims.Subject)
			if parseErr == nil {
				userID = &uid
				userName = claims.Name
				userAvatar = claims.AvatarURL
				authenticated = true
			}
		}
	}

	// If not authenticated via JWT, check for share token
	if !authenticated {
		shareToken := c.Query("share")
		if shareToken != "" {
			// Validate share token
			valid, err := wsh.store.ValidateShareToken(c.Request.Context(), documentID, shareToken)
			if err != nil {
				log.Printf("Error validating share token: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate share token"})
				return
			}
			if !valid {
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid or expired share token"})
				return
			}
			// Share token is valid, allow connection with anonymous user
			userName = "Anonymous"
			authenticated = true
		}
	}

	// If still not authenticated, reject connection
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required. Provide 'token' or 'share' query parameter"})
		return
	}

	// Determine permission level
	var permission string
	if userID != nil {
		// Authenticated user - get their permission level
		perm, err := wsh.store.GetUserPermission(c.Request.Context(), documentID, *userID)
		if err != nil {
			log.Printf("Error getting user permission: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			return
		}
		if perm == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this document"})
			return
		}
		permission = perm
	} else {
		// Share token user - get share link permission
		perm, err := wsh.store.GetShareLinkPermission(c.Request.Context(), documentID)
		if err != nil {
			log.Printf("Error getting share link permission: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			return
		}
		if perm == "" {
			// Share link doesn't exist or document isn't public
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid share link"})
			return
		}
		permission = perm
	}

	// Upgrade connection
	upgrader := wsh.getUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client with user information and permission
	clientID := uuid.New().String()
	client := hub.NewClient(clientID, userID, userName, userAvatar, permission, conn, wsh.hub, roomID)

	// Register client
	wsh.hub.Register <- client

	// Start goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Printf("Client connected: %s (user: %s) to room: %s", clientID, userName, roomID)
}
