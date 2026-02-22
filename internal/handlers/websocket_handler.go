package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/HaykAghajanyan/chat-backend/internal/middleware"
	"github.com/HaykAghajanyan/chat-backend/internal/models"
	"github.com/HaykAghajanyan/chat-backend/internal/repository"
	"github.com/HaykAghajanyan/chat-backend/internal/service"
	ws "github.com/HaykAghajanyan/chat-backend/internal/websocket"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	hub         *ws.Hub
	messageRepo *repository.MessageRepository
	authService *service.AuthService
	upgrader    websocket.Upgrader
}

func NewWebSocketHandler(hub *ws.Hub, messageRepo *repository.MessageRepository, authService *service.AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		messageRepo: messageRepo,
		authService: authService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Try to get user ID from context first (from middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())

	// If not in context, try query parameter token
	if !ok {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		// Validate token
		var err error
		userID, err = h.authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &ws.Client{
		UserID: userID,
		Conn:   &ws.Connection{Ws: conn},
		Send:   make(chan []byte, 256),
	}

	// Register client with hub
	h.hub.Register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump(h.hub, h.handleMessage)
}

func (h *WebSocketHandler) handleMessage(client *ws.Client, wsMsg *models.WSMessage) {
	switch wsMsg.Type {
	case models.WSMessageTypeChat:
		h.handleChatMessage(client, wsMsg)
	case models.WSMessageTypeTyping:
		h.handleTypingMessage(client, wsMsg)
	case models.WSMessageTypeRead:
		h.handleReadMessage(client, wsMsg)
	default:
		log.Printf("Unknown message type: %s", wsMsg.Type)
	}
}

func (h *WebSocketHandler) handleChatMessage(client *ws.Client, wsMsg *models.WSMessage) {
	// Save message to database
	message := &models.Message{
		SenderID:    client.UserID,
		RecipientID: wsMsg.Recipient,
		Content:     wsMsg.Content,
		IsRead:      false,
	}

	if err := h.messageRepo.Create(message); err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// Send to recipient if online
	outgoing := models.WSOutgoingMessage{
		Type:      models.WSMessageTypeChat,
		Message:   message,
		SenderID:  client.UserID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(outgoing)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.hub.SendToUser(wsMsg.Recipient, data)

	// Send confirmation back to sender
	h.hub.SendToUser(client.UserID, data)
}

func (h *WebSocketHandler) handleTypingMessage(client *ws.Client, wsMsg *models.WSMessage) {
	// Forward typing indicator to recipient
	outgoing := models.WSOutgoingMessage{
		Type:      models.WSMessageTypeTyping,
		SenderID:  client.UserID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(outgoing)
	if err != nil {
		log.Printf("Error marshaling typing message: %v", err)
		return
	}

	h.hub.SendToUser(wsMsg.Recipient, data)
}

func (h *WebSocketHandler) handleReadMessage(client *ws.Client, wsMsg *models.WSMessage) {
	// Mark messages from wsMsg.Recipient to client.UserID as read
	if err := h.messageRepo.MarkAsRead(wsMsg.Recipient, client.UserID); err != nil {
		log.Printf("Error marking message as read: %v", err)
		return
	}

	// Notify the original sender that their messages were read
	outgoing := models.WSOutgoingMessage{
		Type:      models.WSMessageTypeRead,
		SenderID:  client.UserID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(outgoing)
	if err != nil {
		log.Printf("Error marshaling read message: %v", err)
		return
	}

	h.hub.SendToUser(wsMsg.Recipient, data)
}
