package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const syncStateColumns = `source, athlete_id, last_synced_at, cursor, last_error, last_error_at`

// GetSyncState returns the sync state for one source, or nil when the source
// has never synced.
func (r *Repository) GetSyncState(ctx context.Context, athleteID uuid.UUID, source string) (*entities.SyncState, error) {
	const q = `SELECT ` + syncStateColumns + ` FROM sync_state WHERE athlete_id = $1 AND source = $2`

	var state entities.SyncState
	if err := r.db.GetContext(ctx, &state, q, athleteID, source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get sync state: %w", err)
	}
	return &state, nil
}

// ListSyncStates returns all sync states of the athlete.
func (r *Repository) ListSyncStates(ctx context.Context, athleteID uuid.UUID) ([]entities.SyncState, error) {
	const q = `SELECT ` + syncStateColumns + ` FROM sync_state WHERE athlete_id = $1`

	var states []entities.SyncState
	if err := r.db.SelectContext(ctx, &states, q, athleteID); err != nil {
		return nil, fmt.Errorf("list sync states: %w", err)
	}
	return states, nil
}

// SaveSyncStateSuccess records a successful sync and advances the cursor.
func (r *Repository) SaveSyncStateSuccess(ctx context.Context, athleteID uuid.UUID, source string, cursor entities.Date) error {
	const q = `
		INSERT INTO sync_state (source, athlete_id, last_synced_at, cursor, last_error, last_error_at)
		VALUES ($1, $2, now(), $3, NULL, NULL)
		ON CONFLICT (source, athlete_id) DO UPDATE SET
			last_synced_at = now(),
			cursor = EXCLUDED.cursor,
			last_error = NULL,
			last_error_at = NULL`

	if _, err := r.db.ExecContext(ctx, q, source, athleteID, cursor); err != nil {
		return fmt.Errorf("save sync state success: %w", err)
	}
	return nil
}

// SaveSyncStateError records a sync failure without touching the cursor.
func (r *Repository) SaveSyncStateError(ctx context.Context, athleteID uuid.UUID, source, message string) error {
	const q = `
		INSERT INTO sync_state (source, athlete_id, last_error, last_error_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (source, athlete_id) DO UPDATE SET
			last_error = EXCLUDED.last_error,
			last_error_at = EXCLUDED.last_error_at`

	if _, err := r.db.ExecContext(ctx, q, source, athleteID, message); err != nil {
		return fmt.Errorf("save sync state error: %w", err)
	}
	return nil
}
