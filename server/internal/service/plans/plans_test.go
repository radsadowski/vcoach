package plans_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/service/goals"
	"github.com/radosny/vcoach/server/internal/service/plans"
	"github.com/radosny/vcoach/server/internal/testing/database"
	goalbuilder "github.com/radosny/vcoach/server/internal/testing/entities/goal"
	planbuilder "github.com/radosny/vcoach/server/internal/testing/entities/plan"
)

func mustDate(value string) entities.Date {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return entities.Date(parsed)
}

func mustUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}

func createGoal(ctx context.Context, repo *repository.Repository, athleteID uuid.UUID) (entities.Goal, error) {
	goalSvc := goals.New(repo)
	return goalSvc.CreateGoal(ctx, athleteID, goalbuilder.NewInput(goalbuilder.WithRunDistance("21.1")))
}

func TestService(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	svc := plans.New(repo)
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	t.Run("creates a plan with weeks and sessions", func(t *testing.T) {
		r := require.New(t)

		// when
		plan, err := svc.CreatePlan(ctx, athleteID, planbuilder.NewInput(planbuilder.WithTitle("10K prep")))

		// then
		r.NoError(err)
		r.GreaterOrEqual(plan.Version, 1)
		r.Equal(entities.PlanStatusActive, plan.Status)
		r.Equal("10K prep", plan.Title)
		r.Len(plan.Weeks, 4)
		r.Equal(1, plan.Weeks[0].WeekNumber)
		r.Len(plan.Weeks[0].Sessions, 2)
		r.Equal(entities.SportRun, plan.Weeks[0].Sessions[0].Sport)
		r.Equal(entities.SessionStatusPlanned, plan.Weeks[0].Sessions[0].Status)
	})

	t.Run("creating a successor archives the active plan and bumps the version", func(t *testing.T) {
		r := require.New(t)

		// given
		first, err := svc.CreatePlan(ctx, athleteID, planbuilder.NewInput())
		r.NoError(err)

		// when
		second, err := svc.CreatePlan(ctx, athleteID, planbuilder.NewInput(planbuilder.WithTitle("regenerated")))
		r.NoError(err)

		// then
		r.Equal(first.Version+1, second.Version)
		r.Equal(entities.PlanStatusActive, second.Status)

		storedFirst, err := svc.GetPlan(ctx, athleteID, first.ID)
		r.NoError(err)
		r.Equal(entities.PlanStatusArchived, storedFirst.Status)
		r.Len(storedFirst.Weeks, 4)

		listed, err := svc.ListPlans(ctx, athleteID)
		r.NoError(err)
		r.Equal(second.Version, listed[0].Version)
		r.Equal(entities.PlanStatusActive, listed[0].Status)
	})

	t.Run("validates plan input", func(t *testing.T) {
		tests := map[string]struct {
			input   entities.CreatePlanInput
			wantErr string
		}{
			"no weeks": {
				input:   entities.CreatePlanInput{Title: "empty"},
				wantErr: "at least one week",
			},
			"duplicate week numbers": {
				input: entities.CreatePlanInput{
					Title: "dup weeks",
					Weeks: []entities.CreateWeekInput{
						{WeekNumber: 1, StartDate: mustDate("2026-01-05")},
						{WeekNumber: 1, StartDate: mustDate("2026-01-12")},
					},
				},
				wantErr: "more than once",
			},
			"invalid sport": {
				input: entities.CreatePlanInput{
					Title: "bad sport",
					Weeks: []entities.CreateWeekInput{{
						WeekNumber: 1,
						StartDate:  mustDate("2026-01-05"),
						Sessions: []entities.CreateSessionInput{{
							Sport:       "swimming",
							SessionType: "easy",
							PlannedDate: mustDate("2026-01-06"),
						}},
					}},
				},
				wantErr: "sport",
			},
			"missing session type": {
				input: entities.CreatePlanInput{
					Title: "no type",
					Weeks: []entities.CreateWeekInput{{
						WeekNumber: 1,
						StartDate:  mustDate("2026-01-05"),
						Sessions: []entities.CreateSessionInput{{
							Sport:       entities.SportRun,
							PlannedDate: mustDate("2026-01-06"),
						}},
					}},
				},
				wantErr: "sessionType",
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				r := require.New(t)

				// when
				_, err := svc.CreatePlan(ctx, athleteID, tt.input)

				// then
				var validationErr *domainerrors.ValidationError
				r.ErrorAs(err, &validationErr)
				r.Contains(err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("unknown goal reference is rejected", func(t *testing.T) {
		r := require.New(t)

		// given
		bogusGoalID := mustUUID("00000000-0000-0000-0000-000000000001")

		// when
		_, err := svc.CreatePlan(ctx, athleteID, planbuilder.NewInput(planbuilder.WithGoalID(bogusGoalID)))

		// then
		var notFoundErr *domainerrors.NotFoundError
		r.ErrorAs(err, &notFoundErr)
	})

	t.Run("archives the active plan", func(t *testing.T) {
		r := require.New(t)

		// given
		plan, err := svc.CreatePlan(ctx, athleteID, planbuilder.NewInput())
		r.NoError(err)

		// when
		err = svc.ArchivePlan(ctx, athleteID, plan.ID)

		// then
		r.NoError(err)
		stored, err := svc.GetPlan(ctx, athleteID, plan.ID)
		r.NoError(err)
		r.Equal(entities.PlanStatusArchived, stored.Status)
	})
}

func TestServiceTiesPlanToGoal(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	goalSvc := plans.New(repo)
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	goal, err := createGoal(ctx, repo, athleteID)
	r.NoError(err)

	// when
	plan, err := goalSvc.CreatePlan(ctx, athleteID, planbuilder.NewInput(planbuilder.WithGoalID(goal.ID)))

	// then
	r.NoError(err)
	r.NotNil(plan.GoalID)
	r.Equal(goal.ID, *plan.GoalID)
}
