package sync_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/service/garmin"
)

func requireNew(t *testing.T) *require.Assertions {
	return require.New(t)
}

func mustParseDay(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed.UTC()
}

func todayStr() string {
	return time.Now().UTC().Format("2006-01-02")
}

// activitiesFixture returns one run and one bouldering activity synced today.
func activitiesFixture() []garmin.Activity {
	return []garmin.Activity{
		{
			ActivityID:     42,
			ActivityName:   "Morning run",
			ActivityType:   garmin.ActivityTypeRef{TypeKey: "running"},
			StartTimeLocal: todayStr() + " 07:30:00.0",
			Duration:       2400,
			Distance:       8000,
			AverageHR:      150,
			MaxHR:          172,
			VO2Max:         51.2,
		},
		{
			ActivityID:     43,
			ActivityName:   "Board session",
			ActivityType:   garmin.ActivityTypeRef{TypeKey: "bouldering"},
			StartTimeLocal: todayStr() + " 18:00:00.0",
			Duration:       5400,
		},
	}
}

// healthFixture returns one day of health metrics for 2026-11-20.
func healthFixture() map[string]any {
	return map[string]any{
		"summaries": []garmin.DailySummary{
			{
				CalendarDate: "2026-11-20",
				HeartRate:    garmin.DailyHeartRate{RestingValue: 52},
				Movement:     garmin.DailyMovement{Steps: garmin.DailySteps{Value: 9800}},
			},
		},
		"stress": []garmin.DailyStress{
			{CalendarDate: "2026-11-20", AverageStressLevel: 28},
		},
		"hrv": []garmin.DailyHRV{
			{CalendarDate: "2026-11-20", LastNightAvg: 64},
		},
		"sleep": []garmin.SleepSummary{
			{CalendarDate: "2026-11-20", SleepTimeSeconds: 7.5 * 3600, SleepScores: garmin.SleepScoreBlocks{Overall: garmin.SleepScoreValue{Value: 82}}},
		},
		"battery": []garmin.DailyBodyBattery{
			{CalendarDate: "2026-11-20", Charged: 55},
		},
		"weights": []garmin.WeightMeasurement{
			{Date: "2026-11-20", Weight: 72.4},
		},
		"vo2max": []garmin.VO2MaxMeasurement{
			{Generic: garmin.VO2MaxEntry{CalendarDate: "2026-11-20", VO2MaxPreciseValue: 53.2}},
		},
	}
}
