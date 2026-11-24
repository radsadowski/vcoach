package coach

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatibleClient talks to any OpenAI-compatible /chat/completions
// endpoint (OpenRouter, OpenAI, Ollama, LM Studio, ...). It is the only
// implementation of LLMClient; swapping providers is pure configuration.
type OpenAICompatibleClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewOpenAICompatibleClient builds a client for the configured provider.
func NewOpenAICompatibleClient(baseURL, apiKey, model string) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Configured reports whether the client has everything it needs.
func (c *OpenAICompatibleClient) Configured() bool {
	return c.baseURL != "" && c.model != ""
}

type wireRequest struct {
	Model    string        `json:"model"`
	Messages []wireMessage `json:"messages"`
	Tools    []wireTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
}

type wireTool struct {
	Type     string       `json:"type"`
	Function wireToolFunc `json:"function"`
}

type wireToolFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

type wireToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Complete streams one chat-completions turn. Text deltas are forwarded to
// onDelta as they arrive; tool calls are accumulated and returned.
func (c *OpenAICompatibleClient) Complete(ctx context.Context, req CompletionRequest, onDelta func(string)) (*CompletionResult, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("coach: LLM provider not configured (LLM_BASE_URL / LLM_MODEL)")
	}

	messages := make([]wireMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		messages = append(messages, wireMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, msg := range req.Messages {
		wire := wireMessage{Role: msg.Role, Content: msg.Content, ToolCallID: msg.ToolCallID, Name: msg.Name}
		for _, call := range msg.ToolCalls {
			var out wireToolCall
			out.ID = call.ID
			out.Type = "function"
			out.Function.Name = call.Name
			out.Function.Arguments = call.Arguments
			wire.ToolCalls = append(wire.ToolCalls, out)
		}
		messages = append(messages, wire)
	}

	tools := make([]wireTool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, wireTool{
			Type: "function",
			Function: wireToolFunc{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	body, err := json.Marshal(wireRequest{Model: c.model, Messages: messages, Tools: tools, Stream: true})
	if err != nil {
		return nil, fmt.Errorf("coach: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("coach: llm request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &LLMHTTPError{StatusCode: resp.StatusCode}
	}

	return c.readStream(resp, onDelta)
}

// LLMHTTPError is a non-200 response from the provider.
type LLMHTTPError struct {
	StatusCode int
}

func (e *LLMHTTPError) Error() string {
	return fmt.Sprintf("coach: llm provider returned status %d", e.StatusCode)
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string         `json:"content"`
			ToolCalls []wireToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *OpenAICompatibleClient) readStream(resp *http.Response, onDelta func(string)) (*CompletionResult, error) {
	result := &CompletionResult{}
	toolArgs := map[int]*strings.Builder{}
	toolMeta := map[int]*ToolCall{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // tolerate malformed keep-alive chunks
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				result.Content += choice.Delta.Content
				if onDelta != nil {
					onDelta(choice.Delta.Content)
				}
			}
			for _, call := range choice.Delta.ToolCalls {
				index := call.Index
				if toolArgs[index] == nil {
					toolArgs[index] = &strings.Builder{}
					toolMeta[index] = &ToolCall{ID: call.ID, Name: call.Function.Name}
				}
				if call.ID != "" {
					toolMeta[index].ID = call.ID
				}
				if call.Function.Name != "" {
					toolMeta[index].Name = call.Function.Name
				}
				toolArgs[index].WriteString(call.Function.Arguments)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("coach: read llm stream: %w", err)
	}

	for i := 0; i < len(toolMeta); i++ {
		if call, ok := toolMeta[i]; ok && call.Name != "" {
			call.Arguments = toolArgs[i].String()
			result.ToolCalls = append(result.ToolCalls, *call)
		}
	}
	return result, nil
}
