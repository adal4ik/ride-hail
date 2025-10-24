package ws

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/golang-jwt/jwt"
)

type EventHandle func(c *Client, e websocketdto.Event) error

type EventHandler struct {
	accessToken string
}

func NewEventHandler(accessToken string) *EventHandler {
	return &EventHandler{
		accessToken: accessToken,
	}
}

func (eh *EventHandler) AuthHandler(client *Client, e websocketdto.Event) error {
	var token websocketdto.AuthMessage
	err := json.Unmarshal(e.Data, &token)
	if err != nil {
		return err
	}
	tokenString := strings.TrimPrefix(token.Token, "Bearer ")
	tokenJWT, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(eh.accessToken), nil
	})
	if err != nil {
		return err
	}

	if !tokenJWT.Valid {
		return err
	}
	claims, ok := tokenJWT.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("cannot get claim")
	}

	userId, ok := claims["user_id"].(string)
	if !ok {
		return fmt.Errorf("cannot get user_id")
	}

	if client.passengerId != userId {
		return fmt.Errorf("different id's")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("no exp")
	}

	if time.Now().Unix() > int64(exp) {
		return fmt.Errorf("nigga time is up")
	}
	client.cancelAuth()

	return nil
}
