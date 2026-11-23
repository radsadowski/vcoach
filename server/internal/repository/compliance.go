package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ApplyActivityMatches links planned (or previously missed) sessions to
// matching synced Garmin activities: same athlete, compatible sport family,
// activity start date within planned_date ±1 day, each activity used at most
// once. The closest activity to the planned day wins.
func (r *Repository) ApplyActivityMatches(ctx context.Context, athleteID uuid.UUID) (int64, error) {
	const q = `
	WITH candidates AS (
		SELECT ps.id AS session_id, a.id AS activity_id, a.start_time,
		       ROW_NUMBER() OVER (
		           PARTITION BY ps.id
		           ORDER BY ABS(EXTRACT(EPOCH FROM (a.start_time - (ps.planned_date + interval '12 hour'))))
		       ) AS rn
		FROM plan_sessions ps
		JOIN activities a
		  ON a.athlete_id = ps.athlete_id
		 AND a.sport_family = CASE ps.sport
		       WHEN 'run' THEN 'run'
		       WHEN 'boulder' THEN 'climb'
		       WHEN 'lead' THEN 'climb'
		     END
		 AND (a.start_time AT TIME ZONE 'UTC')::date BETWEEN ps.planned_date - 1 AND ps.planned_date + 1
		WHERE ps.athlete_id = $1
		  AND ps.status IN ('planned', 'missed')
		  AND a.id NOT IN (SELECT matched_activity_id FROM plan_sessions WHERE matched_activity_id IS NOT NULL)
	)
	UPDATE plan_sessions ps
	SET status = 'completed', matched_activity_id = c.activity_id, completed_at = c.start_time
	FROM candidates c
	WHERE ps.id = c.session_id AND c.rn = 1`

	res, err := r.db.ExecContext(ctx, q, athleteID)
	if err != nil {
		return 0, fmt.Errorf("apply activity matches: %w", err)
	}
	matched, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("apply activity matches rows: %w", err)
	}
	return matched, nil
}

// MarkMissedSessions flags planned sessions whose matching window has fully
// closed (planned_date older than yesterday) and that stayed unmatched.
func (r *Repository) MarkMissedSessions(ctx context.Context, athleteID uuid.UUID) (int64, error) {
	const q = `
		UPDATE plan_sessions SET status = 'missed'
		WHERE athlete_id = $1 AND status = 'planned' AND planned_date < CURRENT_DATE - 1`

	res, err := r.db.ExecContext(ctx, q, athleteID)
	if err != nil {
		return 0, fmt.Errorf("mark missed sessions: %w", err)
	}
	missed, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark missed sessions rows: %w", err)
	}
	return missed, nil
}
