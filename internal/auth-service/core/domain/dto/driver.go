package dto

import "encoding/json"

type DriverRegistrationRequest struct {
	Username      string           `json:"username"`
	Email         string           `json:"email"`
	Password      string           `json:"password"`
	LicenseNumber string           `json:"license_number"`
	VehicleType   string           `json:"vehicle_type"`
	VehicleAttrs  *json.RawMessage `json:"vehicle_attrs"`
}

type DriverAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
