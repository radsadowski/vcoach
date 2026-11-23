package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

// LatestVO2Max returns the most recent VO2max estimate from synced activities.
func (r *Repository) LatestVO2Max(ctx context.Context, athleteID uuid.UUID) (*float64, error) {
	const q = `
		SELECT vo2max FROM activities
		WHERE athlete_id = $1 AND vo2max IS NOT NULL AND vo2max > 0
		ORDER BY start_time DESC LIMIT 1`

	var vo2max float64
	if err := r.db.GetContext(ctx, &vo2max, q, athleteID); err != nil {
		if isNoRowsErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("latest vo2max: %w", err)
	}
	return &vo2max, nil
}

// WeeklyRunVolume is distance run per ISO week.
type WeeklyRunVolume struct {
	Week         entities.Date `db:"week" json:"week"`
	DistanceKm   float64       `db:"distance_km" json:"distanceKm"`
	SessionCount int           `db:"session_count" json:"sessionCount"`
}

// WeeklyRunVolumes returns run volume per week for the last n weeks.
func (r *Repository) WeeklyRunVolumes(ctx context.Context, athleteID uuid.UUID, weeks int) ([]WeeklyRunVolume, error) {
	cutoff := entities.Date(time.Now().UTC().AddDate(0, 0, -(weeks * 7)))
	const q = `
		SELECT (date_trunc('week', (start_time AT TIME ZONE 'UTC')::date))::date AS week,
		       COALESCE(SUM(distance_m), 0) / 1000.0 AS distance_km,
		       COUNT(*) AS session_count
		FROM activities
		WHERE athlete_id = $1 AND sport_family = 'run'
		  AND (start_time AT TIME ZONE 'UTC')::date >= $2
		GROUP BY 1
		ORDER BY 1`

	var volumes []WeeklyRunVolume
	if err := r.db.SelectContext(ctx, &volumes, q, athleteID, cutoff); err != nil {
		return nil, fmt.Errorf("weekly run volumes: %w", err)
	}
	return volumes, nil
}

// LongestRunKm returns the longest run distance since the given date.
func (r *Repository) LongestRunKm(ctx context.Context, athleteID uuid.UUID, since entities.Date) (*float64, error) {
	const q = `
		SELECT MAX(distance_m) / 1000.0 FROM activities
		WHERE athlete_id = $1 AND sport_family = 'run'
		  AND (start_time AT TIME ZONE 'UTC')::date >= $2 AND distance_m > 0`

	var km *float64
	if err := r.db.GetContext(ctx, &km, q, athleteID, since); err != nil {
		return nil, fmt.Errorf("longest run: %w", err)
	}
	return km, nil
}

// LatestWeight returns the most recent weight from daily health.
func (r *Repository) LatestWeight(ctx context.Context, athleteID uuid.UUID) (*float64, error) {
	const q = `
		SELECT weight_kg FROM daily_health
		WHERE athlete_id = $1 AND weight_kg IS NOT NULL
		ORDER BY day DESC LIMIT 1`

	var weight float64
	if err := r.db.GetContext(ctx, &weight, q, athleteID); err != nil {
		if isNoRowsErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("latest weight: %w", err)
	}
	return &weight, nil
}

// WeightDaysAgo returns the latest weight recorded at least the given number
// of days ago, for trend deltas.
func (r *Repository) WeightDaysAgo(ctx context.Context, athleteID uuid.UUID, days int) (*float64, error) {
	const q = `
		SELECT weight_kg FROM daily_health
		WHERE athlete_id = $1 AND weight_kg IS NOT NULL AND day <= CURRENT_DATE - $2::int
		ORDER BY day DESC LIMIT 1`

	var weight float64
	if err := r.db.GetContext(ctx, &weight, q, athleteID, days); err != nil {
		if isNoRowsErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("weight days ago: %w", err)
	}
	return &weight, nil
}

func isNoRowsErr(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
