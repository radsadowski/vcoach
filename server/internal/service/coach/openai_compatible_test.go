package coach

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAICompatibleClient(t *testing.T) {
	t.Parallel()

	t.Run("streams content deltas and accumulates the result", func(t *testing.T) {
		r := require.New(t)

		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n")
			flusher.Flush()
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\", \"},\"finish_reason\":null}]}\n\n")
			flusher.Flush()
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"athlete!\"},\"finish_reason\":\"stop\"}]}\n\n")
			flusher.Flush()
			fmt.Fprint(w, "data: [DONE]\n\n")
		}))
		defer srv.Close()

		client := NewOpenAICompatibleClient(srv.URL, "key", "test-model")
		var deltas []string

		// when
		result, err := client.Complete(t.Context(), CompletionRequest{SystemPrompt: "sys"}, func(delta string) {
			deltas = append(deltas, delta)
		})

		// then
		r.NoError(err)
		r.Equal("Hello, athlete!", result.Content)
		r.Empty(result.ToolCalls)
		r.Equal([]string{"Hello", ", ", "athlete!"}, deltas)
	})

	t.Run("accumulates tool calls split across chunks", func(t *testing.T) {
		r := require.New(t)

		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)
			fmt.Fprint(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"create_goal","arguments":""}}]}}]}`+"\n\n")
			flusher.Flush()
			fmt.Fprint(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"activityType\":\"boulder\","}}]}}]}`+"\n\n")
			flusher.Flush()
			fmt.Fprint(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"targetMetric\":\"grade\",\"targetValue\":\"7C\"}"}}]},"finish_reason":"tool_calls"}]}`+"\n\n")
			flusher.Flush()
			fmt.Fprint(w, "data: [DONE]\n\n")
		}))
		defer srv.Close()

		client := NewOpenAICompatibleClient(srv.URL, "key", "test-model")

		// when
		result, err := client.Complete(t.Context(), CompletionRequest{}, nil)

		// then
		r.NoError(err)
		r.Empty(result.Content)
		r.Len(result.ToolCalls, 1)
		r.Equal("call_1", result.ToolCalls[0].ID)
		r.Equal("create_goal", result.ToolCalls[0].Name)
		r.JSONEq(`{"activityType":"boulder","targetMetric":"grade","targetValue":"7C"}`, result.ToolCalls[0].Arguments)
	})

	t.Run("surfaces provider errors", func(t *testing.T) {
		r := require.New(t)

		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()
		client := NewOpenAICompatibleClient(srv.URL, "bad-key", "test-model")

		// when
		_, err := client.Complete(t.Context(), CompletionRequest{}, nil)

		// then
		var httpErr *LLMHTTPError
		r.ErrorAs(err, &httpErr)
		r.Equal(http.StatusUnauthorized, httpErr.StatusCode)
	})

	t.Run("refuses to run unconfigured", func(t *testing.T) {
		r := require.New(t)

		// given
		client := NewOpenAICompatibleClient("", "", "")

		// when
		_, err := client.Complete(t.Context(), CompletionRequest{}, nil)

		// then
		r.ErrorContains(err, "not configured")
	})
}
