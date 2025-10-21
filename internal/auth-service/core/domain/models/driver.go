package models

import (
	"encoding/json"
	"time"
)

type Driver struct {
	DriverId      string           `json:"driver_id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Username      string           `json:"username"`
	Email         string           `json:"email"`
	PasswordHash  []byte           `json:"password_hash"`
	Coord         *string          `json:"coord,omitempty"`
	LicenseNumber string           `json:"license_number"`
	VehicleType   string           `json:"vehicle_type"`
	VehicleAttrs  *json.RawMessage `json:"vehicle_attrs,omitempty"`
	Rating        *float64         `json:"rating,omitempty"`
	TotalRides    *int             `json:"total_rides,omitempty"`
	TotalEarnings *float64         `json:"total_earnings,omitempty"`
	Status        *string          `json:"status,omitempty"`
	IsVerified    *bool            `json:"is_verified,omitempty"`
}
