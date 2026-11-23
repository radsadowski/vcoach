package profile

import (
	"context"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/service/garmin"
)

// appSettingHRZones mirrors the key used by the sync service.
const appSettingHRZones = "garmin_hr_zones"

// Derived aggregates the Garmin-derived profile metrics computed on read.
type Derived struct {
	HeartRate     *HeartRateInfo `json:"heartRate,omitempty"`
	VO2Max        *float64       `json:"vo2max,omitempty"`
	WeeklyVolumes []WeeklyVolume `json:"weeklyVolumes"`
	LongestRunKm  *float64       `json:"longestRunKm,omitempty"`
	Weight        *WeightTrend   `json:"weight,omitempty"`
}

// HeartRateInfo is the athlete's heart-rate data from Garmin.
type HeartRateInfo struct {
	MaxHeartRate int `json:"maxHeartRate"`
	RestingHR    int `json:"restingHr"`
	Zone1Floor   int `json:"zone1Floor"`
	Zone2Floor   int `json:"zone2Floor"`
	Zone3Floor   int `json:"zone3Floor"`
	Zone4Floor   int `json:"zone4Floor"`
}

// WeeklyVolume is distance run per week.
type WeeklyVolume struct {
	Week         entities.Date `json:"week"`
	DistanceKm   float64       `json:"distanceKm"`
	SessionCount int           `json:"sessionCount"`
}

// WeightTrend compares the latest weight against the value 30 days ago.
type WeightTrend struct {
	LatestKg  float64 `json:"latestKg"`
	DaysAgoKg float64 `json:"daysAgoKg"`
	ChangeKg  float64 `json:"changeKg"`
}

// GetDerived computes the derived profile metrics from synced data.
func (s *Service) GetDerived(ctx context.Context, athleteID uuid.UUID) (Derived, error) {
	derived := Derived{}

	var zones garmin.HeartRateZones
	if ok, err := s.repo.GetAppSetting(ctx, appSettingHRZones, &zones); err == nil && ok && zones.MaxHeartRate > 0 {
		derived.HeartRate = &HeartRateInfo{
			MaxHeartRate: zones.MaxHeartRate,
			RestingHR:    zones.RestingHR,
			Zone1Floor:   zones.Zone1Floor,
			Zone2Floor:   zones.Zone2Floor,
			Zone3Floor:   zones.Zone3Floor,
			Zone4Floor:   zones.Zone4Floor,
		}
	}

	if vo2max, err := s.repo.LatestVO2Max(ctx, athleteID); err == nil && vo2max != nil {
		derived.VO2Max = vo2max
	}

	if volumes, err := s.repo.WeeklyRunVolumes(ctx, athleteID, 4); err == nil {
		for _, volume := range volumes {
			derived.WeeklyVolumes = append(derived.WeeklyVolumes, WeeklyVolume{
				Week:         volume.Week,
				DistanceKm:   volume.DistanceKm,
				SessionCount: volume.SessionCount,
			})
		}
	}
	if derived.WeeklyVolumes == nil {
		derived.WeeklyVolumes = []WeeklyVolume{}
	}

	since30d := entities.Date(nowUTC().AddDate(0, 0, -30))
	if km, err := s.repo.LongestRunKm(ctx, athleteID, since30d); err == nil {
		derived.LongestRunKm = km
	}

	if latest, err := s.repo.LatestWeight(ctx, athleteID); err == nil && latest != nil {
		trend := &WeightTrend{LatestKg: *latest}
		if daysAgo, err := s.repo.WeightDaysAgo(ctx, athleteID, 30); err == nil && daysAgo != nil {
			trend.DaysAgoKg = *daysAgo
			trend.ChangeKg = *latest - *daysAgo
		}
		derived.Weight = trend
	}

	return derived, nil
}

// AthleteWithDerived is the full profile response.
type AthleteWithDerived struct {
	entities.Athlete
	Derived Derived `json:"derived"`
}

// GetWithDerived returns the profile basics plus derived metrics.
func (s *Service) GetWithDerived(ctx context.Context, athleteID uuid.UUID) (AthleteWithDerived, error) {
	athlete, err := s.Get(ctx, athleteID)
	if err != nil {
		return AthleteWithDerived{}, err
	}
	derived, err := s.GetDerived(ctx, athleteID)
	if err != nil {
		return AthleteWithDerived{}, err
	}
	return AthleteWithDerived{Athlete: athlete, Derived: derived}, nil
}
