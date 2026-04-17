package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"pureGo/models"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[AuthMiddleware] %s %s - auth check started", r.Method, r.URL.Path)

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			log.Printf("[AuthMiddleware] %s %s - missing Authorization header", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			log.Printf("[AuthMiddleware] %s %s - Authorization header is not Bearer format", r.Method, r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		tokenString = strings.Trim(tokenString, `"`)
		if tokenString == "" {
			log.Printf("[AuthMiddleware] %s %s - empty bearer token", r.Method, r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		log.Printf("[AuthMiddleware] %s %s - bearer token received (length=%d)", r.Method, r.URL.Path, len(tokenString))

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			log.Printf("[AuthMiddleware] %s %s - JWT_SECRET is empty", r.Method, r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				log.Printf("[AuthMiddleware] %s %s - unexpected signing method: %s", r.Method, r.URL.Path, token.Method.Alg())
				return nil, errors.New("unexpected signing method")
			}

			secretKey := []byte(secret)
			return secretKey, nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				log.Printf("[AuthMiddleware] %s %s - token expired", r.Method, r.URL.Path)
			} else if errors.Is(err, jwt.ErrTokenMalformed) {
				log.Printf("[AuthMiddleware] %s %s - token malformed", r.Method, r.URL.Path)
			} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				log.Printf("[AuthMiddleware] %s %s - token signature invalid (secret mismatch likely)", r.Method, r.URL.Path)
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				log.Printf("[AuthMiddleware] %s %s - token not valid yet (nbf/iat issue)", r.Method, r.URL.Path)
			} else {
				log.Printf("[AuthMiddleware] %s %s - token parse error: %v", r.Method, r.URL.Path, err)
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Printf("[AuthMiddleware] %s %s - token parsed but invalid", r.Method, r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Printf("[AuthMiddleware] %s %s - unable to read token claims", r.Method, r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		emailVal, emailOk := claims["email"]
		email, emailIsString := emailVal.(string)
		if emailOk && emailIsString && strings.TrimSpace(email) != "" {
			user, _, err := models.GetUserByEmail(strings.TrimSpace(email))
			if err != nil || strings.TrimSpace(user.Role) == "" {
				log.Printf("[AuthMiddleware] %s %s - could not resolve role from DB", r.Method, r.URL.Path)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims["role"] = strings.ToLower(strings.TrimSpace(user.Role))
			log.Printf("[AuthMiddleware] %s %s - role synchronized from DB", r.Method, r.URL.Path)
		} else {
			roleVal, roleOk := claims["role"]
			roleStr, roleIsString := roleVal.(string)
			if !roleOk || !roleIsString || strings.TrimSpace(roleStr) == "" {
				log.Printf("[AuthMiddleware] %s %s - missing role and email claims", r.Method, r.URL.Path)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			claims["role"] = strings.ToLower(strings.TrimSpace(roleStr))
		}

		log.Printf("[AuthMiddleware] %s %s - token validated", r.Method, r.URL.Path)
		ctx := context.WithValue(r.Context(), "user", claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
