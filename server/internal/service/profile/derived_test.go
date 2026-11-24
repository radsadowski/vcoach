package profile_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
	"github.com/radosny/vcoach/server/internal/testing/database"
)

func TestGetDerived(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	svc := profilesvc.New(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	today := time.Now().UTC()
	runKm := 10.0
	distanceM := runKm * 1000
	vo2max := 51.2
	_, err = repo.UpsertActivity(ctx, entities.Activity{
		AthleteID:   athleteID,
		ExternalID:  "run-1",
		Sport:       "running",
		SportFamily: "run",
		StartTime:   today.AddDate(0, 0, -3),
		DurationSec: 2700,
		DistanceM:   &distanceM,
		VO2Max:      &vo2max,
		Raw:         []byte(`{}`),
	})
	r.NoError(err)

	weightNow, weightBefore := 72.4, 73.1
	_, err = repo.UpsertDailyHealth(ctx, entities.DailyHealth{
		AthleteID: athleteID,
		Day:       entities.Date(today),
		WeightKg:  &weightNow,
		Raw:       []byte(`{}`),
	})
	r.NoError(err)
	_, err = repo.UpsertDailyHealth(ctx, entities.DailyHealth{
		AthleteID: athleteID,
		Day:       entities.Date(today.AddDate(0, 0, -35)),
		WeightKg:  &weightBefore,
		Raw:       []byte(`{}`),
	})
	r.NoError(err)

	err = repo.SetAppSetting(ctx, "garmin_hr_zones", map[string]int{
		"maxHeartRateUsed": 190, "restingHeartRateUsed": 50,
		"zone1Floor": 100, "zone2Floor": 120, "zone3Floor": 145, "zone4Floor": 165,
	})
	r.NoError(err)

	// when
	derived, err := svc.GetDerived(ctx, athleteID)

	// then
	r.NoError(err)
	r.NotNil(derived.HeartRate)
	r.Equal(190, derived.HeartRate.MaxHeartRate)
	r.Equal(120, derived.HeartRate.Zone2Floor)

	r.NotNil(derived.VO2Max)
	r.InDelta(51.2, *derived.VO2Max, 0.01)

	r.NotNil(derived.LongestRunKm)
	r.InDelta(10.0, *derived.LongestRunKm, 0.01)

	r.NotEmpty(derived.WeeklyVolumes)
	r.InDelta(10.0, derived.WeeklyVolumes[len(derived.WeeklyVolumes)-1].DistanceKm, 0.01)

	r.NotNil(derived.Weight)
	r.InDelta(72.4, derived.Weight.LatestKg, 0.01)
	r.InDelta(73.1, derived.Weight.DaysAgoKg, 0.01)
	r.InDelta(-0.7, derived.Weight.ChangeKg, 0.01)
}
