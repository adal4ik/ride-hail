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

	tokenJWT, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(a.secretKey), nil
	})
	if err != nil {
		return "", err
	}

	if !tokenJWT.Valid {
		return "", err
	}
	claims, ok := tokenJWT.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("Error: could not get claims")
	}

	userId, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("Error: no user_id")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", fmt.Errorf("Error: no exp")
	}

	if time.Now().Unix() > int64(exp) {
		return "", fmt.Errorf("Error: token expired")
	}

	return userId, nil
}
