package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const activityColumns = `id, athlete_id, external_id, sport, sport_family, start_time, duration_sec, distance_m, avg_hr, max_hr, vo2max, raw, created_at`

// UpsertActivity inserts or updates one synced activity, keyed on
// (athlete_id, external_id). Idempotent: re-syncing a window never duplicates.
func (r *Repository) UpsertActivity(ctx context.Context, activity entities.Activity) (entities.Activity, error) {
	const q = `
		INSERT INTO activities (athlete_id, external_id, sport, sport_family, start_time, duration_sec, distance_m, avg_hr, max_hr, vo2max, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (athlete_id, external_id) DO UPDATE SET
			sport = EXCLUDED.sport,
			sport_family = EXCLUDED.sport_family,
			start_time = EXCLUDED.start_time,
			duration_sec = EXCLUDED.duration_sec,
			distance_m = EXCLUDED.distance_m,
			avg_hr = EXCLUDED.avg_hr,
			max_hr = EXCLUDED.max_hr,
			vo2max = EXCLUDED.vo2max,
			raw = EXCLUDED.raw
		RETURNING ` + activityColumns

	var stored entities.Activity
	err := r.db.GetContext(ctx, &stored, q,
		activity.AthleteID, activity.ExternalID, activity.Sport, activity.SportFamily,
		activity.StartTime, activity.DurationSec, activity.DistanceM, activity.AvgHR,
		activity.MaxHR, activity.VO2Max, activity.Raw,
	)
	if err != nil {
		return entities.Activity{}, fmt.Errorf("upsert activity: %w", err)
	}
	return stored, nil
}

// ListActivities returns activities whose start date is in the inclusive
// range, newest first.
func (r *Repository) ListActivities(ctx context.Context, athleteID uuid.UUID, from, to entities.Date) ([]entities.Activity, error) {
	const q = `
		SELECT ` + activityColumns + ` FROM activities
		WHERE athlete_id = $1 AND (start_time AT TIME ZONE 'UTC')::date BETWEEN $2 AND $3
		ORDER BY start_time DESC`

	var activities []entities.Activity
	if err := r.db.SelectContext(ctx, &activities, q, athleteID, from, to); err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	return activities, nil
}

// ListActivitiesByFamily returns activities of one sport family in the
// inclusive date range, newest first.
func (r *Repository) ListActivitiesByFamily(ctx context.Context, athleteID uuid.UUID, family string, from, to entities.Date) ([]entities.Activity, error) {
	const q = `
		SELECT ` + activityColumns + ` FROM activities
		WHERE athlete_id = $1 AND sport_family = $2
		  AND (start_time AT TIME ZONE 'UTC')::date BETWEEN $3 AND $4
		ORDER BY start_time DESC`

	var activities []entities.Activity
	if err := r.db.SelectContext(ctx, &activities, q, athleteID, family, from, to); err != nil {
		return nil, fmt.Errorf("list activities by family: %w", err)
	}
	return activities, nil
}
