package dto

import "encoding/json"

type UserRegistrationRequest struct {
	Username  string           `json:"username"`
	Email     string           `json:"email"`
	Password  string           `json:"password"`
	Role      string           `json:"role"`
	UserAttrs *json.RawMessage `json:"user_attrs"`
}

type UserAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
