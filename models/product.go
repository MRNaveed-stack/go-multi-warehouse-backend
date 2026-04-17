package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"pureGo/config"
	"pureGo/utils"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}
type StockLog struct {
	ProductName  string `json:"product_name"`
	ChangeAmount int    `json:"change_amount"`
	UserEmail    string `json:"user_email"`
	CreatedAt    string `json:"created_at"`
}

type DashboardStats struct {
	TotalProducts int        `json:"total_products"`
	TotalValue    float64    `json:"total_value"`
	RecentLogs    []StockLog `json:"recent_logs"`
}

const DashboardCacheKey = "dashboard_stats"

func GetDashboardStats() (DashboardStats, error) {
	var stats DashboardStats

	summaryQuery := `SELECT COUNT(*), COALESCE(SUM(price * quantity), 0) FROM products WHERE deleted_at IS NULL`
	err := config.DB.QueryRow(summaryQuery).Scan(&stats.TotalProducts, &stats.TotalValue)
	if err != nil {
		return stats, fmt.Errorf("error fetching summary stats: %v", err)
	}

	logsQuery := `
		SELECT p.name, l.change_amount, u.email, l.created_at 
		FROM stock_logs l
		JOIN products p ON l.product_id = p.id
		JOIN users u ON l.user_id = u.id
		ORDER BY l.created_at DESC LIMIT 10`

	rows, err := config.DB.Query(logsQuery)
	if err != nil {
		return stats, fmt.Errorf("error fetching recent logs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var logEntry StockLog

		err := rows.Scan(
			&logEntry.ProductName,
			&logEntry.ChangeAmount,
			&logEntry.UserEmail,
			&logEntry.CreatedAt,
		)
		if err != nil {
			return stats, fmt.Errorf("error scanning log row: %v", err)
		}

		stats.RecentLogs = append(stats.RecentLogs, logEntry)
	}

	if err = rows.Err(); err != nil {
		return stats, fmt.Errorf("error during rows iteration: %v", err)
	}
	return stats, nil
}
func InvalidateDashboardCache() {
	go func() {
		ctx := context.Background()
		err := config.Redis.Del(ctx, DashboardCacheKey).Err()
		if err != nil {
			log.Printf("failed to invalidate cache: %v", err)
		} else {
			log.Printf("Cache Purged: %s removed due to data update", DashboardCacheKey)
		}
	}()
}

func InvalidateSearchCache() {
	go func() {
		ctx := context.Background()
		iter := config.Redis.Scan(ctx, 0, "products_search:*", 100).Iterator()
		for iter.Next(ctx) {
			if err := config.Redis.Del(ctx, iter.Val()).Err(); err != nil {
				log.Printf("failed to delete search cache key %s: %v", iter.Val(), err)
			}
		}
		if err := iter.Err(); err != nil {
			log.Printf("failed to scan search cache keys: %v", err)
		}
	}()
}

func CreateProductWithAudit(p *Product, userID int) error {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO products (name, price, quantity) VALUES ($1, $2, $3) RETURNING id`
	err = tx.QueryRow(query, p.Name, p.Price, p.Quantity).Scan(&p.ID)
	if err != nil {
		return err
	}
	err = utils.RecordAudit(tx, userID, "CREATE", "products", p.ID, nil, p)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetByID(id string) (*Product, error) {
	p := &Product{}
	query := `SELECT id, name , price , quantity FROM products where id = $1`
	err := config.DB.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Price, &p.Quantity)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func UpdateStockWithAuditLegacy(productID int, userID int, amount int) error {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldProduct Product
	err = tx.QueryRow("SELECT id, name, price, quantity FROM products WHERE id = $1", productID).
		Scan(&oldProduct.ID, &oldProduct.Name, &oldProduct.Price, &oldProduct.Quantity)
	if err != nil {
		return err
	}

	var newQty int
	err = tx.QueryRow("UPDATE products SET quantity = quantity + $1 WHERE id = $2 RETURNING quantity",
		amount, productID).Scan(&newQty)
	if err != nil {
		return err
	}

	newProduct := oldProduct
	newProduct.Quantity = newQty

	err = utils.RecordAudit(tx, userID, "UPDATE_STOCK", "products", productID, oldProduct, newProduct)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteByID(id string) error {

	query := `DELETE FROM products where id =$1`
	result, err := config.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no product found with id %s", id)

	}

	InvalidateDashboardCache()
	InvalidateSearchCache()
	return nil
}

func DeleteProductWithAudit(productID int, userID int) error {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	oldProduct, err := GetByIDRaw(tx, productID)
	if err != nil {
		return fmt.Errorf("soft delete failed: product not found: %v", err)
	}

	updateQuery := `UPDATE products SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING id`
	var returnedID int
	err = tx.QueryRow(updateQuery, productID).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("soft delete failed: %v", err)
	}

	err = utils.RecordAudit(tx, userID, "DELETE", "products", productID, oldProduct, nil)
	if err != nil {
		return fmt.Errorf("audit log failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	InvalidateDashboardCache()
	InvalidateSearchCache()
	return nil
}

func UpdateStock(productID int, userID int, amount int, reason string) error {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	var currentQuantity, minLevel int
	var name string
	query := `UPDATE products SET quantity = quantity + $1
	WHERE id = $2 RETURNING name , quantity ,min_stock_level
	`
	err = tx.QueryRow(query, amount, productID).Scan(&name, &currentQuantity, &minLevel)
	if err != nil {
		tx.Rollback()
		return err
	}
	logQuery := `INSERT INTO stock_logs (product_id , user_id , change_amount, reason)
    VALUES ($1, $2, $3, $4)  
   `
	_, err = tx.Exec(logQuery, productID, userID, amount, reason)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	if currentQuantity <= minLevel {
		payload := utils.LowStockPayLoad{
			ProductName:     name,
			CurrentQuantity: currentQuantity,
		}
		utils.JobQueue <- utils.Job{
			Type: "LOW_STOCK_EMAIL",
			Data: payload,
		}
		InvalidateDashboardCache()
		InvalidateSearchCache()
		go func() {
			utils.SendLowStockAlert(name, currentQuantity)
		}()
	}
	return nil
}

func UpdateStockSecure(productID int, amount int, currentVersion int) error {
	query := `UPDATE products SET quantity = quantity + $1, 
	version = version + $1
    WHERE id = $2 AND version = $3
	`

	result, err := config.DB.Exec(query, amount, productID, currentVersion)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("edit conflict: someone else updated this product. Please refresh")
	}
	return nil
}

func SearchProductsAdvanced(term string) ([]Product, error) {
	query := `
		SELECT id, name, price, quantity 
		FROM products 
		WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(to_tsvector('english', name), plainto_tsquery('english', $1)) DESC`
	rows, err := config.DB.Query(query, term)
	if err != nil {
		return nil, fmt.Errorf("error searching products: %w", err)
	}
	defer rows.Close()
	var product []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Quantity)
		if err != nil {
			return nil, err
		}
		product = append(product, p)
	}

	return product, nil
}

func SoftDeleteProduct(id int) error {
	query := `UPDATE products SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := config.DB.Exec(query, id)
	return err
}

func GetByIDRaw(tx *sql.Tx, id int) (*Product, error) {
	var p Product
	query := `SELECT id, name, price,quantity FROM products WHERE id = $1 AND deleted_at IS NULL`
	err := tx.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Price, &p.Quantity)
	return &p, err
}

func UpdateStockWithAudit(productID int, userID int, amount int, reason string) error {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	oldProduct, err := GetByIDRaw(tx, productID)
	if err != nil {
		return fmt.Errorf("audit failed: could not find original product: %v", err)
	}
	var newQty int
	updateQuery := `UPDATE products SET quantity = quantity + $1 WHERE id = $2 RETURNING quantity`
	err = tx.QueryRow(updateQuery, amount, productID).Scan(&newQty)
	if err != nil {
		return err
	}
	newProduct := *oldProduct
	newProduct.Quantity = newQty
	err = utils.RecordAudit(tx, userID, "UPDATE_STOCK", "products", productID, oldProduct, newProduct)
	if err != nil {
		return fmt.Errorf("audit log failed: %v", err)
	}
	return tx.Commit()
}
