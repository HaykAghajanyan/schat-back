package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/HaykAghajanyan/chat-backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	var displayName interface{}
	if user.DisplayName.Valid {
		displayName = user.DisplayName.String
	} else {
		displayName = nil
	}

	err := r.db.QueryRow(query, user.Username, user.Email, user.PasswordHash, displayName).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.Get(user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT * FROM users WHERE username = $1`

	err := r.db.Get(user, query, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(id int) (*models.User, error) {
	user := &models.User{}
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.Get(user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// SearchUsers searches for users by username or email
func (r *UserRepository) SearchUsers(query string, limit int) ([]models.User, error) {
	searchQuery := `
        SELECT * FROM users 
        WHERE username ILIKE $1 OR email ILIKE $1
        LIMIT $2
    `

	var users []models.User
	searchPattern := "%" + query + "%"
	err := r.db.Select(&users, searchQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) GetAllUsers(excludeUserID int, limit int) ([]models.User, error) {
	query := `
        SELECT * FROM users 
        WHERE id != $1
        ORDER BY username
        LIMIT $2
    `

	var users []models.User
	err := r.db.Select(&users, query, excludeUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}
