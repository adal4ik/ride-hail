package db

import "errors"

var (
	ErrEmailRegistered               = errors.New("email already registered")
	ErrDriverLicenseNumberRegistered = errors.New("driver licence number is already registered")
	ErrUnknownEmail                  = errors.New("unknown email")
)
