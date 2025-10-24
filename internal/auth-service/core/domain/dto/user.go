package dto

type UserRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
