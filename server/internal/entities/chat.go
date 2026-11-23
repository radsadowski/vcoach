package entities

import (
	"time"

	"github.com/google/uuid"
)

// SyncState tracks per-source sync progress for one athlete.
type SyncState struct {
	Source       string     `db:"source" json:"source"`
	AthleteID    uuid.UUID  `db:"athlete_id" json:"athleteId"`
	LastSyncedAt *time.Time `db:"last_synced_at" json:"lastSyncedAt,omitempty"`
	Cursor       *time.Time `db:"cursor" json:"cursor,omitempty"`
	LastError    *string    `db:"last_error" json:"lastError,omitempty"`
	LastErrorAt  *time.Time `db:"last_error_at" json:"lastErrorAt,omitempty"`
}

// Sync sources tracked in sync_state.
const (
	SyncSourceGarminActivities  = "garmin_activities"
	SyncSourceGarminDailyHealth = "garmin_daily_health"
)

// ChatSession groups a coaching conversation.
type ChatSession struct {
	ID        uuid.UUID `db:"id" json:"id"`
	AthleteID uuid.UUID `db:"athlete_id" json:"athleteId"`
	Title     string    `db:"title" json:"title"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// ChatMessage is one turn in a coaching conversation.
type ChatMessage struct {
	ID            uuid.UUID `db:"id" json:"id"`
	ChatSessionID uuid.UUID `db:"chat_session_id" json:"chatSessionId"`
	AthleteID     uuid.UUID `db:"athlete_id" json:"athleteId"`
	Role          string    `db:"role" json:"role"`
	Content       string    `db:"content" json:"content"`
	ToolCalls     []byte    `db:"tool_calls" json:"toolCalls,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}

// Chat roles.
const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
)
