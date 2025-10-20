package model

import (
	"encoding/json"
	"time"
)

type RideEvents struct {
	Id        string // uuid
	CreatedAt time.Time
	RideId    string
	EventType string
	EventData json.RawMessage
}
