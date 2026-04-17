package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"pureGo/models"
	"pureGo/utils"
)

func Signup(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	hashed, _ := utils.HashPassword(u.Password)
	u.Password = hashed

	vToken := utils.GenerateRandomToken()
	u.VerificationToken = vToken

	if err := u.Create(); err != nil {
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	}

	err := utils.SendVerification(u.Email, vToken)
	if err != nil {
		log.Println("[Signup] Email sending failed:", err)
		http.Error(w, "Failed to send verification email", 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Signup Successful. Please verify email"})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var input models.User

	// 1. Decode JSON
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("[Login] ERROR: Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Get user from database
	user, hashedPass, err := models.GetUserByEmail(input.Email)
	if err != nil {
		log.Println("[Login] ERROR: User not found in database")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 3. Check password
	if !utils.CheckPassword(input.Password, hashedPass) {
		log.Println("[Login] ERROR: Password verification failed")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 4. Generate tokens
	at, rt, err := utils.GenerateToken(user.Email, user.Role)
	if err != nil {
		log.Println("[Login] ERROR: Failed to generate tokens:", err)
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}
	log.Println("[Login] Login successful for user ID:", user.ID)

	// 5. Send Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  at,
		"refresh_token": rt,
	})
}

func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token is required", http.StatusUnauthorized)

		return
	}
	err := models.VerifyUser(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Email verified successfully. You can now login."})
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&input)
	user, _, err := models.GetUserByEmail(input.Email)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	token := utils.GenerateRandomToken()
	user.SetResetToken(token)
	go utils.SendResetEmail(user.Email, token)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password reset Link sent"})

}

func ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	json.NewDecoder(r.Body).Decode(&input)
	hashed, _ := utils.HashPassword(input.NewPassword)
	err := models.ResetPassword(input.Token, hashed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
}
