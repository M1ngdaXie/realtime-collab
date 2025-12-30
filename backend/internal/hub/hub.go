package hub

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	RoomID  string
	Data	[]byte
	sender  *Client
}

type Client struct {
	ID    string
	Conn  *websocket.Conn
	send chan []byte
	hub    *Hub
	roomID string
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
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
		close(client.send)
		log.Printf("Client %s disconnected from room %s", client.ID, client.roomID)
	}

	room.mu.Unlock()
	log.Printf("Client %s left room %s (total clients: %d)", client.ID, client.roomID, len(room.clients))

	if len(room.clients) == 0 {
		delete(h.rooms, client.roomID)
		log.Printf("Deleted empty room with ID: %s", client.roomID)
	}
}
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
			default:
				log.Printf("Failed to send to client %s (channel full)", client.ID)
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
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
	defer func() {
		c.Conn.Close()
	}()
	for message := range c.send {
		err := c.Conn.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Printf("Error writing message to client %s: %v", c.ID, err)
			break
		}
	}
}

func NewClient(id string, conn *websocket.Conn, hub *Hub, roomID string) *Client {
	return &Client{
		ID:     id,
		Conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		roomID: roomID,
	}
}