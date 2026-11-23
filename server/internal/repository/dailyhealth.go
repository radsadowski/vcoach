package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const dailyHealthColumns = `id, athlete_id, day, sleep_score, sleep_hours, hrv_ms, resting_hr, body_battery, stress_avg, steps, weight_kg, raw, created_at`

// UpsertDailyHealth inserts or updates one day of health metrics. Nil metrics
// keep their previously stored values, so partial fetches never erase data.
func (r *Repository) UpsertDailyHealth(ctx context.Context, health entities.DailyHealth) (entities.DailyHealth, error) {
	const q = `
		INSERT INTO daily_health (athlete_id, day, sleep_score, sleep_hours, hrv_ms, resting_hr, body_battery, stress_avg, steps, weight_kg, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (athlete_id, day) DO UPDATE SET
			sleep_score = COALESCE(EXCLUDED.sleep_score, daily_health.sleep_score),
			sleep_hours = COALESCE(EXCLUDED.sleep_hours, daily_health.sleep_hours),
			hrv_ms = COALESCE(EXCLUDED.hrv_ms, daily_health.hrv_ms),
			resting_hr = COALESCE(EXCLUDED.resting_hr, daily_health.resting_hr),
			body_battery = COALESCE(EXCLUDED.body_battery, daily_health.body_battery),
			stress_avg = COALESCE(EXCLUDED.stress_avg, daily_health.stress_avg),
			steps = COALESCE(EXCLUDED.steps, daily_health.steps),
			weight_kg = COALESCE(EXCLUDED.weight_kg, daily_health.weight_kg),
			raw = EXCLUDED.raw
		RETURNING ` + dailyHealthColumns

	var stored entities.DailyHealth
	err := r.db.GetContext(ctx, &stored, q,
		health.AthleteID, health.Day, health.SleepScore, health.SleepHours, health.HRVMS,
		health.RestingHR, health.BodyBattery, health.StressAvg, health.Steps, health.WeightKg, health.Raw,
	)
	if err != nil {
		return entities.DailyHealth{}, fmt.Errorf("upsert daily health: %w", err)
	}
	return stored, nil
}

// ListDailyHealth returns rows in the inclusive day range, oldest first.
func (r *Repository) ListDailyHealth(ctx context.Context, athleteID uuid.UUID, from, to entities.Date) ([]entities.DailyHealth, error) {
	const q = `
		SELECT ` + dailyHealthColumns + ` FROM daily_health
		WHERE athlete_id = $1 AND day BETWEEN $2 AND $3
		ORDER BY day`

	var rows []entities.DailyHealth
	if err := r.db.SelectContext(ctx, &rows, q, athleteID, from, to); err != nil {
		return nil, fmt.Errorf("list daily health: %w", err)
	}
	return rows, nil
}
