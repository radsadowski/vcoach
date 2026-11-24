// Package sync pulls the athlete's data from Garmin into the local database:
// activities, daily health and heart-rate zones. Syncs are idempotent and
// re-fetch a full backfill window every run, so a previously partial sync
// self-heals; failures are recorded per source and degrade gracefully.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/service/garmin"
)

// BackfillDays is how far back every sync reaches. Garmin caps the range of
// some daily-health endpoints, so windows are chunked per endpoint below.
const BackfillDays = 90

// Endpoint range caps, discovered against the live Personal API: sleep and
// body battery reject ranges over 31 days, stress over 14 days. The other
// daily-health endpoints accept the full backfill window in one request.
const (
	sleepRangeDays   = 30
	batteryRangeDays = 30
	stressRangeDays  = 14
)

// Result summarizes one sync run.
type Result struct {
	ActivitiesSynced int      `json:"activitiesSynced"`
	HealthDaysSynced int      `json:"healthDaysSynced"`
	Errors           []string `json:"errors,omitempty"`
}

// Service orchestrates Garmin syncs.
type Service struct {
	repo   *repository.Repository
	client garmin.Client
	log    *slog.Logger
}

// New builds a sync Service.
func New(repo *repository.Repository, client garmin.Client, log *slog.Logger) *Service {
	return &Service{repo: repo, client: client, log: log}
}

// SyncNow runs both sync sources for the athlete and applies compliance
// matching afterwards. Per-source errors are recorded and returned; one
// failing source never blocks the other.
func (s *Service) SyncNow(ctx context.Context, athleteID uuid.UUID) (Result, error) {
	result := Result{}

	activitiesSynced, activitiesErr := s.syncActivities(ctx, athleteID)
	result.ActivitiesSynced = activitiesSynced
	if activitiesErr != nil {
		result.Errors = append(result.Errors, activitiesErr.Error())
	}

	healthDays, healthErr := s.syncDailyHealth(ctx, athleteID)
	result.HealthDaysSynced = healthDays
	if healthErr != nil {
		result.Errors = append(result.Errors, healthErr.Error())
	}

	// Compliance matching runs whenever any data arrived: it links newly
	// synced activities to planned sessions and flags closed-window misses.
	var matched, missed int64
	err := s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		var err error
		matched, err = tx.ApplyActivityMatches(ctx, athleteID)
		if err != nil {
			return err
		}
		missed, err = tx.MarkMissedSessions(ctx, athleteID)
		return err
	})
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else if matched > 0 || missed > 0 {
		s.log.Info("compliance applied", "matched", matched, "missed", missed)
	}

	if len(result.Errors) > 0 {
		return result, errors.New("sync completed with errors")
	}
	return result, nil
}

// backfillWindow returns the inclusive date range every sync fetches.
func backfillWindow() (entities.Date, entities.Date) {
	today := entities.Date(time.Now().UTC())
	return today.AddDays(-(BackfillDays - 1)), today
}

// chunkRanges splits [from, to] into inclusive windows of at most chunkDays.
func chunkRanges(from, to entities.Date, chunkDays int) [][2]entities.Date {
	var ranges [][2]entities.Date
	for start := from; !start.After(to); {
		end := start.AddDays(chunkDays - 1)
		if end.After(to) {
			end = to
		}
		ranges = append(ranges, [2]entities.Date{start, end})
		start = end.AddDays(1)
	}
	return ranges
}

// syncActivities pulls the full backfill window of activities and upserts
// them (idempotent by Garmin activity ID). The window is deliberately not
// cursor-limited: re-fetching is cheap and self-heals earlier partial
// syncs; the stored cursor is informational.
func (s *Service) syncActivities(ctx context.Context, athleteID uuid.UUID) (int, error) {
	from, today := backfillWindow()

	activities, err := s.client.ListActivities(ctx, from.String(), today.String())
	if err != nil {
		s.recordSyncError(ctx, athleteID, entities.SyncSourceGarminActivities, err)
		return 0, err
	}

	for _, activity := range activities {
		normalized, err := normalizeActivity(activity, athleteID)
		if err != nil {
			s.log.Warn("skipping unparseable activity", "activityId", activity.ActivityID, "error", err)
			continue
		}
		if _, err := s.repo.UpsertActivity(ctx, *normalized); err != nil {
			s.recordSyncError(ctx, athleteID, entities.SyncSourceGarminActivities, err)
			return 0, err
		}
	}

	if err := s.repo.SaveSyncStateSuccess(ctx, athleteID, entities.SyncSourceGarminActivities, today); err != nil {
		return len(activities), fmt.Errorf("save activities sync state: %w", err)
	}
	return len(activities), nil
}

// syncDailyHealth merges the daily health endpoints into one row per day
// across the full backfill window. Endpoints that cap the request range are
// fetched in chunks; entries without a calendar date are skipped. Nil
// metrics keep their stored values on conflict, so partial fetches never
// erase data.
func (s *Service) syncDailyHealth(ctx context.Context, athleteID uuid.UUID) (int, error) {
	from, today := backfillWindow()

	byDay := map[entities.Date]*entities.DailyHealth{}
	rawByDay := map[entities.Date]map[string]json.RawMessage{}

	// add returns the row for a day, or nil when the entry carries no date
	// (Garmin occasionally emits undated junk entries).
	add := func(day string) *entities.DailyHealth {
		if day == "" {
			return nil
		}
		parsed, err := time.Parse("2006-01-02", day)
		if err != nil {
			s.log.Warn("skipping entry with unparseable date", "date", day)
			return nil
		}
		date := entities.Date(parsed)
		if _, ok := byDay[date]; !ok {
			byDay[date] = &entities.DailyHealth{AthleteID: athleteID, Day: date}
			rawByDay[date] = map[string]json.RawMessage{}
		}
		return byDay[date]
	}

	// fetchChunked calls a per-day-list endpoint for each chunk of the
	// backfill window.
	fetchChunked := func(chunkDays int, fetch func(ctx context.Context, startDate, endDate string) error) {
		for _, window := range chunkRanges(from, today, chunkDays) {
			if err := fetch(ctx, window[0].String(), window[1].String()); err != nil {
				s.log.Warn("daily health chunk unavailable", "from", window[0].String(), "to", window[1].String(), "error", err)
			}
		}
	}

	fetchChunked(BackfillDays, func(ctx context.Context, startDate, endDate string) error {
		summaries, err := s.client.DailySummaries(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, summary := range summaries {
			day := add(summary.CalendarDate)
			if day == nil {
				continue
			}
			if summary.Movement.Steps.Value > 0 {
				steps := int(summary.Movement.Steps.Value)
				day.Steps = &steps
			}
			if summary.HeartRate.RestingValue > 0 {
				resting := int(summary.HeartRate.RestingValue)
				day.RestingHR = &resting
			}
			rawByDay[day.Day]["daily_summary"] = mustMarshal(summary)
		}
		return nil
	})

	fetchChunked(stressRangeDays, func(ctx context.Context, startDate, endDate string) error {
		stress, err := s.client.StressDaily(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, entry := range stress {
			if entry.AverageStressLevel <= 0 {
				continue
			}
			day := add(entry.CalendarDate)
			if day == nil {
				continue
			}
			avg := entry.AverageStressLevel
			day.StressAvg = &avg
			rawByDay[day.Day]["stress"] = mustMarshal(entry)
		}
		return nil
	})

	fetchChunked(BackfillDays, func(ctx context.Context, startDate, endDate string) error {
		hrv, err := s.client.HRVDaily(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, entry := range hrv {
			if entry.LastNightAvg <= 0 {
				continue
			}
			day := add(entry.CalendarDate)
			if day == nil {
				continue
			}
			hrvAvg := entry.LastNightAvg
			day.HRVMS = &hrvAvg
			rawByDay[day.Day]["hrv"] = mustMarshal(entry)
		}
		return nil
	})

	fetchChunked(sleepRangeDays, func(ctx context.Context, startDate, endDate string) error {
		sleep, err := s.client.SleepSummaries(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, entry := range sleep {
			day := add(entry.CalendarDate)
			if day == nil {
				continue
			}
			if entry.SleepTimeSeconds > 0 {
				hours := entry.SleepTimeSeconds / 3600
				day.SleepHours = &hours
			}
			if entry.SleepScores.Overall.Value > 0 {
				score := entry.SleepScores.Overall.Value
				day.SleepScore = &score
			}
			rawByDay[day.Day]["sleep"] = mustMarshal(entry)
		}
		return nil
	})

	fetchChunked(batteryRangeDays, func(ctx context.Context, startDate, endDate string) error {
		battery, err := s.client.BodyBatteryDaily(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, entry := range battery {
			if entry.Charged <= 0 {
				continue
			}
			day := add(entry.CalendarDate)
			if day == nil {
				continue
			}
			charged := entry.Charged
			day.BodyBattery = &charged
			rawByDay[day.Day]["body_battery"] = mustMarshal(entry)
		}
		return nil
	})

	fetchChunked(BackfillDays, func(ctx context.Context, startDate, endDate string) error {
		weights, err := s.client.WeightRange(ctx, startDate, endDate)
		if err != nil {
			return err
		}
		for _, measurement := range weights {
			if measurement.Weight <= 0 {
				continue
			}
			day := add(measurement.Date)
			if day == nil {
				continue
			}
			weight := measurement.Weight
			day.WeightKg = &weight
			rawByDay[day.Day]["weight"] = mustMarshal(measurement)
		}
		return nil
	})

	for date, health := range byDay {
		health.Raw = mustMarshal(rawByDay[date])
		if _, err := s.repo.UpsertDailyHealth(ctx, *health); err != nil {
			s.recordSyncError(ctx, athleteID, entities.SyncSourceGarminDailyHealth, err)
			return 0, err
		}
	}

	// Heart-rate zones change rarely; refresh them opportunistically.
	if zones, err := s.client.HeartRateZones(ctx); err == nil {
		if err := s.repo.SetAppSetting(ctx, appSettingHRZones, zones); err != nil {
			s.log.Warn("store heart rate zones", "error", err)
		}
	} else {
		s.log.Warn("heart rate zones unavailable", "error", err)
	}

	// VO2max comes from the metrics endpoint (activity summaries usually omit
	// it); keep the latest measurement for the derived profile.
	s.refreshVO2Max(ctx, athleteID, from, today)

	if err := s.repo.SaveSyncStateSuccess(ctx, athleteID, entities.SyncSourceGarminDailyHealth, today); err != nil {
		return len(byDay), fmt.Errorf("save health sync state: %w", err)
	}
	return len(byDay), nil
}

// appSettingHRZones is the app_settings key for the synced Garmin zones.
const appSettingHRZones = "garmin_hr_zones"

// appSettingVO2Max is the app_settings key for the latest VO2max estimate.
const appSettingVO2Max = "garmin_vo2max"

// VO2MaxSnapshot is the stored latest VO2max measurement.
type VO2MaxSnapshot struct {
	Value float64       `json:"value"`
	Date  entities.Date `json:"date"`
}

// refreshVO2Max fetches the VO2max history over the window and stores the
// most recent estimate. The maxmet endpoint caps its range like sleep does,
// so it is fetched in chunks.
func (s *Service) refreshVO2Max(ctx context.Context, athleteID uuid.UUID, from, to entities.Date) {
	var latest *VO2MaxSnapshot
	for _, window := range chunkRanges(from, to, sleepRangeDays) {
		measurements, err := s.client.VO2MaxDaily(ctx, window[0].String(), window[1].String())
		if err != nil {
			s.log.Warn("vo2max unavailable", "from", window[0].String(), "to", window[1].String(), "error", err)
			continue
		}
		for _, measurement := range measurements {
			if measurement.Generic.VO2MaxPreciseValue <= 0 || measurement.Generic.CalendarDate == "" {
				continue
			}
			parsed, err := time.Parse("2006-01-02", measurement.Generic.CalendarDate)
			if err != nil {
				continue
			}
			snapshot := &VO2MaxSnapshot{Value: measurement.Generic.VO2MaxPreciseValue, Date: entities.Date(parsed)}
			if latest == nil || snapshot.Date.After(latest.Date) {
				latest = snapshot
			}
		}
	}
	if latest != nil {
		if err := s.repo.SetAppSetting(ctx, appSettingVO2Max, latest); err != nil {
			s.log.Warn("store vo2max", "error", err)
		}
	}
}

// SyncStatus reports the per-source sync state for the status endpoint.
type SyncStatus struct {
	Sources []entities.SyncState `json:"sources"`
}

// Status returns the athlete's sync states.
func (s *Service) Status(ctx context.Context, athleteID uuid.UUID) (SyncStatus, error) {
	states, err := s.repo.ListSyncStates(ctx, athleteID)
	if err != nil {
		return SyncStatus{}, err
	}
	if states == nil {
		states = []entities.SyncState{}
	}
	return SyncStatus{Sources: states}, nil
}

func (s *Service) recordSyncError(ctx context.Context, athleteID uuid.UUID, source string, syncErr error) {
	s.log.Error("sync failed", "source", source, "error", syncErr)
	if err := s.repo.SaveSyncStateError(ctx, athleteID, source, syncErr.Error()); err != nil {
		s.log.Error("record sync error", "source", source, "error", err)
	}
}

func normalizeActivity(activity garmin.Activity, athleteID uuid.UUID) (*entities.Activity, error) {
	startTime, err := garmin.ParseStartTime(activity.StartTimeLocal)
	if err != nil {
		return nil, err
	}

	normalized := &entities.Activity{
		AthleteID:   athleteID,
		ExternalID:  fmt.Sprintf("%d", activity.ActivityID),
		Sport:       activity.ActivityType.TypeKey,
		SportFamily: garmin.SportFamily(activity.ActivityType.TypeKey),
		StartTime:   startTime,
		DurationSec: int(activity.Duration),
		Raw:         mustMarshal(activity),
	}
	if activity.Distance > 0 {
		distance := activity.Distance
		normalized.DistanceM = &distance
	}
	if activity.AverageHR > 0 {
		avg := int(activity.AverageHR)
		normalized.AvgHR = &avg
	}
	if activity.MaxHR > 0 {
		max := int(activity.MaxHR)
		normalized.MaxHR = &max
	}
	if activity.VO2Max > 0 {
		vo2max := activity.VO2Max
		normalized.VO2Max = &vo2max
	}
	return normalized, nil
}

func mustMarshal(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}
