package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"ride-hail/internal/auth-service/core/domain/dto"
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

func getAllowedRoles() []string {
	return []string{"PASSENGER", "ADMIN", "DRIVER"}
}

var AllowedRoles = map[string]bool{
	"PASSENGER": true,
	"ADMIN":     true,
	"DRIVER":    true,
}

func validateUserRegistration(ctx context.Context, regReq dto.UserRegistrationRequest) error {
	if err := validateName(regReq.Username); err != nil {
		return fmt.Errorf("invalid name: %v", err)
	}

	if err := validateEmail(regReq.Email); err != nil {
		return fmt.Errorf("invalid email: %v", err)
	}

	if err := validatePassword(regReq.Password); err != nil {
		return fmt.Errorf("invalid password: %v", err)
	}

	if regReq.UserAttrs != nil && len(*regReq.UserAttrs) > 0 {
		if err := validateUserAttrs(*regReq.UserAttrs); err != nil {
			if errors.Is(err, ErrInvalidPhoneNumber) {
				return ErrInvalidPhoneNumber
			}
			return fmt.Errorf("invalid user attributes: %v", err)
		}
	}
	if err := validateRole(regReq.Role); err != nil {
		return fmt.Errorf("invalid role: %v", err)
	}

	return nil
}

func validateRole(role string) error {
	if role == "" {
		return ErrFieldIsEmpty
	}
	role = strings.ToUpper(role)
	if ok := AllowedRoles[role]; !ok {
		return fmt.Errorf("invalid role: %s. Allowed values: %v",
			role, getAllowedRoles())
	}
	return nil
}

func validateUserAttrs(userAttrs json.RawMessage) error {
	if !json.Valid(userAttrs) {
		return fmt.Errorf("invalid JSON format for user attributes")
	}

	var attrs map[string]interface{}
	if err := json.Unmarshal(userAttrs, &attrs); err != nil {
		return err
	}

	// Ensure "phone" attribute exists and is a valid phone number
	phone, exists := attrs["phone"].(string)
	if exists && !isValidPhoneNumber(phone) {
		return ErrInvalidPhoneNumber
	}

	return nil
}

func validateLogin(ctx context.Context, email, password string) error {
	if err := validateEmail(email); err != nil {
		return fmt.Errorf("invalid email: %v", err)
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

// ================ Driver ================

var AllowedVehicleTypes = map[string]bool{
	"ECONOMY": true,
	"PREMIUM": true,
	"XL":      true,
}

func getAllowedVehicleTypes() []string {
	return []string{"ECONOMY", "PREMIUM", "XL"}
}

// Ensure all required vehicle attributes are present

func getVehicleAtributesRequiredFields() []string {
	return []string{"make", "model", "color", "plate", "year"}
}

const (
	MinLicenseNumberLen = 5
	MaxLicenseNumberLen = 20
	// Add other constants as needed
)

func validateDriverRegistration(ctx context.Context, licenseNumber, vehicleType string, vehicleAttrs *json.RawMessage) error {
	if err := validateLicenseNumber(licenseNumber); err != nil {
		return fmt.Errorf("invalid license number: %v", err)
	}

	if err := validateVehicleType(vehicleType); err != nil {
		return fmt.Errorf("invalid vehicle type: %v", err)
	}

	// If vehicle attributes are provided, validate they are proper JSON
	if vehicleAttrs == nil || len(*vehicleAttrs) == 0 {
		return fmt.Errorf("vehicle attributes not specified")
	}

	if err := validateVehicleAttrs(*vehicleAttrs); err != nil {
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
		return fmt.Errorf("invalid license number format: %s. Allowed format is uppercase letters and numbers", licenseNumber)
	}

	return nil
}

func validateVehicleType(vehicleType string) error {
	if vehicleType == "" {
		return ErrFieldIsEmpty
	}

	// Convert to uppercase for consistency with enum
	vehicleType = strings.ToUpper(vehicleType)

	if !AllowedVehicleTypes[vehicleType] {
		return fmt.Errorf("invalid vehicle type: %s. Allowed values: %v",
			vehicleType, getAllowedVehicleTypes())
	}

	return nil
}

func validateVehicleAttrs(vehicleAttrs json.RawMessage) error {
	if !json.Valid(vehicleAttrs) {
		return fmt.Errorf("invalid JSON format for vehicle attributes")
	}

	// Optional: Validate specific vehicle attribute structure based on vehicle type
	var attrs map[string]interface{}
	if err := json.Unmarshal(vehicleAttrs, &attrs); err != nil {
		return err
	}

	for _, field := range getVehicleAtributesRequiredFields() {
		if _, exists := attrs[field]; !exists {
			return fmt.Errorf("missing %s attribute. Required fields: %v", field, getVehicleAtributesRequiredFields())
		}
		// Check if "year" is a valid number (e.g., greater than 1885 for the first car)
		if field == "year" {
			year, ok := attrs["year"].(float64) // JSON unmarshals numbers as float64
			if !ok || year < 1885 {
				return errors.New("invalid or missing vehicle year")
			}
		} else {
			s, ok := attrs[field].(string) // JSON unmarshals numbers as float64
			if !ok || len(s) == 0 || len(s) > 20 {
				return fmt.Errorf("invalid or missing vehicle %s", field)
			}
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

// Helper function to validate phone numbers (you can adapt this regex based on your needs)
func isValidPhoneNumber(phone string) bool {
	// Simple regex for a phone number (e.g., +7-XXX-XXX-XX-XX)
	re := regexp.MustCompile(`^\+7-\d{3}-\d{3}-\d{2}-\d{2}$`)
	return re.MatchString(phone)
}
