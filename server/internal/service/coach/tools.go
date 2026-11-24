package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	"github.com/radosny/vcoach/server/internal/service/goals"
	"github.com/radosny/vcoach/server/internal/service/plans"
)

// maxToolRounds bounds the tool-call loop so a runaway model cannot loop
// forever; one more round than the expected maximum (create goal + plan).
const maxToolRounds = 6

// maxReadRangeDays bounds read-tool date ranges (token and query safety).
const maxReadRangeDays = 31

// executeTool runs one tool call and returns its JSON result. Errors are
// returned as tool results (not Go errors) so the model can correct itself.
func (s *Service) executeTool(ctx context.Context, athleteID uuid.UUID, call ToolCall) string {
	switch call.Name {
	case "create_goal":
		return s.toolCreateGoal(ctx, athleteID, call.Arguments)
	case "set_primary_goal":
		return s.toolSetPrimaryGoal(ctx, athleteID, call.Arguments)
	case "update_goal_status":
		return s.toolUpdateGoalStatus(ctx, athleteID, call.Arguments)
	case "create_training_plan":
		return s.toolCreatePlan(ctx, athleteID, call.Arguments)
	case "archive_plan":
		return s.toolArchivePlan(ctx, athleteID, call.Arguments)
	case "get_activities":
		return s.toolGetActivities(ctx, athleteID, call.Arguments)
	case "get_daily_health":
		return s.toolGetDailyHealth(ctx, athleteID, call.Arguments)
	case "get_plan_compliance":
		return s.toolGetPlanCompliance(ctx, athleteID)
	case "get_climbing_log":
		return s.toolGetClimbingLog(ctx, athleteID, call.Arguments)
	default:
		return toolError(fmt.Sprintf("unknown tool %q", call.Name))
	}
}

func (s *Service) toolCreateGoal(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input entities.CreateGoalInput
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	goal, err := s.deps.Goals.CreateGoal(ctx, athleteID, input)
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"created": true, "goal": goal})
}

func (s *Service) toolSetPrimaryGoal(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		GoalID string `json:"goalId"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	goalID, err := uuid.Parse(input.GoalID)
	if err != nil {
		return toolError("goalId must be a UUID")
	}

	primary := true
	goal, err := s.deps.Goals.UpdateGoal(ctx, athleteID, goalID, entities.UpdateGoalInput{IsPrimary: &primary})
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"updated": true, "goal": goal})
}

func (s *Service) toolUpdateGoalStatus(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		GoalID string              `json:"goalId"`
		Status entities.GoalStatus `json:"status"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	goalID, err := uuid.Parse(input.GoalID)
	if err != nil {
		return toolError("goalId must be a UUID")
	}

	goal, err := s.deps.Goals.UpdateGoal(ctx, athleteID, goalID, entities.UpdateGoalInput{Status: &input.Status})
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"updated": true, "goal": goal})
}

func (s *Service) toolCreatePlan(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input entities.CreatePlanInput
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	if input.GoalID != nil && *input.GoalID == uuid.Nil {
		input.GoalID = nil
	}

	plan, err := s.deps.Plans.CreatePlan(ctx, athleteID, input)
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"created": true, "plan": plan})
}

func (s *Service) toolArchivePlan(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		PlanID string `json:"planId"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	planID, err := uuid.Parse(input.PlanID)
	if err != nil {
		return toolError("planId must be a UUID")
	}

	if err := s.deps.Plans.ArchivePlan(ctx, athleteID, planID); err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"archived": true})
}

func (s *Service) toolGetActivities(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	from, to, err := parseRange(input.From, input.To)
	if err != nil {
		return toolError(err.Error())
	}

	activities, err := s.deps.Repo.ListActivities(ctx, athleteID, from, to)
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"activities": summarizeActivities(activities)})
}

func (s *Service) toolGetDailyHealth(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	from, to, err := parseRange(input.From, input.To)
	if err != nil {
		return toolError(err.Error())
	}

	rows, err := s.deps.Repo.ListDailyHealth(ctx, athleteID, from, to)
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"dailyHealth": rows})
}

func (s *Service) toolGetPlanCompliance(ctx context.Context, athleteID uuid.UUID) string {
	activePlans, err := s.deps.Repo.ListPlans(ctx, athleteID)
	if err != nil {
		return toolError(err.Error())
	}
	for _, plan := range activePlans {
		if plan.Status != entities.PlanStatusActive {
			continue
		}
		full, err := s.deps.Plans.GetPlan(ctx, athleteID, plan.ID)
		if err != nil {
			return toolError(err.Error())
		}
		return toolResult(map[string]any{"plan": full})
	}
	return toolResult(map[string]any{"plan": nil, "note": "no active plan"})
}

func (s *Service) toolGetClimbingLog(ctx context.Context, athleteID uuid.UUID, args string) string {
	var input struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}
	from, to, err := parseRange(input.From, input.To)
	if err != nil {
		return toolError(err.Error())
	}

	fromTime, toTime := from.Time(), to.Time()
	sessions, err := s.deps.Repo.ListClimbingSessions(ctx, athleteID, &fromTime, &toTime)
	if err != nil {
		return toolError(err.Error())
	}
	return toolResult(map[string]any{"climbingSessions": sessions})
}

func parseRange(fromStr, toStr string) (entities.Date, entities.Date, error) {
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return entities.Date{}, entities.Date{}, fmt.Errorf("from must be YYYY-MM-DD, got %q", fromStr)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return entities.Date{}, entities.Date{}, fmt.Errorf("to must be YYYY-MM-DD, got %q", toStr)
	}
	if to.Before(from) {
		return entities.Date{}, entities.Date{}, fmt.Errorf("to (%s) is before from (%s)", toStr, fromStr)
	}
	if to.Sub(from).Hours()/24 > maxReadRangeDays {
		return entities.Date{}, entities.Date{}, fmt.Errorf("range too wide: max %d days", maxReadRangeDays)
	}
	return entities.Date(from), entities.Date(to), nil
}

// summarizeActivities keeps tool results compact.
func summarizeActivities(activities []entities.Activity) []map[string]any {
	summaries := make([]map[string]any, 0, len(activities))
	for _, activity := range activities {
		summary := map[string]any{
			"date":        activity.StartTime.UTC().Format("2006-01-02"),
			"sport":       activity.Sport,
			"sportFamily": activity.SportFamily,
			"durationMin": activity.DurationSec / 60,
		}
		if activity.DistanceM != nil {
			summary["distanceKm"] = *activity.DistanceM / 1000
		}
		if activity.AvgHR != nil {
			summary["avgHr"] = *activity.AvgHR
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func toolResult(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return `{"error":"encoding result failed"}`
	}
	return string(raw)
}

func toolError(message string) string {
	return toolResult(map[string]any{"error": message})
}

// Compile-time interface checks for the tool dependencies.
var _ interface {
	CreateGoal(context.Context, uuid.UUID, entities.CreateGoalInput) (entities.Goal, error)
	UpdateGoal(context.Context, uuid.UUID, uuid.UUID, entities.UpdateGoalInput) (entities.Goal, error)
} = (*goals.Service)(nil)

var _ interface {
	CreatePlan(context.Context, uuid.UUID, entities.CreatePlanInput) (entities.PlanWithSessions, error)
	GetPlan(context.Context, uuid.UUID, uuid.UUID) (entities.PlanWithSessions, error)
	ArchivePlan(context.Context, uuid.UUID, uuid.UUID) error
} = (*plans.Service)(nil)

var _ interface {
	List(context.Context, uuid.UUID, *string, *string) ([]entities.ClimbingSession, error)
} = (*climbingsvc.Service)(nil)
