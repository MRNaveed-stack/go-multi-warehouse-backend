package models

import (
	"database/sql"
	"pureGo/config"
	"pureGo/utils"
)

func TransferStock(productID, fromWH, toWH, qty, userID int, reason string) (err error) {
	tx, err := config.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = DeductFromWarehouseFIFO(tx, productID, fromWH, qty)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
        INSERT INTO stock_batches (
            product_id,
            warehouse_id,
            batch_number,
            initial_quantity,
            current_quantity,
            expiry_date
        )
        VALUES ($1, $2, $3, $4, $5, NOW() + interval '6 months')`,
		productID, toWH, "INTERNAL-TRANSFER", qty, qty)
	if err != nil {
		return err
	}

	err = utils.RecordAudit(tx, userID, "TRANSFER", "warehouses", fromWH,
		map[string]interface{}{"from_warehouse": fromWH, "qty": qty, "reason": reason},
		map[string]interface{}{"to_warehouse": toWH, "qty": qty, "reason": reason})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func DeductFromWarehouseFIFO(tx *sql.Tx, productID, warehouseID, qty int) error {
	rows, err := tx.Query(`
		SELECT id, current_quantity FROM stock_batches
		WHERE product_id = $1 AND warehouse_id = $2 AND current_quantity > 0
		ORDER BY expiry_date ASC`, productID, warehouseID)
	if err != nil {
		return err
	}

	type fifoBatch struct {
		id  int
		qty int
	}
	batches := make([]fifoBatch, 0)

	for rows.Next() {
		var b fifoBatch
		if err := rows.Scan(&b.id, &b.qty); err != nil {
			_ = rows.Close()
			return err
		}
		batches = append(batches, b)
	}

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	remaining := qty
	for _, b := range batches {
		if remaining <= 0 {
			break
		}

		toTake := b.qty
		if remaining < b.qty {
			toTake = remaining
		}

		if _, err := tx.Exec("UPDATE stock_batches SET current_quantity = current_quantity - $1 WHERE id = $2", toTake, b.id); err != nil {
			return err
		}
		remaining -= toTake
	}
	if remaining > 0 {
		return sql.ErrNoRows
	}

	return nil
}

func UpdateStockWithWebhook(productID, amount, userID int) error {
	err := UpdateStockWithAudit(productID, userID, amount, "stock update")
	if err != nil {
		return err
	}

	var currentQty int
	err = config.DB.QueryRow("SELECT quantity FROM products WHERE id = $1", productID).Scan(&currentQty)
	if err != nil {
		return err
	}

	if currentQty < 10 {
		utils.DispatchWebHook("stock.low", map[string]interface{}{
			"product_id":  productID,
			"current_qty": currentQty,
			"message":     "Warning: Stock is running low!",
		})
	}
	return nil
}
