package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
	ErrUnknownEmail    = errors.New("unknown email")
	ErrPasswordUnknown = errors.New("unknown password")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailRegistered = errors.New("email already registered")
)

func validateRegistration(ctx context.Context, username, email, password string) error {
	if err := validateName(username); err != nil {
		return fmt.Errorf("invalid name: %v", err)
	}

	if err := validateEmail(email); err != nil {
		return fmt.Errorf("invalid email: %v", err)
	}

	if err := validatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	return nil
}

func validateLogin(ctx context.Context, email, password string) error {
	if err := validateEmail(email); err != nil {
		return fmt.Errorf("invalid username: %v", err)
	}

	if err := validatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}
	return nil
}

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

// ================ Driver

const (
	MinLicenseNumberLen = 5
	MaxLicenseNumberLen = 20
	// Add other constants as needed
)

func validateDriverRegistration(ctx context.Context, licenseNumber, vehicleType string, vehicleAttrs json.RawMessage) error {
	if err := validateLicenseNumber(licenseNumber); err != nil {
		return fmt.Errorf("invalid license number: %v", err)
	}

	if err := validateVehicleType(vehicleType); err != nil {
		return fmt.Errorf("invalid vehicle type: %v", err)
	}

	if err := validateVehicleAttrs(vehicleAttrs); err != nil {
		return fmt.Errorf("invalid vehicle attributes: %v", err)
	}

	return nil
}

func validateLicenseNumber(licenseNumber string) error {
	if licenseNumber == "" {
		return ErrFieldIsEmpty
	}

	licenseLen := len(licenseNumber)
	if licenseLen < MinLicenseNumberLen || licenseLen > MaxLicenseNumberLen {
		return fmt.Errorf("must be in range [%d, %d]", MinLicenseNumberLen, MaxLicenseNumberLen)
	}

	// Basic format validation - adjust based on your country's license format
	if !isValidLicenseFormat(licenseNumber) {
		return fmt.Errorf("invalid license number format: %s", licenseNumber)
	}

	return nil
}

func validateVehicleType(vehicleType string) error {
	if vehicleType == "" {
		return ErrFieldIsEmpty
	}

	// Convert to uppercase for consistency with enum
	vehicleType = strings.ToUpper(vehicleType)

	// Validate against allowed vehicle types
	allowedTypes := map[string]bool{
		"ECONOMY": true,
		"PREMIUM": true,
		"XL":      true,
		// Add other vehicle types as needed
	}

	if !allowedTypes[vehicleType] {
		return fmt.Errorf("invalid vehicle type: %s. Allowed values: %v",
			vehicleType, getAllowedVehicleTypes())
	}

	return nil
}

func validateVehicleAttrs(vehicleAttrs json.RawMessage) error {
	// If vehicle attributes are provided, validate they are proper JSON
	if len(vehicleAttrs) > 0 {
		if !json.Valid(vehicleAttrs) {
			return fmt.Errorf("invalid JSON format for vehicle attributes")
		}

		// Optional: Validate specific vehicle attribute structure based on vehicle type
		if err := validateVehicleAttrsStructure(vehicleAttrs); err != nil {
			return fmt.Errorf("invalid vehicle attributes structure: %v", err)
		}
	}

	return nil
}

// Helper functions
func isValidLicenseFormat(licenseNumber string) bool {
	// Basic license number validation - adjust based on your requirements
	// Example: alphanumeric, no special characters except hyphens
	matched, _ := regexp.MatchString(`^[A-Z0-9-]+$`, licenseNumber)
	return matched
}

func getAllowedVehicleTypes() []string {
	return []string{"ECONOMY", "PREMIUM", "XL"} // Add your actual enum values
}

func validateVehicleAttrsStructure(vehicleAttrs json.RawMessage) error {
	// Optional: Validate specific structure based on your requirements
	// Example: Check if required fields exist for specific vehicle types

	var attrs map[string]interface{}
	if err := json.Unmarshal(vehicleAttrs, &attrs); err != nil {
		return err
	}

	// Add specific validation logic here if needed
	// For example, for cars you might require "model", "year", etc.

	return nil
}
