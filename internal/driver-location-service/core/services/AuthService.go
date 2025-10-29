package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type AuthService struct {
	secretKey string
}

func NewAuthService(secretKey string) *AuthService {
	return &AuthService{
		secretKey: secretKey,
	}
}

type DriverClaims struct {
	DriverID string `json:"driver_id"`
	Role     string `json:"role"`
}

func (a *AuthService) ValidateDriverToken(tokenString string) (string, error) {
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.secretKey), nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return "", fmt.Errorf("token expired")
			}
		}

		role, ok := claims["role"].(string)
		if !ok || role != "DRIVER" {
			return "", fmt.Errorf("invalid role")
		}

		driverID, ok := claims["driver_id"].(string)
		if !ok || driverID == "" {
			return "", fmt.Errorf("driver_id is required")
		}

		return driverID, nil
	}

	return "", fmt.Errorf("invalid token")
}
