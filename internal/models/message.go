package models

import (
	"time"
)

type Message struct {
	ID          int       `db:"id" json:"id"`
	SenderID    int       `db:"sender_id" json:"sender_id"`
	RecipientID int       `db:"recipient_id" json:"recipient_id"`
	Content     string    `db:"content" json:"content"`
	IsRead      bool      `db:"is_read" json:"is_read"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type WSMessageType string

const (
	WSMessageTypeChat     WSMessageType = "chat"
	WSMessageTypeTyping   WSMessageType = "typing"
	WSMessageTypeRead     WSMessageType = "read"
	WSMessageTypeOnline   WSMessageType = "online"
	WSMessageTypeOffline  WSMessageType = "offline"
	WSMessageTypePresence WSMessageType = "presence"
)

type WSMessage struct {
	Type      WSMessageType `json:"type"`
	Content   string        `json:"content,omitempty"`
	Recipient int           `json:"recipient,omitempty"`
	MessageID int           `json:"message_id,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type WSOutgoingMessage struct {
	Type        WSMessageType `json:"type"`
	Message     *Message      `json:"message,omitempty"`
	MessageID   int           `json:"message_id,omitempty"`
	SenderID    int           `json:"sender_id,omitempty"`
	OnlineUsers []int         `json:"online_users,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

type ConversationPreview struct {
	OtherUserID   int       `db:"other_user_id" json:"other_user_id"`
	LastMessage   string    `db:"last_message" json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
	IsRead        bool      `db:"is_read" json:"is_read"`
	LastSenderID  int       `db:"last_sender_id" json:"last_sender_id"`
}
