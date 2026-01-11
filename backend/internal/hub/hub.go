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
	Permission     string     // User's permission level: "owner", "edit", "view"
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
	observedYjsIDs map[uint64]uint64 // clientID -> maxClock
	idsMu          sync.Mutex
}
type Room struct {
	ID            string
	clients       map[*Client]bool
	mu            sync.RWMutex
	lastAwareness []byte // 存储最新的 awareness 消息，用于新用户加入时立即同步
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
		Broadcast:  make(chan *Message, 1024),
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
	// 获取现有的 awareness 数据（如果有的话）
	awarenessData := room.lastAwareness
	room.mu.Unlock()
	log.Printf("Client %s joined room %s (total clients: %d)", client.ID, client.roomID, len(room.clients))

	// 如果房间有之前的 awareness 状态，立即发送给新用户
	// 这样新用户不需要等待其他用户的下一次广播就能看到在线用户
	if len(awarenessData) > 0 {
		select {
		case client.send <- awarenessData:
			log.Printf("📤 Sent existing awareness to new client %s", client.ID)
		default:
			log.Printf("⚠️ Failed to send awareness to new client %s (channel full)", client.ID)
		}
	}
}
func (h *Hub) unregisterClient(client *Client) {
    h.mu.Lock()
    // ---------------------------------------------------
    // 注意：这里不要用 defer h.mu.Unlock()，因为我们要手动控制锁
    // ---------------------------------------------------

    room, exists := h.rooms[client.roomID]
    if !exists {
        h.mu.Unlock()
        return
    }

    room.mu.Lock() // 锁住房间
    if _, ok := room.clients[client]; ok {
        delete(room.clients, client)
        
        // 关闭连接通道
        client.sendMu.Lock()
        if !client.sendClosed {
            close(client.send)
            client.sendClosed = true
        }
        client.sendMu.Unlock()

        log.Printf("Client disconnected: %s", client.ID)
    }
    
    // 检查房间是否还有其他人
    remainingClientsCount := len(room.clients)
    room.mu.Unlock() // 解锁房间 (我们已经删完人了)

    // ---------------------------------------------------
    // [新增] 僵尸用户清理逻辑 (核心修改)
    // ---------------------------------------------------
    
    // 只有当房间里还有其他人时，才需要广播通知
    if remainingClientsCount > 0 {
        // 1. 从 client 的小本本里取出它用过的 Yjs ID 和对应的 clock
        client.idsMu.Lock()
        clientClocks := make(map[uint64]uint64, len(client.observedYjsIDs))
        for id, clock := range client.observedYjsIDs {
            clientClocks[id] = clock
        }
        client.idsMu.Unlock()

        // 2. 如果有记录到的 ID，就伪造删除消息 (使用 clock+1)
        if len(clientClocks) > 0 {
            deleteMsg := MakeYjsDeleteMessage(clientClocks) // 调用工具函数，传入 clientID -> clock map

            log.Printf("🧹 Notifying others to remove Yjs IDs with clocks: %v", clientClocks)
            
            // 3. 广播给房间里的幸存者
            // 构造一个消息对象
            msg := &Message{
                RoomID: client.roomID,
                Data:   deleteMsg,
                sender: nil, // sender 设为 nil，表示系统消息
            }
            
            // ！！特别注意！！
            // 不要在这里直接调用 h.broadcastMessage(msg)，因为那会尝试重新获取 h.mu 锁导致死锁
            // 我们直接把它扔到 Channel 里，让 Run() 去处理
            // 必须在一个非阻塞的 goroutine 里发，或者确保 channel 有缓冲
            
            go func() {
                // 使用 select 尝试发送，但如果满了，我们要稍微等一下，而不是直接丢弃
                // 因为这是“清理僵尸”的关键消息，丢了就会出 Bug
                select {
                case h.Broadcast <- msg:
                    // 发送成功
                case <-time.After(500 * time.Millisecond):
                    // 如果 500ms 还没塞进去，那说明系统真的挂了，只能丢弃并打印错误
                    log.Printf("❌ Critical: Failed to broadcast cleanup message (Channel blocked)")
                }
            }()
        }
    }

    // ---------------------------------------------------
    // 结束清理逻辑
    // ---------------------------------------------------

    if remainingClientsCount == 0 {
        delete(h.rooms, client.roomID)
        log.Printf("Room destroyed: %s", client.roomID)
    }

    h.mu.Unlock() // 最后解锁 Hub
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

    // 如果是 awareness 消息 (type=1)，保存它供新用户使用
    if len(message.Data) > 0 && message.Data[0] == 1 {
        room.mu.Lock()
        room.lastAwareness = make([]byte, len(message.Data))
        copy(room.lastAwareness, message.Data)
        room.mu.Unlock()
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
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break
        }
        if len(message) > 0 && message[0] == 1 {
    log.Printf("DEBUG: 收到 Awareness (光标) 消息 from User %s Permission: %s", c.ID, c.Permission)
}
        // ==========================================================
        // 1. 偷听逻辑 (Sniff) - 必须放在转发之前！
        // ==========================================================
        if messageType == websocket.BinaryMessage && len(message) > 0 && message[0] == 1 {
            clockMap := SniffYjsClientIDs(message)
            if len(clockMap) > 0 {
                c.idsMu.Lock()
                for id, clock := range clockMap {
                    if clock > c.observedYjsIDs[id] {
                        log.Printf("🕵️ [Sniff] Client %s uses YjsID: %d (clock: %d)", c.ID, id, clock)
                        c.observedYjsIDs[id] = clock
                    }
                }
                c.idsMu.Unlock()
            }
        }

        // ==========================================================
        // 2. 权限检查 - 只有编辑权限的用户才能广播消息
        // ==========================================================
        // ==========================================================
        // 2. 权限检查 - 精细化拦截 (Fine-grained Permission Check)
        // ==========================================================
        if c.Permission == "view" {
            // Yjs Protocol:
            // message[0]: MessageType (0=Sync, 1=Awareness)
            // message[1]: SyncMessageType (0=Step1, 1=Step2, 2=Update)
            
            // 只有当消息是 "Sync Update" (修改文档) 时，才拦截
            isSyncMessage := len(message) > 0 && message[0] == 0
            isUpdateOp := len(message) > 1 && message[1] == 2

            if isSyncMessage && isUpdateOp {
                log.Printf("🛡️ [Security] Blocked unauthorized WRITE from view-only user: %s", c.ID)
                continue // ❌ 拦截修改
            }
            
            // ✅ 放行：
            // 1. Awareness (光标): message[0] == 1
            // 2. SyncStep1/2 (握手加载文档): message[1] == 0 or 1
        }

        // ==========================================================
        // 3. 转发逻辑 (Broadcast) - 恢复协作功能
        // ==========================================================
        if messageType == websocket.BinaryMessage {
            // 注意：这里要检查 channel 是否已满，避免阻塞导致 ReadPump 卡死
            select {
            case c.hub.Broadcast <- &Message{
                RoomID: c.roomID,
                Data:   message,
                sender: c,
            }:
                // 发送成功
            default:
                log.Printf("⚠️ Hub broadcast channel is full, dropping message from %s", c.ID)
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


func NewClient(id string, userID *uuid.UUID, userName string, userAvatar *string, permission string, conn *websocket.Conn, hub *Hub, roomID string) *Client {
	return &Client{
		ID:             id,
		UserID:         userID,
		UserName:       userName,
		UserAvatar:     userAvatar,
		Permission:     permission,
		Conn:           conn,
		send:           make(chan []byte, 1024),
		hub:            hub,
		roomID:         roomID,
		observedYjsIDs: make(map[uint64]uint64),
	}
}
func (c *Client) unregister() {
    c.unregisterOnce.Do(func() {
        c.hub.Unregister <- c
    })
}