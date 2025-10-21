package dto

type RegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
