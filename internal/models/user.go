package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type User struct {
	ID           int            `db:"id" json:"id"`
	Username     string         `db:"username" json:"username"`
	Email        string         `db:"email" json:"email"`
	PasswordHash string         `db:"password_hash" json:"-"`
	DisplayName  sql.NullString `db:"display_name" json:"-"`
	AvatarURL    sql.NullString `db:"avatar_url" json:"-"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User

	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}

	var avatarURL *string
	if u.AvatarURL.Valid {
		avatarURL = &u.AvatarURL.String
	}

	return json.Marshal(&struct {
		*Alias
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}{
		Alias:       (*Alias)(&u),
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	})
}

type RegisterRequest struct {
	Username    string `json:"username" validate:"required,min=2,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	DisplayName string `json:"display_name" validate:"max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
