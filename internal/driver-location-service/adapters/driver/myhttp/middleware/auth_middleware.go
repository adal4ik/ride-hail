package middleware

import (
	"net/http"
	"strings"

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
			http.Error(w, "Empty JWT-Token", http.StatusBadRequest)
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(am.accessSecret), nil
		})
		if err != nil {
			http.Error(w, "Failed to parse JWT-Token", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid JWT-Token", http.StatusBadRequest)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		userId, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Username not found in token", http.StatusUnauthorized)
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			http.Error(w, "role not found in token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-UserId", userId)
		r.Header.Set("X-Role", role)

		next.ServeHTTP(w, r)
	})
}
