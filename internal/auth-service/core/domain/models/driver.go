package models

import (
	"encoding/json"
	"time"
)

type Driver struct {
	UserId       string          `json:"user_id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Username     string          `json:"username"`
	Email        string          `json:"email"`
	Role         string          `json:"role"`
	Status       string          `json:"status"`
	PasswordHash []byte          `json:"password_hash"`
	Attrs        json.RawMessage `json:"attrs"`
}
