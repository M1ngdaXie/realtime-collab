package hub

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	RoomID  string
	Data	[]byte
	sender  *Client
}

type Client struct {
	ID             string
	UserID         *uuid.UUID // Authenticated user ID (nil for public share access)
	UserName       string     // User's display name for presence
	UserAvatar     *string    // User's avatar URL for presence
	Conn           *websocket.Conn
	send           chan []byte
	sendMu         sync.Mutex
	sendClosed     bool
	hub            *Hub
	roomID         string
	mutex          sync.Mutex
	unregisterOnce sync.Once
	failureCount   int
	failureMu      sync.Mutex
}
type Room struct {
	ID 	string
	clients map[*Client]bool
	mu sync.RWMutex
}

type Hub struct {
	rooms 	map[string]*Room
	mu		sync.RWMutex
	Register   chan *Client // Exported
  	Unregister chan *Client // Exported
  	Broadcast  chan *Message // Exported
}
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.Unregister:
			h.unregisterClient(client)
		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.roomID]
	if !exists {
		room = &Room{
			ID:      client.roomID,
			clients: make(map[*Client]bool),
		}
		h.rooms[client.roomID] = room
		log.Printf("Created new room with ID: %s", client.roomID)
	}
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()
	log.Printf("Client %s joined room %s (total clients: %d)", client.ID, client.roomID, len(room.clients))
}
func (h *Hub) unregisterClient(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    room, exists := h.rooms[client.roomID]
    if !exists {
        log.Printf("Room %s does not exist for client %s", client.roomID, client.ID)
        return
    }

    room.mu.Lock()
    defer room.mu.Unlock()

    if _, ok := room.clients[client]; ok {
        delete(room.clients, client)

        // Safely close send channel exactly once
        client.sendMu.Lock()
        if !client.sendClosed {
            close(client.send)
            client.sendClosed = true
        }
        client.sendMu.Unlock()

        log.Printf("Client %s disconnected from room %s (total clients: %d)",
            client.ID, client.roomID, len(room.clients))
    }

    if len(room.clients) == 0 {
        delete(h.rooms, client.roomID)
        log.Printf("Deleted empty room with ID: %s", client.roomID)
    }
}

const (
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10  // 54 seconds
	maxSendFailures = 5
)

func (h *Hub) broadcastMessage(message *Message) {
    h.mu.RLock()
    room, exists := h.rooms[message.RoomID]
    h.mu.RUnlock()
    if !exists {
        log.Printf("Room %s does not exist for broadcasting", message.RoomID)
        return
    }

    room.mu.RLock()
    defer room.mu.RUnlock()

    for client := range room.clients {
        if client != message.sender {
            select {
            case client.send <- message.Data:
                // Success - reset failure count
                client.failureMu.Lock()
                client.failureCount = 0
                client.failureMu.Unlock()

            default:
                // Failed - increment failure count
                client.failureMu.Lock()
                client.failureCount++
                currentFailures := client.failureCount
                client.failureMu.Unlock()

                log.Printf("Failed to send to client %s (channel full, failures: %d/%d)",
                    client.ID, currentFailures, maxSendFailures)

                // Disconnect if threshold exceeded
                if currentFailures >= maxSendFailures {
                    log.Printf("Client %s exceeded max send failures, disconnecting", client.ID)
                    go func(c *Client) {
                        c.unregister()
                        c.Conn.Close()
                    }(client)
                }
            }
        }
    }
}


func (c *Client) ReadPump() {
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	defer func() {
		c.unregister()
		c.Conn.Close()
	}()
	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from client %s: %v", c.ID, err)
			break
		}
		if messageType == websocket.BinaryMessage {
			c.hub.Broadcast <- &Message{
				RoomID: c.roomID,
				Data:   message,
				sender: c,
			}
		}
	}
}

func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.unregister()  // NEW: Now WritePump also unregisters
        c.Conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // Hub closed the channel
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            err := c.Conn.WriteMessage(websocket.BinaryMessage, message)
            if err != nil {
                log.Printf("Error writing message to client %s: %v", c.ID, err)
                return
            }

        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                log.Printf("Ping failed for client %s: %v", c.ID, err)
                return
            }
        }
    }
}


func NewClient(id string, userID *uuid.UUID, userName string, userAvatar *string, conn *websocket.Conn, hub *Hub, roomID string) *Client {
	return &Client{
		ID:         id,
		UserID:     userID,
		UserName:   userName,
		UserAvatar: userAvatar,
		Conn:       conn,
		send:       make(chan []byte, 256),
		hub:        hub,
		roomID:     roomID,
	}
}
func (c *Client) unregister() {
    c.unregisterOnce.Do(func() {
        c.hub.Unregister <- c
    })
}