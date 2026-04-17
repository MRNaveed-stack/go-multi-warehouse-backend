package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"pureGo/models"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func TransferStockHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProductID       int    `json:"product_id"`
		FromWarehouseID int    `json:"from_warehouse_id"`
		ToWarehouseID   int    `json:"to_warehouse_id"`
		Quantity        int    `json:"quantity"`
		Reason          string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	claimsVal := r.Context().Value("user")
	claims, ok := claimsVal.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	emailVal, ok := claims["email"]
	email, okEmail := emailVal.(string)
	if !ok || !okEmail || strings.TrimSpace(email) == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, _, err := models.GetUserByEmail(strings.TrimSpace(email))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = models.TransferStock(input.ProductID, input.FromWarehouseID, input.ToWarehouseID, input.Quantity, user.ID, input.Reason)
	if err != nil {
		log.Printf("[TransferStockHandler] ERROR: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Transfer successful"})
}
