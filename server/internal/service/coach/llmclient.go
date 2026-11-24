package coach

import (
	"context"
	"encoding/json"
)

// LLMClient is the OpenAI-compatible chat-completions backend of the coach.
// Implementations stream text deltas to onDelta as they arrive and return
// the complete turn, including any tool calls the model requested.
//
//go:generate mockgen -destination ./mock/mocks.go -package mock . LLMClient
type LLMClient interface {
	Complete(ctx context.Context, req CompletionRequest, onDelta func(string)) (*CompletionResult, error)
}

// CompletionRequest is one chat-completions turn.
type CompletionRequest struct {
	SystemPrompt string    `json:"-"`
	Messages     []Message `json:"messages"`
	Tools        []ToolDef `json:"tools,omitempty"`
}

// Message is one message in the conversation. Role is user, assistant or
// tool; assistant messages may carry tool calls, tool messages carry results.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall is a function call the model requested.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDef describes one callable tool in the OpenAI tools format.
type ToolDef struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// CompletionResult is the outcome of one model turn.
type CompletionResult struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
