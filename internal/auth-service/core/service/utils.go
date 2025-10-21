package service

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinCustomerNameLen = 1
	MaxCustomerNameLen = 100

	MinEmailLen = 5
	MaxEmailLen = 100

	MinPasswordLen = 5
	MaxPasswordLen = 50

	HashFactor = 10

	TokenLen = 32
)

var AllowedRoles = map[string]bool{
	"PASSENGER": true,
	"ADMIN":     true,
	"DRIVER":    true,
}

var (
	ErrFieldIsEmpty    = errors.New("field is empty")
	ErrUsernameUnknown = errors.New("unknown username")
	ErrPasswordUnknown = errors.New("unknown password")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailRegistered = errors.New("email already registered")
)

func validateName(username string) error {
	if username == "" {
		return ErrFieldIsEmpty
	}

	usernameLen := len(username)
	if usernameLen < MinCustomerNameLen || usernameLen > MaxCustomerNameLen {
		return fmt.Errorf("must be in range [%d, %d]", MinCustomerNameLen, MaxCustomerNameLen)
	}

	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrFieldIsEmpty
	}

	emailLen := len(email)
	if emailLen < MinEmailLen || emailLen > MaxEmailLen {
		return fmt.Errorf("must be in range [%d, %d]", MinEmailLen, MaxEmailLen)
	}

	if strings.Count(email, "@") != 1 {
		return fmt.Errorf("must contain only one @: %s", email)
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrFieldIsEmpty
	}

	passwordLen := len(password)
	if passwordLen < MinPasswordLen || passwordLen > MaxPasswordLen {
		return fmt.Errorf("must be in range [%d, %d]", MinPasswordLen, MaxPasswordLen)
	}
	return nil
}

func hashPassword(password string) ([]byte, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), HashFactor)
	return bytes, err
}

func checkPassword(hashed []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hashed, []byte(password)) == nil
}
