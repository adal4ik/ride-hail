package websocketdto

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}
