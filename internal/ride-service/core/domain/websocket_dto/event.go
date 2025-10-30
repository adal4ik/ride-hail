package websocketdto

import "encoding/json"

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type RideComplete struct {
	Tips   uint   `json:"tips"`
	Rating uint   `json:"rating"`
	RideId string `json:"ride_id"`
}
