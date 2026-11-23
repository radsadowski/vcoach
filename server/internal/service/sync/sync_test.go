package sync_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/service/garmin"
	"github.com/radosny/vcoach/server/internal/service/garmin/mock"
	"github.com/radosny/vcoach/server/internal/service/plans"
	syncsvc "github.com/radosny/vcoach/server/internal/service/sync"
	"github.com/radosny/vcoach/server/internal/testing/database"
	planbuilder "github.com/radosny/vcoach/server/internal/testing/entities/plan"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestService(t *testing.T) {
	t.Parallel()
	r := requireNew(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	ctrl := gomock.NewController(t)
	client := mock.NewMockClient(ctrl)
	svc := syncsvc.New(repo, client, discardLogger())
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	expectHealthyEndpoints(client, activitiesFixture(), healthFixture())

	t.Run("syncs activities and daily health idempotently", func(t *testing.T) {
		r := requireNew(t)

		// when
		result, err := svc.SyncNow(ctx, athleteID)

		// then
		r.NoError(err)
		r.Equal(2, result.ActivitiesSynced)
		r.GreaterOrEqual(result.HealthDaysSynced, 1)

		activities, err := repo.ListActivitiesByFamily(ctx, athleteID, "run", entities.Date(mustParseDay("2000-01-01")), entities.Date(mustParseDay("2999-01-01")))
		r.NoError(err)
		r.Len(activities, 1)
		r.Equal("42", activities[0].ExternalID)
		r.Equal("run", activities[0].SportFamily)
		r.NotNil(activities[0].DistanceM)

		health, err := repo.ListDailyHealth(ctx, athleteID, entities.Date(mustParseDay("2000-01-01")), entities.Date(mustParseDay("2999-01-01")))
		r.NoError(err)
		r.GreaterOrEqual(len(health), 1)
		found := false
		for _, day := range health {
			if day.Day.String() == "2026-11-20" {
				found = true
				r.NotNil(day.RestingHR)
				r.NotNil(day.Steps)
				r.NotNil(day.HRVMS)
				r.NotNil(day.SleepHours)
				r.NotNil(day.BodyBattery)
				r.NotNil(day.WeightKg)
			}
		}
		r.True(found, "expected a daily health row for 2026-11-20")

		// when: sync the same window again
		result, err = svc.SyncNow(ctx, athleteID)

		// then: no duplicates
		r.NoError(err)
		activities, err = repo.ListActivitiesByFamily(ctx, athleteID, "run", entities.Date(mustParseDay("2000-01-01")), entities.Date(mustParseDay("2999-01-01")))
		r.NoError(err)
		r.Len(activities, 1)
		r.Equal(2, result.ActivitiesSynced)

		climbs, err := repo.ListActivitiesByFamily(ctx, athleteID, "climb", entities.Date(mustParseDay("2000-01-01")), entities.Date(mustParseDay("2999-01-01")))
		r.NoError(err)
		r.Len(climbs, 1)
		r.Equal("43", climbs[0].ExternalID)
	})

	t.Run("advances the cursor and records sync state", func(t *testing.T) {
		r := requireNew(t)

		// given / when
		_, err := svc.SyncNow(ctx, athleteID)

		// then
		r.NoError(err)
		activityState, err := repo.GetSyncState(ctx, athleteID, entities.SyncSourceGarminActivities)
		r.NoError(err)
		r.NotNil(activityState)
		r.NotNil(activityState.Cursor)
		r.Nil(activityState.LastError)

		healthState, err := repo.GetSyncState(ctx, athleteID, entities.SyncSourceGarminDailyHealth)
		r.NoError(err)
		r.NotNil(healthState)
		r.Nil(healthState.LastError)
	})

	t.Run("records per-source errors on unauthorized", func(t *testing.T) {
		r := requireNew(t)

		// given
		failing := mock.NewMockClient(gomock.NewController(t))
		failing.EXPECT().ListActivities(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &garmin.UnauthorizedError{}).AnyTimes()
		failing.EXPECT().DailySummaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().StressDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().HRVDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().SleepSummaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().BodyBatteryDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().WeightRange(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		failing.EXPECT().HeartRateZones(gomock.Any()).Return(garmin.HeartRateZones{}, nil).AnyTimes()
		failingSvc := syncsvc.New(repo, failing, discardLogger())

		// when
		result, err := failingSvc.SyncNow(ctx, athleteID)

		// then
		r.Error(err)
		r.NotEmpty(result.Errors)

		state, err := repo.GetSyncState(ctx, athleteID, entities.SyncSourceGarminActivities)
		r.NoError(err)
		r.NotNil(state)
		r.NotNil(state.LastError)
		r.Contains(*state.LastError, "unauthorized")
	})

	t.Run("matches synced activities to planned sessions", func(t *testing.T) {
		r := requireNew(t)

		// given: a plan with a boulder session today (matching the fixture
		// bouldering activity) and a run session far in the past (missed)
		planSvc := plans.New(repo)
		plan, err := planSvc.CreatePlan(ctx, athleteID, planbuilder.NewInput(
			planbuilder.WithWeeks([]entities.CreateWeekInput{
				{
					WeekNumber: 1,
					StartDate:  entities.Date(mustParseDay(todayStr())),
					Sessions: []entities.CreateSessionInput{
						{Sport: entities.SportBoulder, SessionType: "limit bouldering", PlannedDate: entities.Date(mustParseDay(todayStr()))},
						{Sport: entities.SportRun, SessionType: "easy", PlannedDate: entities.Date(mustParseDay("2000-01-01"))},
					},
				},
			}),
		))
		r.NoError(err)

		// when
		_, err = svc.SyncNow(ctx, athleteID)

		// then
		r.NoError(err)
		stored, err := planSvc.GetPlan(ctx, athleteID, plan.ID)
		r.NoError(err)
		sessionStates := map[string]entities.SessionStatus{}
		for _, session := range stored.Weeks[0].Sessions {
			sessionStates[session.Sport+"/"+session.SessionType] = session.Status
		}
		r.Equal(entities.SessionStatusCompleted, sessionStates["boulder/limit bouldering"])
		r.Equal(entities.SessionStatusMissed, sessionStates["run/easy"])
	})

	t.Run("status lists sync sources", func(t *testing.T) {
		r := requireNew(t)

		// given / when
		_, err := svc.SyncNow(ctx, athleteID)
		r.NoError(err)
		status, err := svc.Status(ctx, athleteID)

		// then
		r.NoError(err)
		r.Len(status.Sources, 2)
	})
}

func expectHealthyEndpoints(client *mock.MockClient, activities []garmin.Activity, daily map[string]any) {
	client.EXPECT().ListActivities(gomock.Any(), gomock.Any(), gomock.Any()).Return(activities, nil).AnyTimes()
	client.EXPECT().DailySummaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["summaries"], nil).AnyTimes()
	client.EXPECT().StressDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["stress"], nil).AnyTimes()
	client.EXPECT().HRVDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["hrv"], nil).AnyTimes()
	client.EXPECT().SleepSummaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["sleep"], nil).AnyTimes()
	client.EXPECT().BodyBatteryDaily(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["battery"], nil).AnyTimes()
	client.EXPECT().WeightRange(gomock.Any(), gomock.Any(), gomock.Any()).Return(daily["weights"], nil).AnyTimes()
	client.EXPECT().HeartRateZones(gomock.Any()).Return(garmin.HeartRateZones{MaxHeartRate: 190, RestingHR: 50, Zone2Floor: 120}, nil).AnyTimes()
}
