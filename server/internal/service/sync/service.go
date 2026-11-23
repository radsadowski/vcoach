// Package sync pulls the athlete's data from Garmin into the local database:
// activities, daily health and heart-rate zones. Syncs are idempotent and
// cursor-based; failures are recorded per source and degrade gracefully.
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

// BackfillDays is how far back the first sync reaches.
const BackfillDays = 90

// OverlapDays re-fetches a small window behind the cursor so late-arriving
// data and edits on Garmin's side are picked up.
const OverlapDays = 2

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

// syncActivities pulls activities since the cursor (minus overlap) and upserts
// them. The cursor only advances after a successful persist.
func (s *Service) syncActivities(ctx context.Context, athleteID uuid.UUID) (int, error) {
	today := entities.Date(time.Now().UTC())
	state, err := s.repo.GetSyncState(ctx, athleteID, entities.SyncSourceGarminActivities)
	if err != nil {
		return 0, fmt.Errorf("read activities sync state: %w", err)
	}

	from := today.AddDays(-(BackfillDays - 1))
	if state != nil && state.Cursor != nil {
		cursor := entities.Date(*state.Cursor)
		if overlapStart := cursor.AddDays(-OverlapDays); overlapStart.After(from) {
			from = overlapStart
		}
	}

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

// syncDailyHealth merges the daily health endpoints into one row per day.
// Nil metrics keep their stored values on conflict, so partial fetches never
// erase data.
func (s *Service) syncDailyHealth(ctx context.Context, athleteID uuid.UUID) (int, error) {
	today := entities.Date(time.Now().UTC())
	state, err := s.repo.GetSyncState(ctx, athleteID, entities.SyncSourceGarminDailyHealth)
	if err != nil {
		return 0, fmt.Errorf("read health sync state: %w", err)
	}

	from := today.AddDays(-(BackfillDays - 1))
	if state != nil && state.Cursor != nil {
		cursor := entities.Date(*state.Cursor)
		if overlapStart := cursor.AddDays(-OverlapDays); overlapStart.After(from) {
			from = overlapStart
		}
	}

	byDay := map[entities.Date]*entities.DailyHealth{}
	rawByDay := map[entities.Date]map[string]json.RawMessage{}

	add := func(day string) *entities.DailyHealth {
		date := entities.Date(mustParseDate(day))
		if _, ok := byDay[date]; !ok {
			byDay[date] = &entities.DailyHealth{AthleteID: athleteID, Day: date}
			rawByDay[date] = map[string]json.RawMessage{}
		}
		return byDay[date]
	}

	if summaries, err := s.client.DailySummaries(ctx, from.String(), today.String()); err == nil {
		for _, summary := range summaries {
			day := add(summary.CalendarDate)
			if summary.RestingHeartRate > 0 {
				resting := summary.RestingHeartRate
				day.RestingHR = &resting
			}
			if summary.Steps > 0 {
				steps := summary.Steps
				day.Steps = &steps
			}
			rawByDay[day.Day]["daily_summary"] = mustMarshal(summary)
		}
	} else {
		s.log.Warn("daily summaries unavailable", "error", err)
	}

	if stress, err := s.client.StressDaily(ctx, from.String(), today.String()); err == nil {
		for _, day := range stress {
			if day.AverageStressLevel > 0 {
				entry := add(day.CalendarDate)
				avg := day.AverageStressLevel
				entry.StressAvg = &avg
				rawByDay[entry.Day]["stress"] = mustMarshal(day)
			}
		}
	} else {
		s.log.Warn("stress unavailable", "error", err)
	}

	if hrv, err := s.client.HRVDaily(ctx, from.String(), today.String()); err == nil {
		for _, day := range hrv {
			if day.LastNightAvg > 0 {
				entry := add(day.CalendarDate)
				hrvAvg := day.LastNightAvg
				entry.HRVMS = &hrvAvg
				rawByDay[entry.Day]["hrv"] = mustMarshal(day)
			}
		}
	} else {
		s.log.Warn("hrv unavailable", "error", err)
	}

	if sleep, err := s.client.SleepSummaries(ctx, from.String(), today.String()); err == nil {
		for _, day := range sleep {
			entry := add(day.CalendarDate)
			if day.SleepTimeSeconds > 0 {
				hours := day.SleepTimeSeconds / 3600
				entry.SleepHours = &hours
			}
			if day.SleepScores.Overall.Value > 0 {
				score := day.SleepScores.Overall.Value
				entry.SleepScore = &score
			}
			rawByDay[entry.Day]["sleep"] = mustMarshal(day)
		}
	} else {
		s.log.Warn("sleep unavailable", "error", err)
	}

	if battery, err := s.client.BodyBatteryDaily(ctx, from.String(), today.String()); err == nil {
		for _, day := range battery {
			if day.Charged > 0 {
				entry := add(day.CalendarDate)
				charged := day.Charged
				entry.BodyBattery = &charged
				rawByDay[entry.Day]["body_battery"] = mustMarshal(day)
			}
		}
	} else {
		s.log.Warn("body battery unavailable", "error", err)
	}

	if weights, err := s.client.WeightRange(ctx, from.String(), today.String()); err == nil {
		for _, measurement := range weights {
			if measurement.Weight > 0 {
				entry := add(measurement.Date)
				weight := measurement.Weight
				entry.WeightKg = &weight
				rawByDay[entry.Day]["weight"] = mustMarshal(measurement)
			}
		}
	} else {
		s.log.Warn("weight unavailable", "error", err)
	}

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

	if err := s.repo.SaveSyncStateSuccess(ctx, athleteID, entities.SyncSourceGarminDailyHealth, today); err != nil {
		return len(byDay), fmt.Errorf("save health sync state: %w", err)
	}
	return len(byDay), nil
}

// appSettingHRZones is the app_settings key for the synced Garmin zones.
const appSettingHRZones = "garmin_hr_zones"

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

func mustParseDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func mustMarshal(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}
