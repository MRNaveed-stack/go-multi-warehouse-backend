package middleware

import (
	"bytes"
	"log"
	"net/http"
	"pureGo/config"
	"pureGo/models"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}

func CheckIdempotency(key string) (int, string, bool) {
	var code int
	var body string

	query := `SELECT response_code, response_body FROM idempotency_keys WHERE id_key = $1`

	err := config.DB.QueryRow(query, key).Scan(&code, &body)
	if err != nil {
		return 0, "", false
	}

	return code, body, true
}

func SaveIdempotency(key string, userID int, code int, body string) {
	query := `INSERT INTO idempotency_keys (id_key, user_id, response_code, response_body) 
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (id_key) DO NOTHING` // Safety check for race conditions

	_, err := config.DB.Exec(query, key, userID, code, body)
	if err != nil {
		log.Printf("Failed to save idempotency key: %v", err)
	}
}

func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := getIdempotencyKey(r)
		if key == "" || r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}
		code, body, exists := CheckIdempotency(key)
		if exists {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotency-Key", key)
			w.Header().Set("X-Idempotency-Hit", "true")
			w.WriteHeader(code)
			w.Write([]byte(body))
			return
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           new(bytes.Buffer),
		}

		// Echo the accepted idempotency key so clients can verify correlation.
		w.Header().Set("X-Idempotency-Key", key)
		w.Header().Set("X-Idempotency-Hit", "false")

		next.ServeHTTP(rec, r)

		// Extract user_id from JWT claims in context
		userID := extractUserIDFromContext(r)
		if userID > 0 {
			SaveIdempotency(key, userID, rec.statusCode, rec.body.String())
		} else {
			log.Printf("Idempotency: Failed to extract user_id from context")
		}
	})
}

func getIdempotencyKey(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
}

func extractUserIDFromContext(r *http.Request) int {
	claimsVal := r.Context().Value("user")
	claims, ok := claimsVal.(jwt.MapClaims)
	if !ok {
		return 0
	}

	emailVal, ok := claims["email"]
	email, okEmail := emailVal.(string)
	if !ok || !okEmail || strings.TrimSpace(email) == "" {
		return 0
	}

	user, _, err := models.GetUserByEmail(strings.TrimSpace(email))
	if err != nil {
		return 0
	}

	return user.ID
}
