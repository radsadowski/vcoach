// Package chat implements chat session storage. Coach orchestration (LLM,
// tools, streaming) is layered on top in service/coach.
package chat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
)

// Service holds the chat business logic dependencies.
type Service struct {
	repo *repository.Repository
}

// New builds a chat Service.
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateSession starts a new chat session with an optional title.
func (s *Service) CreateSession(ctx context.Context, athleteID uuid.UUID, title string) (entities.ChatSession, error) {
	if title == "" {
		title = "New chat"
	}
	if len(title) > 200 {
		return entities.ChatSession{}, domainerrors.NewValidationError("title must be at most 200 characters")
	}
	return s.repo.CreateChatSession(ctx, athleteID, title)
}

// ListSessions returns the athlete's chat sessions, newest first.
func (s *Service) ListSessions(ctx context.Context, athleteID uuid.UUID) ([]entities.ChatSession, error) {
	return s.repo.ListChatSessions(ctx, athleteID)
}

// ListMessages returns the message history of a chat session.
func (s *Service) ListMessages(ctx context.Context, athleteID, sessionID uuid.UUID) ([]entities.ChatMessage, error) {
	if _, err := s.repo.GetChatSession(ctx, athleteID, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerrors.NewNotFoundError("chat session %s not found", sessionID)
		}
		return nil, err
	}
	return s.repo.ListChatMessages(ctx, athleteID, sessionID)
}

// AppendMessage adds a message to a chat session. Role must be user or
// assistant; toolCalls is the raw JSON record of tool calls made in the turn.
func (s *Service) AppendMessage(ctx context.Context, athleteID, sessionID uuid.UUID, role, content string, toolCalls []byte) (entities.ChatMessage, error) {
	if role != entities.ChatRoleUser && role != entities.ChatRoleAssistant {
		return entities.ChatMessage{}, domainerrors.NewValidationError("role must be user or assistant, got %q", role)
	}
	if content == "" && len(toolCalls) == 0 {
		return entities.ChatMessage{}, domainerrors.NewValidationError("content is required")
	}

	message, err := s.repo.CreateChatMessage(ctx, athleteID, sessionID, role, content, toolCalls)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.ChatMessage{}, domainerrors.NewNotFoundError("chat session %s not found", sessionID)
		}
		return entities.ChatMessage{}, err
	}
	return message, nil
}
