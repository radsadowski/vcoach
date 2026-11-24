package coach_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	"github.com/radosny/vcoach/server/internal/service/coach"
	"github.com/radosny/vcoach/server/internal/service/coach/mock"
	"github.com/radosny/vcoach/server/internal/service/goals"
	"github.com/radosny/vcoach/server/internal/service/plans"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
	"github.com/radosny/vcoach/server/internal/testing/database"
)

func newCoach(t *testing.T, llm coach.LLMClient) (*coach.Service, *repository.Repository, uuid.UUID) {
	t.Helper()
	db := database.SetupDB(t)
	repo := repository.New(db)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := coach.New(coach.Dependencies{
		Repo:     repo,
		Log:      log,
		LLM:      llm,
		Goals:    goals.New(repo),
		Plans:    plans.New(repo),
		Climbing: climbingsvc.New(repo),
		Profile:  profilesvc.New(repo, log),
	})

	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	require.NoError(t, err)
	return svc, repo, athleteID
}

func TestSendMessage(t *testing.T) {
	t.Parallel()

	t.Run("executes tool calls and persists the final answer", func(t *testing.T) {
		r := require.New(t)

		// given: the model first requests a goal, then a plan, then answers
		ctrl := gomock.NewController(t)
		llm := mock.NewMockLLMClient(ctrl)

		goalArgs := `{"activityType":"boulder","targetMetric":"grade","targetValue":"7C","isPrimary":true}`
		planArgs := fmt.Sprintf(`{"title":"7C project","weeks":[{"weekNumber":1,"startDate":"%s","focus":"base","sessions":[{"sport":"boulder","sessionType":"limit bouldering","plannedDate":"%s"}]}]}`,
			"2026-11-24", "2026-11-25")

		gomock.InOrder(
			llm.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ coach.CompletionRequest, _ func(string)) (*coach.CompletionResult, error) {
					return &coach.CompletionResult{
						ToolCalls: []coach.ToolCall{
							{ID: "call-goal", Name: "create_goal", Arguments: goalArgs},
							{ID: "call-plan", Name: "create_training_plan", Arguments: planArgs},
						},
					}, nil
				}),
			llm.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ coach.CompletionRequest, onDelta func(string)) (*coach.CompletionResult, error) {
					onDelta("Plan is ready!")
					return &coach.CompletionResult{Content: "Plan is ready!"}, nil
				}),
		)

		svc, repo, athleteID := newCoach(t, llm)
		ctx := context.Background()

		session, err := repo.CreateChatSession(ctx, athleteID, "Season")
		r.NoError(err)

		// when
		var streamed []string
		message, err := svc.SendMessage(ctx, athleteID, session.ID, "I want to climb 7C. Build me a plan.", func(delta string) {
			streamed = append(streamed, delta)
		})

		// then: the goal and plan exist as structured data
		r.NoError(err)
		r.Equal("Plan is ready!", message.Content)
		r.Equal(entities.ChatRoleAssistant, message.Role)

		goalList, err := repo.ListGoals(ctx, athleteID)
		r.NoError(err)
		r.Len(goalList, 1)
		r.Equal("7C", goalList[0].TargetValue)
		r.True(goalList[0].IsPrimary)

		planList, err := repo.ListPlans(ctx, athleteID)
		r.NoError(err)
		r.Len(planList, 1)
		r.Equal(1, planList[0].Version)
		r.Equal(entities.PlanStatusActive, planList[0].Status)

		// and: messages persisted — user, tool turn, final answer
		history, err := repo.ListChatMessages(ctx, athleteID, session.ID)
		r.NoError(err)
		r.Len(history, 3)
		r.Equal(entities.ChatRoleUser, history[0].Role)
		r.NotNil(history[1].ToolCalls)
		r.Equal(entities.ChatRoleAssistant, history[2].Role)

		// and: deltas were streamed
		r.Equal([]string{"Plan is ready!"}, streamed)
	})

	t.Run("feeds tool errors back to the model", func(t *testing.T) {
		r := require.New(t)

		// given: the model sends an invalid goal first, a valid one after
		ctrl := gomock.NewController(t)
		llm := mock.NewMockLLMClient(ctrl)

		var secondRoundMessages []coach.Message
		gomock.InOrder(
			llm.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ coach.CompletionRequest, _ func(string)) (*coach.CompletionResult, error) {
					return &coach.CompletionResult{
						ToolCalls: []coach.ToolCall{
							{ID: "call-1", Name: "create_goal", Arguments: `{"activityType":"boulder","targetMetric":"grade","targetValue":"NOT_A_GRADE"}`},
						},
					}, nil
				}),
			llm.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, req coach.CompletionRequest, _ func(string)) (*coach.CompletionResult, error) {
					secondRoundMessages = req.Messages
					return &coach.CompletionResult{Content: "Fixed it."}, nil
				}),
		)

		svc, repo, athleteID := newCoach(t, llm)
		ctx := context.Background()
		session, err := repo.CreateChatSession(ctx, athleteID, "tool errors")
		r.NoError(err)

		// when
		_, err = svc.SendMessage(ctx, athleteID, session.ID, "make me a goal", nil)
		r.NoError(err)

		// then: the tool result with the validation error went back to the model
		var toolResultMessage *coach.Message
		for i := range secondRoundMessages {
			if secondRoundMessages[i].Role == "tool" {
				toolResultMessage = &secondRoundMessages[i]
			}
		}
		r.NotNil(toolResultMessage)
		r.Contains(toolResultMessage.Content, "error")
		r.Contains(toolResultMessage.Content, "grade")
	})

	t.Run("snapshot carries profile, goals, plan and activities", func(t *testing.T) {
		r := require.New(t)

		// given
		ctrl := gomock.NewController(t)
		llm := mock.NewMockLLMClient(ctrl)

		var capturedPrompt string
		llm.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req coach.CompletionRequest, _ func(string)) (*coach.CompletionResult, error) {
				capturedPrompt = req.SystemPrompt
				return &coach.CompletionResult{Content: "Got it."}, nil
			}).AnyTimes()

		svc, repo, athleteID := newCoach(t, llm)
		ctx := context.Background()

		goalSvc := goals.New(repo)
		_, err := goalSvc.CreateGoal(ctx, athleteID, entities.CreateGoalInput{
			ActivityType: entities.GoalActivityBoulder,
			TargetMetric: entities.TargetMetricGrade,
			TargetValue:  "7A",
		})
		r.NoError(err)

		distance := 8000.0
		_, err = repo.UpsertActivity(ctx, entities.Activity{
			AthleteID:   athleteID,
			ExternalID:  "snap-1",
			Sport:       "running",
			SportFamily: "run",
			StartTime:   time.Now().UTC().AddDate(0, 0, -1),
			DurationSec: 2400,
			DistanceM:   &distance,
			Raw:         []byte(`{}`),
		})
		r.NoError(err)

		session, err := repo.CreateChatSession(ctx, athleteID, "snap")
		r.NoError(err)

		// when
		_, err = svc.SendMessage(ctx, athleteID, session.ID, "how am I doing?", nil)
		r.NoError(err)

		// then: the system prompt embeds the snapshot JSON
		r.Contains(capturedPrompt, "vcoach")
		r.Contains(capturedPrompt, `"targetValue":"7A"`)
		r.Contains(capturedPrompt, `"family":"run"`)
		r.Contains(capturedPrompt, "Rules:")
	})

	t.Run("rejects empty messages", func(t *testing.T) {
		r := require.New(t)

		// given
		ctrl := gomock.NewController(t)
		llm := mock.NewMockLLMClient(ctrl)
		svc, _, athleteID := newCoach(t, llm)

		// when
		_, err := svc.SendMessage(context.Background(), athleteID, uuid.New(), "", nil)

		// then
		r.ErrorContains(err, "empty message")
	})

	t.Run("unknown session returns a clear error", func(t *testing.T) {
		r := require.New(t)

		// given
		ctrl := gomock.NewController(t)
		llm := mock.NewMockLLMClient(ctrl)
		svc, _, athleteID := newCoach(t, llm)

		// when
		_, err := svc.SendMessage(context.Background(), athleteID, uuid.New(), "hello", nil)

		// then: the user message insert fails on the missing session
		r.Error(err)
	})
}
