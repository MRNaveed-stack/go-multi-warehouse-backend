package utils

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"encoding/hex"

	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is not set")
	}
	return []byte(secret), nil
}

func GenerateToken(email string, role string) (string, string, error) {
	secretKey, err := getJWTSecret()
	if err != nil {
		return "", "", err
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(time.Minute * 15).Unix(),
	})
	at, err := accessToken.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
	})
	rt, err := refreshToken.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	return at, rt, nil
}

func GenerateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
