package repository

import (
	"database/sql"
	"fmt"

	"github.com/HaykAghajanyan/chat-backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(message *models.Message) error {
	query := `
INSERT INTO messages (sender_id, recipient_id, content, is_read)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at
`
	err := r.db.QueryRow(
		query,
		message.SenderID,
		message.RecipientID,
		message.Content,
		message.IsRead,
	).Scan(&message.ID, &message.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

func (r *MessageRepository) GetConversation(userID1, userID2 int, limit int) ([]models.Message, error) {
	query := `
SELECT * FROM messages
WHERE (sender_id = $1 AND recipient_id = $2)
OR (sender_id = $2 AND recipient_id = $1)
ORDER BY created_at DESC
LIMIT $3`

	var messages []models.Message
	err := r.db.Select(&messages, query, userID1, userID2, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	for i := len(messages)/2 - 1; i >= 0; i-- {
		opp := len(messages) - 1 - i
		messages[i], messages[opp] = messages[opp], messages[i]
	}

	return messages, nil
}

func (r *MessageRepository) MarkAsRead(userID1, userID2 int) error {
	query := `
UPDATE messages 
SET is_read = true
WHERE sender_id = $1 AND recipient_id = $2`

	result, err := r.db.Exec(query, userID1, userID2)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *MessageRepository) GetUnreadCount(userID int) (int, error) {
	query := `
SELECT COUNT(*) FROM messages
WHERE recipient_id = $1 AND is_read = false`

	var count int
	err := r.db.Get(&count, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get message unread count: %w", err)
	}

	return count, nil
}

func (r *MessageRepository) GetConversationList(userID int) ([]models.ConversationPreview, error) {
	query := `
        WITH ranked_messages AS (
            SELECT 
                CASE 
                    WHEN sender_id = $1 THEN recipient_id 
                    ELSE sender_id 
                END as other_user_id,
                content,
                created_at,
                is_read,
                sender_id,
                ROW_NUMBER() OVER (
                    PARTITION BY CASE 
                        WHEN sender_id = $1 THEN recipient_id 
                        ELSE sender_id 
                    END 
                    ORDER BY created_at DESC
                ) as rn
            FROM messages
            WHERE sender_id = $1 OR recipient_id = $1
        )
        SELECT 
            other_user_id,
            content as last_message,
            created_at as last_message_at,
            is_read,
            sender_id as last_sender_id
        FROM ranked_messages
        WHERE rn = 1
        ORDER BY created_at DESC
    `

	var conversations []models.ConversationPreview
	err := r.db.Select(&conversations, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation list: %w", err)
	}

	return conversations, nil
}
