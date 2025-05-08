package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"pornterest/internal/config" // Импортируем пакет config

	"github.com/golang-jwt/jwt/v5"
)

// contextKey - собственный тип для ключей контекста
type contextKey string

const UserID contextKey = "user_id"

// AuthMiddleware создает middleware для проверки JWT токена, используя предоставленную конфигурацию
func AuthMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	jwtSecret := cfg.JWTSecret // Получаем JWTSecret из конфигурации

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.Split(authHeader, " ")
			if len(tokenString) != 2 || tokenString[0] != "Bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString[1], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				log.Printf("Failed to parse JWT: %v", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userID, ok := claims["user_id"].(float64) // JWT stores numbers as float64
				if !ok {
					http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
					return
				}

				// Добавляем ID пользователя в контекст запроса, используя наш собственный тип ключа
				ctx := context.WithValue(r.Context(), UserID, int(userID))
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}
