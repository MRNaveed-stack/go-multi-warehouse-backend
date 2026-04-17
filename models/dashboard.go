package models

import (
	"pureGo/config"
)

type WarehouseStock struct {
	WarehouseName string `json:"warehouse_name"`
	TotalCurrent  int    `json:"total_current"`
	TotalInitial  int    `json:"total_initial"`
}

type DashboardStat struct {
	TotalProducts     int              `json:"total_products"`
	GlobalStockCount  int              `json:"global_stock_count"`
	WarehouseBalances []WarehouseStock `json:"warehouse_balances"`
}

func GetMultiWarehouseStats() (DashboardStat, error) {
	var stats DashboardStat
	err := config.DB.QueryRow(
		`SELECT COUNT(DISTINCT product_id),
		SUM(current_quantity)
		FROM stock_batches`).Scan(&stats.TotalProducts, &stats.GlobalStockCount)

	if err != nil {
		return stats, err
	}
	rows, err := config.DB.Query(
		`
			SELECT w.name,SUM(sb.current_quantity),
			SUM(sb.initial_quantity)
			FROM warehouses w
			LEFT JOIN stock_batches sb on w.id = sb.warehouse_id
			GROUP BY w.name
			`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var ws WarehouseStock
		rows.Scan(&ws.WarehouseName, &ws.TotalCurrent, &ws.TotalInitial)
		stats.WarehouseBalances = append(stats.WarehouseBalances, ws)
	}
	return stats, nil
}
