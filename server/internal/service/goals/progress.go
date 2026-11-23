package goals

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
)

// nowUTC is the clock used for progress windows.
func nowUTC() time.Time {
	return time.Now().UTC()
}

// ProgressDays is the look-back window for goal progress.
const ProgressDays = 30

// RunProgress holds progress facts for run goals.
type RunProgress struct {
	LongestRunKm     *float64                     `json:"longestRunKm,omitempty"`
	BestPaceSecPerKm *int                         `json:"bestPaceSecPerKm,omitempty"`
	WeeklyVolumes    []repository.WeeklyRunVolume `json:"weeklyVolumes"`
}

// ClimbProgress holds progress facts for boulder/lead goals.
type ClimbProgress struct {
	Discipline      string `json:"discipline"`
	HardestSent     string `json:"hardestSent"`
	HardestSentRank int    `json:"hardestSentRank"`
	TargetRank      int    `json:"targetRank"`
	RankGap         int    `json:"rankGap"`
	Attempts30d     int    `json:"attempts30d"`
	Sessions30d     int    `json:"sessions30d"`
}

// Progress is the per-goal progress report.
type Progress struct {
	GoalID       uuid.UUID                 `json:"goalId"`
	ActivityType entities.GoalActivityType `json:"activityType"`
	TargetValue  string                    `json:"targetValue"`
	Run          *RunProgress              `json:"run,omitempty"`
	Climb        *ClimbProgress            `json:"climb,omitempty"`
}

// GetProgress computes the progress report for one goal.
func (s *Service) GetProgress(ctx context.Context, athleteID, goalID uuid.UUID) (Progress, error) {
	goal, err := s.GetGoal(ctx, athleteID, goalID)
	if err != nil {
		return Progress{}, err
	}

	progress := Progress{
		GoalID:       goal.ID,
		ActivityType: goal.ActivityType,
		TargetValue:  goal.TargetValue,
	}

	switch goal.ActivityType {
	case entities.GoalActivityRun:
		progress.Run, err = s.runProgress(ctx, athleteID)
	case entities.GoalActivityBoulder:
		progress.Climb, err = s.climbProgress(ctx, athleteID, "boulder", goal.TargetValue, entities.BoulderGradeRank)
	case entities.GoalActivityLead:
		progress.Climb, err = s.climbProgress(ctx, athleteID, "lead", goal.TargetValue, entities.LeadGradeRank)
	}
	if err != nil {
		return Progress{}, err
	}
	return progress, nil
}

func (s *Service) runProgress(ctx context.Context, athleteID uuid.UUID) (*RunProgress, error) {
	since := entities.Date(nowUTC().AddDate(0, 0, -ProgressDays))

	progress := &RunProgress{}

	km, err := s.repo.LongestRunKm(ctx, athleteID, since)
	if err != nil {
		return nil, err
	}
	progress.LongestRunKm = km

	pace, err := s.repo.BestRunPaceSecPerKm(ctx, athleteID, since)
	if err != nil {
		return nil, err
	}
	progress.BestPaceSecPerKm = pace

	volumes, err := s.repo.WeeklyRunVolumes(ctx, athleteID, 4)
	if err != nil {
		return nil, err
	}
	progress.WeeklyVolumes = volumes
	if progress.WeeklyVolumes == nil {
		progress.WeeklyVolumes = []repository.WeeklyRunVolume{}
	}
	return progress, nil
}

func (s *Service) climbProgress(ctx context.Context, athleteID uuid.UUID, discipline, target string, rank func(string) (int, error)) (*ClimbProgress, error) {
	targetRank, err := rank(target)
	if err != nil {
		return nil, err
	}

	progress := &ClimbProgress{
		Discipline: discipline,
		TargetRank: targetRank,
	}

	from := nowUTC().AddDate(0, 0, -ProgressDays)
	to := nowUTC()
	sessions, err := s.repo.ListClimbingSessions(ctx, athleteID, &from, &to)
	if err != nil {
		return nil, err
	}

	sentRank := -1
	for _, session := range sessions {
		if session.Discipline != discipline {
			continue
		}
		progress.Sessions30d++
		progress.Attempts30d += session.Attempts

		var rank int
		if discipline == "boulder" {
			rank, _ = entities.BoulderGradeRank(session.HardestSent)
		} else {
			rank, _ = entities.LeadGradeRank(session.HardestSent)
		}
		if rank > sentRank {
			sentRank = rank
			progress.HardestSent = session.HardestSent
			progress.HardestSentRank = rank
		}
	}

	if sentRank >= 0 {
		progress.RankGap = targetRank - sentRank
		if progress.RankGap < 0 {
			progress.RankGap = 0
		}
	}
	return progress, nil
}
