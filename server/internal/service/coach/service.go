// Package coach is the conversational AI coach: it orchestrates the LLM
// tool-call loop, builds the context snapshot and streams replies. It is
// the only writer of goals and plans.
package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	"github.com/radosny/vcoach/server/internal/repository"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	"github.com/radosny/vcoach/server/internal/service/goals"
	"github.com/radosny/vcoach/server/internal/service/plans"
	"github.com/radosny/vcoach/server/internal/service/profile"
)

// Dependencies bundles the coach's collaborators.
type Dependencies struct {
	Repo     *repository.Repository
	Log      *slog.Logger
	LLM      LLMClient
	Goals    *goals.Service
	Plans    *plans.Service
	Climbing *climbingsvc.Service
	Profile  *profile.Service
}

// Service implements the coach conversation loop.
type Service struct {
	deps Dependencies
}

// New builds a coach Service.
func New(deps Dependencies) *Service {
	return &Service{deps: deps}
}

// SendMessage processes one user message: it persists the message, builds
// the system prompt with a fresh snapshot, then runs the LLM tool-call loop,
// streaming final-answer deltas to onDelta. Tool calls execute atomically
// against the services; the persisted assistant message records them.
func (s *Service) SendMessage(ctx context.Context, athleteID, sessionID uuid.UUID, userMessage string, onDelta func(string)) (entities.ChatMessage, error) {
	if userMessage == "" {
		return entities.ChatMessage{}, fmt.Errorf("coach: empty message")
	}

	if _, err := s.deps.Repo.CreateChatMessage(ctx, athleteID, sessionID, entities.ChatRoleUser, userMessage, nil); err != nil {
		return entities.ChatMessage{}, fmt.Errorf("coach: persist user message: %w", err)
	}

	history, err := s.deps.Repo.ListChatMessages(ctx, athleteID, sessionID)
	if err != nil {
		return entities.ChatMessage{}, fmt.Errorf("coach: load history: %w", err)
	}

	snapshot, err := s.buildSnapshot(ctx, athleteID)
	if err != nil {
		return entities.ChatMessage{}, fmt.Errorf("coach: build snapshot: %w", err)
	}

	messages := make([]Message, 0, len(history)+4)
	for _, msg := range history {
		messages = append(messages, Message{Role: msg.Role, Content: msg.Content})
	}

	var lastAssistant entities.ChatMessage
	for round := 0; round < maxToolRounds; round++ {
		request := CompletionRequest{
			SystemPrompt: systemPrompt(snapshot),
			Messages:     messages,
			Tools:        toolDefs(),
		}

		result, err := s.deps.LLM.Complete(ctx, request, onDelta)
		if err != nil {
			return entities.ChatMessage{}, err
		}

		if len(result.ToolCalls) == 0 {
			// Final answer: persist and finish.
			lastAssistant, err = s.deps.Repo.CreateChatMessage(ctx, athleteID, sessionID, entities.ChatRoleAssistant, result.Content, nil)
			if err != nil {
				return entities.ChatMessage{}, fmt.Errorf("coach: persist assistant message: %w", err)
			}
			return lastAssistant, nil
		}

		// The model requested tools: record the assistant turn with tool
		// calls, execute them, and feed results back into the loop.
		messages = append(messages, Message{
			Role:      entities.ChatRoleAssistant,
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		toolCallsJSON, _ := json.Marshal(result.ToolCalls)
		lastAssistant, err = s.deps.Repo.CreateChatMessage(ctx, athleteID, sessionID, entities.ChatRoleAssistant, result.Content, toolCallsJSON)
		if err != nil {
			return entities.ChatMessage{}, fmt.Errorf("coach: persist tool turn: %w", err)
		}

		for _, call := range result.ToolCalls {
			toolResult := s.executeTool(ctx, athleteID, call)
			messages = append(messages, Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: call.ID,
				Name:       call.Name,
			})
		}
	}

	return lastAssistant, fmt.Errorf("coach: exceeded %d tool rounds", maxToolRounds)
}
