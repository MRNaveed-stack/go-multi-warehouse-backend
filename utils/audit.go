package utils

import (
	"database/sql"
	"encoding/json"
)

func RecordAudit(tx *sql.Tx, userID int, action string, entity string, entityID int, oldVal, newVal interface{}) error {
	oldJSON, _ := json.Marshal(oldVal)
	newJSON, _ := json.Marshal(newVal)

	query := `INSERT INTO audit_logs (user_id, action , entity_name , entity_id , old_value , new_value )
    VALUES ($1,$2,$3,$4,$5,$6)  
   `
	_, err := tx.Exec(query, userID, action, entity, entityID, oldJSON, newJSON)
	return err
}
