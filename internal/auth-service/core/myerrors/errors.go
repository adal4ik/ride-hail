package myerrors

import "errors"

var (
	ErrFieldIsEmpty       = errors.New("field is empty")
	ErrUnknownEmail       = errors.New("unknown email")
	ErrPasswordUnknown    = errors.New("unknown password")
	ErrInvalidPhoneNumber = errors.New("invalid phone number (should be +7-XXX-XXX-XX-XX)")

	ErrEmailRegistered               = errors.New("email already registered")
	ErrDriverLicenseNumberRegistered = errors.New("driver licence number is already registered")
)
