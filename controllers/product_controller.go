package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"pureGo/config"
	"pureGo/models"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getUserIDFromRequest(r *http.Request) (int, error) {
	claimsVal := r.Context().Value("user")
	claims, ok := claimsVal.(jwt.MapClaims)
	if !ok {
		return 0, http.ErrNoCookie
	}

	emailVal, ok := claims["email"]
	email, okEmail := emailVal.(string)
	if !ok || !okEmail || strings.TrimSpace(email) == "" {
		return 0, http.ErrNoCookie
	}

	user, _, err := models.GetUserByEmail(strings.TrimSpace(email))
	if err != nil {
		return 0, err
	}

	return user.ID, nil
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 10
	}

	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	rows, err := config.DB.Query(`
		SELECT id, name, price, quantity
		FROM products
		WHERE deleted_at IS NULL
		ORDER BY id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Quantity); err != nil {
			http.Error(w, "Failed to read products", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read products", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": products,
		"meta": map[string]int{
			"current_page": page,
			"per_page":     limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := getUserIDFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = models.CreateProductWithAudit(&p, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)

}

func GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	product, err := models.GetByID(id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	productID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid product id", http.StatusBadRequest)
		return
	}

	var input struct {
		Quantity int    `json:"quantity"`
		Reason   string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	userID, err := getUserIDFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := models.UpdateStockWithAudit(productID, userID, input.Quantity, input.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Product updated successfully"})
}

func UpdateStockWithAudit(w http.ResponseWriter, r *http.Request) {
	UpdateProduct(w, r)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	productID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid product id", http.StatusBadRequest)
		return
	}

	userID, err := getUserIDFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = models.DeleteProductWithAudit(productID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	val, err := config.Redis.Get(ctx, models.DashboardCacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(val))
		return
	}

	stats, err := models.GetDashboardStats()
	if err != nil {
		http.Error(w, "Could not retrieve dashboard data", http.StatusInternalServerError)
		return
	}
	data, _ := json.Marshal(stats)
	config.Redis.Set(ctx, "dashboard_stats", data, 5*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(stats)
}

func SearchProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("search"))
	}
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("keyword"))
	}

	if query == "" {
		w.Header().Set("X-Cache", "BYPASS")
		GetProducts(w, r)
		return
	}

	cacheKey := "products_search:" + strings.ToLower(query)
	if cached, err := config.Redis.Get(ctx, cacheKey).Result(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cached))
		return
	}

	products, err := models.SearchProductsAdvanced(query)
	if err != nil {
		log.Printf("Advanced search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(products)
	if err != nil {
		http.Error(w, "Failed to encode search response", http.StatusInternalServerError)
		return
	}
	if err := config.Redis.Set(ctx, cacheKey, data, 2*time.Minute).Err(); err != nil {
		log.Printf("Search cache set failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func GetProductsPaginated(w http.ResponseWriter, r *http.Request) {
	lastIDStr := r.URL.Query().Get("last_id")
	limitStr := r.URL.Query().Get("limit")

	lastID, _ := strconv.Atoi(lastIDStr)
	limit, _ := strconv.Atoi(limitStr)

	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id,name,price,quantity
	FROM products
	WHERE id > $1 AND deleted_at IS NULL
	ORDER BY id ASC LIMIT $2
	`
	rows, err := config.DB.Query(query, lastID, limit)
	if err != nil {
		return
	}
	defer rows.Close()
	type Product struct {
		ID       int     `json:"id"`
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Quantity int     `json:"quantity"`
	}
	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Quantity)
		if err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}
	var nextID int
	if len(products) > 0 {
		nextID = products[len(products)-1].ID
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":        products,
		"next_cursor": nextID,
	})

}
