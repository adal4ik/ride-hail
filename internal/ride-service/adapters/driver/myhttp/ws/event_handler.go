package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/golang-jwt/jwt"
)

type EventHandle func(c *Client, e websocketdto.Event) error

type EventHandler struct {
	passengerService ports.IPassengerService
	accessToken      string
}

func NewEventHandler(accessToken string, passengerService ports.IPassengerService) *EventHandler {
	return &EventHandler{
		accessToken:      accessToken,
		passengerService: passengerService,
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

	client.cancelAuth()

	return nil
}

func (eh *EventHandler) RideCompleteHandler(c *Client, e websocketdto.Event) error {
	eventData := websocketdto.RideComplete{}
	err := json.Unmarshal(e.Data, &eventData)
	if err != nil {
		return err
	}

	eh.passengerService.CompleteRide(c.passengerId, eventData.RideId, eventData.Rating, eventData.Tips)

	return nil
}
