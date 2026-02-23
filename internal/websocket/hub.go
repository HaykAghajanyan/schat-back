package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/HaykAghajanyan/chat-backend/internal/models"
)

type Client struct {
	UserID int
	Conn   *Connection
	Send   chan []byte
}

type Hub struct {
	clients    map[int]*Client
	mu         sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *BroadcastMessage
}

type BroadcastMessage struct {
	UserID  int
	Message []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *BroadcastMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			log.Printf("User %d connected. Total connections: %d", client.UserID, len(h.clients))
			h.sendOnlineListLocked(client)
			h.broadcastPresenceLocked(client.UserID, models.WSMessageTypeOnline, client.UserID)
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
				log.Printf("User %d disconnected. Total connections: %d", client.UserID, len(h.clients))
				h.broadcastPresenceLocked(-1, models.WSMessageTypeOffline, client.UserID)
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			client, ok := h.clients[message.UserID]
			h.mu.RUnlock()

			if ok {
				select {
				case client.Send <- message.Message:
					// Message sent successfully
				default:
					// Client's send channel is full, close connection
					h.mu.Lock()
					close(client.Send)
					delete(h.clients, client.UserID)
					h.mu.Unlock()
				}
			}
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID int, message []byte) {
	h.Broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: message,
	}
}

// IsUserOnline checks if a user is currently connected
func (h *Hub) IsUserOnline(userID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUsers returns list of online user IDs
func (h *Hub) GetOnlineUsers() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]int, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// sendOnlineListLocked sends the current list of online users to the given client.
// Must be called with h.mu held.
func (h *Hub) sendOnlineListLocked(client *Client) {
	onlineUsers := make([]int, 0, len(h.clients))
	for id := range h.clients {
		onlineUsers = append(onlineUsers, id)
	}

	outgoing := models.WSOutgoingMessage{
		Type:        models.WSMessageTypePresence,
		OnlineUsers: onlineUsers,
		Timestamp:   time.Now(),
	}
	data, err := json.Marshal(outgoing)
	if err != nil {
		log.Printf("Error marshaling presence list: %v", err)
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

// broadcastPresenceLocked notifies all clients (except excludeUserID) about a user's status change.
// Pass excludeUserID = -1 to notify all clients.
// Must be called with h.mu held.
func (h *Hub) broadcastPresenceLocked(excludeUserID int, msgType models.WSMessageType, userID int) {
	outgoing := models.WSOutgoingMessage{
		Type:      msgType,
		SenderID:  userID,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(outgoing)
	if err != nil {
		log.Printf("Error marshaling presence message: %v", err)
		return
	}
	for id, c := range h.clients {
		if id == excludeUserID {
			continue
		}
		select {
		case c.Send <- data:
		default:
		}
	}
}
