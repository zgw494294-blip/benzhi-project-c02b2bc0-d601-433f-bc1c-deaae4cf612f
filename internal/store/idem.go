package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

func readIdempotency(ctx context.Context, tx *sql.Tx, operation, key string) (json.RawMessage, bool, error) {
	var raw []byte
	err := tx.QueryRowContext(ctx, "SELECT response_json FROM idempotency_results WHERE operation=? AND idempotency_key=?", operation, key).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return json.RawMessage(raw), true, nil
}
func writeIdempotency(ctx context.Context, tx *sql.Tx, operation, key, caseID string, raw []byte, at time.Time) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO idempotency_results(operation,idempotency_key,case_id,response_json,created_at) VALUES(?,?,?,?,?)", operation, key, caseID, raw, at.Format(time.RFC3339Nano))
	return err
}
