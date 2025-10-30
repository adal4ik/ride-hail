package models

import (
	"encoding/json"
	"time"
)

type User struct {
	UserId       string           `json:"user_id"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Username     string           `json:"username"`
	Email        string           `json:"email"`
	PasswordHash []byte           `json:"password_hash"`
	Role         string           `json:"role"`
	Status       *string          `json:"status,omitempty"`
	UserAttrs    *json.RawMessage `json:"user_attrs,omitempty"`
}
