package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/HaykAghajanyan/chat-backend/internal/middleware"
	"github.com/HaykAghajanyan/chat-backend/internal/models"
	"github.com/HaykAghajanyan/chat-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type MessageHandler struct {
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
}

func NewMessageHandler(messageRepo *repository.MessageRepository, userRepo *repository.UserRepository) *MessageHandler {
	return &MessageHandler{
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

// GetConversation returns messages between current user and another user
func (h *MessageHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	// Get other user ID from URL parameter
	otherUserIDStr := chi.URLParam(r, "userID")
	otherUserID, err := strconv.Atoi(otherUserIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid user ID"})
		return
	}

	// Get limit from query parameter (default 50)
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Verify other user exists
	otherUser, err := h.userRepo.GetByID(otherUserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to get user"})
		return
	}
	if otherUser == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "user not found"})
		return
	}

	// Get conversation
	messages, err := h.messageRepo.GetConversation(userID, otherUserID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to get conversation"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// GetUnreadCount returns the number of unread messages for current user
func (h *MessageHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	count, err := h.messageRepo.GetUnreadCount(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to get unread count"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unread_count": count})
}

// MarkAsRead marks a message as read
func (h *MessageHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	messageIDStr := chi.URLParam(r, "messageID")
	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid message ID"})
		return
	}

	if err := h.messageRepo.MarkAsRead(messageID, userID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to mark as read"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *MessageHandler) GetConversationList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
		return
	}

	conversations, err := h.messageRepo.GetConversationList(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to get conversations"})
		return
	}

	type ConversationWithUser struct {
		User          *models.User `json:"user"`
		LastMessage   string       `json:"last_message"`
		LastMessageAt time.Time    `json:"last_message_at"`
		IsRead        bool         `json:"is_read"`
		LastSenderID  int          `json:"last_sender_id"`
	}

	result := make([]ConversationWithUser, 0, len(conversations))
	for _, conv := range conversations {
		user, err := h.userRepo.GetByID(conv.OtherUserID)
		if err != nil || user == nil {
			continue
		}

		result = append(result, ConversationWithUser{
			User:          user,
			LastMessage:   conv.LastMessage,
			LastMessageAt: conv.LastMessageAt,
			IsRead:        conv.IsRead,
			LastSenderID:  conv.LastSenderID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
