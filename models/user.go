package models

import (
	"fmt"
	"log"
	"pureGo/config"
	"strings"
	"time"
)

type User struct {
	ID                int    `json:"id"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	Password          string `json:"password,omitempty"`
	IsVerified        bool   `json:"is_verified"`
	VerificationToken string `json:"verification_token"`
}

func (u *User) Create() error {
	role := strings.ToLower(strings.TrimSpace(u.Role))
	if role == "" {
		role = "user"
	}
	if role != "admin" {
		role = "user"
	}

	query := `
	INSERT INTO users (email, password, verification_token, is_verified, role)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
	`

	return config.DB.QueryRow(
		query,
		u.Email,
		u.Password,
		u.VerificationToken,
		false,
		role,
	).Scan(&u.ID)
}

func GetUserByEmail(email string) (*User, string, error) {
	var u User
	var hashedPassword string
	query := `SELECT id , email , password, is_verified,role FROM users WHERE email = $1`
	err := config.DB.QueryRow(query, email).Scan(&u.ID, &u.Email, &hashedPassword, &u.IsVerified, &u.Role)
	return &u, hashedPassword, err
}

func VerifyUser(token string) error {
	query := `UPDATE users SET is_verified = true, verification_token = NULL
	WHERE verification_token = $1
	`
	result, err := config.DB.Exec(query, token)
	if err != nil {
		return err
	}
	log.Println("[Verify] Token received:", token)
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("Invalid or expired token")
	}
	return nil

}

func LogoutUser(email string) error {
	query := `UPDATE users SET refresh_token = NULL WHERE email = $1`
	_, err := config.DB.Exec(query, email)
	return err
}

// SetResetToken (expire in 15 minutes)
func (u *User) SetResetToken(token string) error {
	expiry := time.Now().Add(time.Minute * 15)
	query := `UPDATE users SET reset_token = $1 , reset_token_expiry = $2 WHERE email = $3`
	_, err := config.DB.Exec(query, token, expiry, u.Email)
	return err
}

func ResetPassword(token, newHashedPassword string) error {
	query := `UPDATE users SET password = $1, reset_token = NULL , reset_token_expiry = NULL
	WHERE reset_token = $2 AND reset_token_expiry > $3
	`
	result, err := config.DB.Exec(query, newHashedPassword, token, time.Now())
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("invalid token or expired reset token")
	}
	return nil
}
