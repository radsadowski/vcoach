package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
)

const chatSessionColumns = `id, athlete_id, title, created_at, updated_at`
const chatMessageColumns = `id, chat_session_id, athlete_id, role, content, tool_calls, created_at`

// CreateChatSession inserts a chat session.
func (r *Repository) CreateChatSession(ctx context.Context, athleteID uuid.UUID, title string) (entities.ChatSession, error) {
	const q = `INSERT INTO chat_sessions (athlete_id, title) VALUES ($1, $2) RETURNING ` + chatSessionColumns

	var created entities.ChatSession
	if err := r.db.GetContext(ctx, &created, q, athleteID, title); err != nil {
		return entities.ChatSession{}, fmt.Errorf("insert chat session: %w", err)
	}
	return created, nil
}

// GetChatSession returns a chat session scoped to the athlete.
func (r *Repository) GetChatSession(ctx context.Context, athleteID, sessionID uuid.UUID) (entities.ChatSession, error) {
	const q = `SELECT ` + chatSessionColumns + ` FROM chat_sessions WHERE athlete_id = $1 AND id = $2`

	var session entities.ChatSession
	if err := r.db.GetContext(ctx, &session, q, athleteID, sessionID); err != nil {
		return entities.ChatSession{}, err
	}
	return session, nil
}

// ListChatSessions returns the athlete's chat sessions, newest first.
func (r *Repository) ListChatSessions(ctx context.Context, athleteID uuid.UUID) ([]entities.ChatSession, error) {
	const q = `SELECT ` + chatSessionColumns + ` FROM chat_sessions WHERE athlete_id = $1 ORDER BY updated_at DESC`

	var sessions []entities.ChatSession
	if err := r.db.SelectContext(ctx, &sessions, q, athleteID); err != nil {
		return nil, fmt.Errorf("list chat sessions: %w", err)
	}
	return sessions, nil
}

// CreateChatMessage appends a message to a chat session and refreshes the
// session's updated_at.
func (r *Repository) CreateChatMessage(ctx context.Context, athleteID, sessionID uuid.UUID, role, content string, toolCalls []byte) (entities.ChatMessage, error) {
	err := r.WithTx(ctx, func(tx *Repository) error {
		const touch = `UPDATE chat_sessions SET updated_at = now() WHERE athlete_id = $1 AND id = $2`
		res, err := tx.db.ExecContext(ctx, touch, athleteID, sessionID)
		if err != nil {
			return fmt.Errorf("touch chat session: %w", err)
		}
		if err := requireRowAffected(res, "chat session"); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return entities.ChatMessage{}, err
	}

	const q = `
		INSERT INTO chat_messages (chat_session_id, athlete_id, role, content, tool_calls)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + chatMessageColumns

	var created entities.ChatMessage
	if err := r.db.GetContext(ctx, &created, q, sessionID, athleteID, role, content, toolCalls); err != nil {
		return entities.ChatMessage{}, fmt.Errorf("insert chat message: %w", err)
	}
	return created, nil
}

// ListChatMessages returns the message history of a chat session, oldest first.
func (r *Repository) ListChatMessages(ctx context.Context, athleteID, sessionID uuid.UUID) ([]entities.ChatMessage, error) {
	const q = `
		SELECT ` + chatMessageColumns + ` FROM chat_messages
		WHERE athlete_id = $1 AND chat_session_id = $2
		ORDER BY created_at, id`

	var messages []entities.ChatMessage
	if err := r.db.SelectContext(ctx, &messages, q, athleteID, sessionID); err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	return messages, nil
}
