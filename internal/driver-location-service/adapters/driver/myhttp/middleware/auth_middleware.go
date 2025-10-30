package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"

	jwt "github.com/golang-jwt/jwt"
)

type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware(accessSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		accessSecret: accessSecret,
	}
}

func (am *AuthMiddleware) SessionHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			handlers.JsonError(w, http.StatusBadRequest, fmt.Errorf("Empty JWT-Token"))
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(am.accessSecret), nil
		})
		if err != nil {
			handlers.JsonError(w, http.StatusBadRequest, fmt.Errorf("Failed to parse JWT-Token"))
			return
		}

		if !token.Valid {
			handlers.JsonError(w, http.StatusBadRequest, fmt.Errorf("Invalid JWT-Token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			handlers.JsonError(w, http.StatusBadRequest, fmt.Errorf("Invalid claims"))
			return
		}

		userId, ok := claims["user_id"].(string)
		if !ok {
			handlers.JsonError(w, http.StatusUnauthorized, fmt.Errorf("Username not found in token"))
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			handlers.JsonError(w, http.StatusUnauthorized, fmt.Errorf("Role not found in token"))
			return
		}

		if role != "DRIVER" {
			handlers.JsonError(w, http.StatusBadRequest, fmt.Errorf("Only drivers allowed to use this service"))
			return
		}

		r.Header.Set("X-UserId", userId)

		next.ServeHTTP(w, r)
	})
}
