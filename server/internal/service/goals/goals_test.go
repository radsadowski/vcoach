package goals_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/service/goals"
	"github.com/radosny/vcoach/server/internal/testing/database"
	goalbuilder "github.com/radosny/vcoach/server/internal/testing/entities/goal"
)

func randomUUID() uuid.UUID {
	return uuid.Must(uuid.NewRandom())
}

func TestService(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	svc := goals.New(repo)
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	t.Run("creates a valid boulder goal", func(t *testing.T) {
		r := require.New(t)

		// when
		goal, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(
			goalbuilder.WithBoulderGrade("7C"),
			goalbuilder.WithPrimary(),
			goalbuilder.WithNotes("project season"),
		))

		// then
		r.NoError(err)
		r.NotEqual(entities.Goal{}.ID, goal.ID)
		r.Equal(entities.GoalActivityBoulder, goal.ActivityType)
		r.Equal("7C", goal.TargetValue)
		r.True(goal.IsPrimary)
		r.Equal(entities.GoalStatusActive, goal.Status)
		r.Equal("project season", goal.Notes)
	})

	t.Run("validates catalog input", func(t *testing.T) {
		tests := map[string]struct {
			input   entities.CreateGoalInput
			wantErr string
		}{
			"invalid boulder grade": {
				input:   goalbuilder.NewInput(goalbuilder.WithBoulderGrade("7X")),
				wantErr: "Font boulder grade",
			},
			"run goal with grade metric": {
				input:   goalbuilder.NewInput(goalbuilder.WithRunDistance("10"), goalbuilder.WithTargetMetric(entities.TargetMetricGrade)),
				wantErr: "targetMetric",
			},
			"run goal with non-numeric distance": {
				input:   goalbuilder.NewInput(goalbuilder.WithRunDistance("ten")),
				wantErr: "distance in km",
			},
			"lead goal with boulder grade": {
				input:   goalbuilder.NewInput(goalbuilder.WithLeadGrade("7C")),
				wantErr: "Font lead grade",
			},
			"unknown activity type": {
				input:   goalbuilder.NewInput(goalbuilder.WithActivityType("swim")),
				wantErr: "activityType",
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				r := require.New(t)

				// when
				_, err := svc.CreateGoal(ctx, athleteID, tt.input)

				// then
				r.Error(err)
				var validationErr *domainerrors.ValidationError
				r.ErrorAs(err, &validationErr)
				r.Contains(err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("creating a new primary goal clears the old primary", func(t *testing.T) {
		r := require.New(t)

		// given
		first, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(
			goalbuilder.WithRunDistance("10"), goalbuilder.WithPrimary()))
		r.NoError(err)
		second, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(
			goalbuilder.WithBoulderGrade("7A"), goalbuilder.WithPrimary()))
		r.NoError(err)

		// when
		storedFirst, err := svc.GetGoal(ctx, athleteID, first.ID)
		r.NoError(err)
		storedSecond, err := svc.GetGoal(ctx, athleteID, second.ID)
		r.NoError(err)

		// then
		r.False(storedFirst.IsPrimary)
		r.True(storedSecond.IsPrimary)
	})

	t.Run("updating primary switches the primary flag", func(t *testing.T) {
		r := require.New(t)

		// given
		first, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(
			goalbuilder.WithRunDistance("10"), goalbuilder.WithPrimary()))
		r.NoError(err)
		second, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(
			goalbuilder.WithBoulderGrade("7A"), goalbuilder.WithPrimary()))
		r.NoError(err)

		// when
		primary := true
		_, err = svc.UpdateGoal(ctx, athleteID, first.ID, entities.UpdateGoalInput{IsPrimary: &primary})
		r.NoError(err)

		// then
		storedFirst, err := svc.GetGoal(ctx, athleteID, first.ID)
		r.NoError(err)
		storedSecond, err := svc.GetGoal(ctx, athleteID, second.ID)
		r.NoError(err)
		r.True(storedFirst.IsPrimary)
		r.False(storedSecond.IsPrimary)
	})

	t.Run("updates status and rejects invalid status", func(t *testing.T) {
		r := require.New(t)

		// given
		goal, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput())
		r.NoError(err)

		// when
		status := entities.GoalStatusAchieved
		updated, err := svc.UpdateGoal(ctx, athleteID, goal.ID, entities.UpdateGoalInput{Status: &status})

		// then
		r.NoError(err)
		r.Equal(entities.GoalStatusAchieved, updated.Status)

		// when
		badStatus := entities.GoalStatus("nope")
		_, err = svc.UpdateGoal(ctx, athleteID, goal.ID, entities.UpdateGoalInput{Status: &badStatus})

		// then
		var validationErr *domainerrors.ValidationError
		r.ErrorAs(err, &validationErr)
	})

	t.Run("unknown goal returns NotFoundError", func(t *testing.T) {
		r := require.New(t)

		// when
		_, err := svc.GetGoal(ctx, athleteID, randomUUID())

		// then
		var notFoundErr *domainerrors.NotFoundError
		r.ErrorAs(err, &notFoundErr)
	})

	t.Run("deletes a goal", func(t *testing.T) {
		r := require.New(t)

		// given
		goal, err := svc.CreateGoal(ctx, athleteID, goalbuilder.NewInput())
		r.NoError(err)

		// when
		err = svc.DeleteGoal(ctx, athleteID, goal.ID)

		// then
		r.NoError(err)
		var notFoundErr *domainerrors.NotFoundError
		_, err = svc.GetGoal(ctx, athleteID, goal.ID)
		r.ErrorAs(err, &notFoundErr)
	})
}
