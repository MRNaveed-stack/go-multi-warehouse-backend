package controllers

import (
	"encoding/json"
	"net/http"
	"pureGo/config"
	"pureGo/models"
	"time"
)

func GetAdminDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	val, err := config.Redis.Get(ctx, "admin_dashboard").Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(val))
		return
	}
	stats, err := models.GetMultiWarehouseStats()
	if err != nil {
		http.Error(w, "Failed to load stats", 500)
		return
	}
	data, _ := json.Marshal(stats)
	config.Redis.Set(ctx, "admin_dashboard", data, 10*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(stats)
}
