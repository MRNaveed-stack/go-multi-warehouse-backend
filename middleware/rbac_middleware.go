package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func RoleMiddleware(requiredRole string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claimsVal := r.Context().Value("user")
		if claimsVal == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userClaims, ok := claimsVal.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		roleVal, ok := userClaims["role"]
		if !ok {
			http.Error(w, "Forbidden: role not found", http.StatusForbidden)
			return
		}

		role, ok := roleVal.(string)
		if !ok {
			http.Error(w, "Forbidden: invalid role claim", http.StatusForbidden)
			return
		}

		role = strings.ToLower(strings.TrimSpace(role))
		requiredRole = strings.ToLower(strings.TrimSpace(requiredRole))

		if role != requiredRole {
			http.Error(w, "Forbidden: You do not have the required permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
