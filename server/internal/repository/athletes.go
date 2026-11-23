package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const athleteColumns = `id, name, is_default, birth_year, sex, weight_kg, running_experience, climbing_experience, created_at`

// GetOrCreateDefaultAthlete returns the single default athlete's ID, creating
// it on first use. This is the personal-first seam: the server layer resolves
// every request to this athlete today; real auth later swaps the resolution,
// not the queries.
func (r *Repository) GetOrCreateDefaultAthlete(ctx context.Context) (uuid.UUID, error) {
	const q = `
		WITH ins AS (
			INSERT INTO athletes (name, is_default) VALUES ('Athlete', true)
			ON CONFLICT DO NOTHING
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM athletes WHERE is_default
		LIMIT 1`

	var athleteID uuid.UUID
	if err := r.db.GetContext(ctx, &athleteID, q); err != nil {
		return uuid.Nil, err
	}
	return athleteID, nil
}

// GetAthlete returns the athlete with the given ID.
func (r *Repository) GetAthlete(ctx context.Context, athleteID uuid.UUID) (entities.Athlete, error) {
	const q = `SELECT ` + athleteColumns + ` FROM athletes WHERE id = $1`

	var athlete entities.Athlete
	if err := r.db.GetContext(ctx, &athlete, q, athleteID); err != nil {
		return entities.Athlete{}, err
	}
	return athlete, nil
}

// UpdateAthleteBasics updates manual profile basics. Nil fields are kept.
func (r *Repository) UpdateAthleteBasics(ctx context.Context, athleteID uuid.UUID, input entities.UpdateAthleteInput) (entities.Athlete, error) {
	const q = `
		UPDATE athletes SET
			birth_year = COALESCE($2, birth_year),
			sex = COALESCE($3, sex),
			weight_kg = COALESCE($4, weight_kg),
			running_experience = COALESCE($5, running_experience),
			climbing_experience = COALESCE($6, climbing_experience)
		WHERE id = $1
		RETURNING ` + athleteColumns

	var athlete entities.Athlete
	err := r.db.GetContext(ctx, &athlete, q,
		athleteID,
		input.BirthYear,
		input.Sex,
		input.WeightKg,
		input.RunningExperience,
		input.ClimbingExperience,
	)
	if err != nil {
		return entities.Athlete{}, err
	}
	return athlete, nil
}
