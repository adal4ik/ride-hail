package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"ride-hail/internal/ride-service/adapters/driver/myhttp/handle"

	"github.com/golang-jwt/jwt"
)

type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware(accessSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		accessSecret: accessSecret,
	}
}

func (am *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			handle.JsonError(w, http.StatusBadRequest, fmt.Errorf("Empty JWT-Token"))
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(am.accessSecret), nil
		})
		if err != nil {
			handle.JsonError(w, http.StatusBadRequest, fmt.Errorf("Failed to parse JWT-Token"))
			return
		}

		if !token.Valid {
			handle.JsonError(w, http.StatusBadRequest, fmt.Errorf("Invalid JWT-Token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			handle.JsonError(w, http.StatusBadRequest, fmt.Errorf("Invalid claims"))
			return
		}

		userId, ok := claims["user_id"].(string)
		if !ok {
			handle.JsonError(w, http.StatusUnauthorized, fmt.Errorf("Username not found in token"))
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			handle.JsonError(w, http.StatusUnauthorized, fmt.Errorf("Role not found in token"))
			return
		}

		if role != "PASSENGER" {
			handle.JsonError(w, http.StatusBadRequest, fmt.Errorf("Only passengers allowed to use this service"))
			return
		}

		r.Header.Set("X-UserId", userId)

		next.ServeHTTP(w, r)
	})
}
