package coach

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

// SnapshotDays is how many days of activity history the snapshot carries.
const SnapshotDays = 7

// Snapshot is the compact, always-injected context the coach reasons over.
// It is a curated roll-up, never raw data dumps, to bound token cost.
type Snapshot struct {
	AsOf        string                     `json:"asOf"`
	Athlete     snapshotAthlete            `json:"athlete"`
	ActiveGoals []entities.Goal            `json:"activeGoals"`
	Plan        *entities.PlanWithSessions `json:"plan,omitempty"`
	Activities  []activitySummary          `json:"last7dActivities"`
	LastNight   *entities.DailyHealth      `json:"lastNight,omitempty"`
}

type snapshotAthlete struct {
	BirthYear          *int       `json:"birthYear,omitempty"`
	Sex                *string    `json:"sex,omitempty"`
	WeightKg           *float64   `json:"weightKg,omitempty"`
	RunningExperience  *string    `json:"runningExperience,omitempty"`
	ClimbingExperience *string    `json:"climbingExperience,omitempty"`
	MaxHR              int        `json:"maxHr,omitempty"`
	RestingHR          int        `json:"restingHr,omitempty"`
	VO2Max             *float64   `json:"vo2max,omitempty"`
	WeeklyRunKm        []weeklyKm `json:"weeklyRunKm"`
}

type weeklyKm struct {
	Week       string  `json:"week"`
	DistanceKm float64 `json:"km"`
}

type activitySummary struct {
	Date        string   `json:"date"`
	Sport       string   `json:"sport"`
	Family      string   `json:"family"`
	DurationMin int      `json:"min"`
	DistanceKm  *float64 `json:"km,omitempty"`
}

// buildSnapshot assembles the current coaching context for the athlete.
func (s *Service) buildSnapshot(ctx context.Context, athleteID uuid.UUID) (Snapshot, error) {
	now := time.Now().UTC()
	snapshot := Snapshot{
		AsOf:        now.Format("2006-01-02 15:04 MST"),
		ActiveGoals: []entities.Goal{},
		Activities:  []activitySummary{},
	}

	athlete, err := s.deps.Profile.Get(ctx, athleteID)
	if err != nil {
		return snapshot, err
	}
	snapshot.Athlete = snapshotAthlete{
		BirthYear:          athlete.BirthYear,
		Sex:                athlete.Sex,
		WeightKg:           athlete.WeightKg,
		RunningExperience:  athlete.RunningExperience,
		ClimbingExperience: athlete.ClimbingExperience,
		WeeklyRunKm:        []weeklyKm{},
	}

	if derived, err := s.deps.Profile.GetDerived(ctx, athleteID); err == nil {
		if derived.HeartRate != nil {
			snapshot.Athlete.MaxHR = derived.HeartRate.MaxHeartRate
			snapshot.Athlete.RestingHR = derived.HeartRate.RestingHR
		}
		snapshot.Athlete.VO2Max = derived.VO2Max
		for _, volume := range derived.WeeklyVolumes {
			snapshot.Athlete.WeeklyRunKm = append(snapshot.Athlete.WeeklyRunKm, weeklyKm{
				Week:       volume.Week.String(),
				DistanceKm: volume.DistanceKm,
			})
		}
	}

	if goalsList, err := s.deps.Repo.ListGoals(ctx, athleteID); err == nil {
		for _, goal := range goalsList {
			if goal.Status == entities.GoalStatusActive {
				snapshot.ActiveGoals = append(snapshot.ActiveGoals, goal)
			}
		}
	}

	if plans, err := s.deps.Repo.ListPlans(ctx, athleteID); err == nil {
		for _, plan := range plans {
			if plan.Status == entities.PlanStatusActive {
				if full, err := s.deps.Plans.GetPlan(ctx, athleteID, plan.ID); err == nil {
					snapshot.Plan = &full
				}
				break
			}
		}
	}

	from := entities.Date(now.AddDate(0, 0, -(SnapshotDays - 1)))
	to := entities.Date(now)
	if activities, err := s.deps.Repo.ListActivities(ctx, athleteID, from, to); err == nil {
		for _, activity := range activities {
			summary := activitySummary{
				Date:        activity.StartTime.UTC().Format("2006-01-02"),
				Sport:       activity.Sport,
				Family:      activity.SportFamily,
				DurationMin: activity.DurationSec / 60,
			}
			if activity.DistanceM != nil {
				km := *activity.DistanceM / 1000
				summary.DistanceKm = &km
			}
			snapshot.Activities = append(snapshot.Activities, summary)
		}
	}

	if health, err := s.deps.Repo.ListDailyHealth(ctx, athleteID, to, to); err == nil && len(health) > 0 {
		snapshot.LastNight = &health[0]
	}

	return snapshot, nil
}
