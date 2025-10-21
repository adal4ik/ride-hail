package middleware

import (
	"net/http"
	"strings"

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

func (am *AuthMiddleware) Middle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Пошел нахуй", http.StatusBadRequest)
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(am.accessSecret), nil
		})
		if err != nil {
			http.Error(w, "Пошел нахуй", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			http.Error(w, "Пошел нахуй", http.StatusBadRequest)
			return
		}

		// valid by time also

		next.ServeHTTP(w, r)
	})
}
