package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// GetAppSetting reads a JSON app setting into out. Returns false when the
// key does not exist.
func (r *Repository) GetAppSetting(ctx context.Context, key string, out any) (bool, error) {
	const q = `SELECT value FROM app_settings WHERE key = $1`

	var raw []byte
	if err := r.db.GetContext(ctx, &raw, q, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("get app setting %s: %w", key, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, fmt.Errorf("decode app setting %s: %w", key, err)
	}
	return true, nil
}

// SetAppSetting upserts a JSON app setting.
func (r *Repository) SetAppSetting(ctx context.Context, key string, value any) error {
	const q = `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode app setting %s: %w", key, err)
	}
	if _, err := r.db.ExecContext(ctx, q, key, raw); err != nil {
		return fmt.Errorf("set app setting %s: %w", key, err)
	}
	return nil
}
