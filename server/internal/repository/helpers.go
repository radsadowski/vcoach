package repository

import (
	"database/sql"
	"fmt"
)

// requireRowAffected verifies a write affected exactly one row, converting
// zero rows into the caller-friendly sentinel sql.ErrNoRows.
func requireRowAffected(res sql.Result, what string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for %s: %w", what, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
