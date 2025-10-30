package service

import "errors"

var (
	ErrFieldIsEmpty       = errors.New("field is empty")
	ErrUnknownEmail       = errors.New("unknown email")
	ErrPasswordUnknown    = errors.New("unknown password")
	ErrInvalidPhoneNumber = errors.New("invalid phone number (should be +7-XXX-XXX-XX-XX)")
)
