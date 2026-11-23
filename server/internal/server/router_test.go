package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/server"
	chatsvc "github.com/radosny/vcoach/server/internal/service/chat"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	goalsvc "github.com/radosny/vcoach/server/internal/service/goals"
	plansvc "github.com/radosny/vcoach/server/internal/service/plans"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
	"github.com/radosny/vcoach/server/internal/testing/database"
	climbbuilder "github.com/radosny/vcoach/server/internal/testing/entities/climbing"
	goalbuilder "github.com/radosny/vcoach/server/internal/testing/entities/goal"
	planbuilder "github.com/radosny/vcoach/server/internal/testing/entities/plan"
)

// newTestServer builds the full router with real services over a throwaway
// Postgres, mirroring the production wiring.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := database.SetupDB(t)
	repo := repository.New(db)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := server.NewRouter(server.Deps{
		Log:      log,
		Athletes: repo,
		Profile:  profilesvc.New(repo, log),
		Goals:    goalsvc.New(repo),
		Plans:    plansvc.New(repo),
		Climbing: climbingsvc.New(repo),
		Chat:     chatsvc.New(repo),
	})
	return httptest.NewServer(handler)
}

func TestAPI(t *testing.T) {
	t.Parallel()

	// given
	ts := newTestServer(t)
	defer ts.Close()

	t.Run("profile basics round-trip", func(t *testing.T) {
		r := require.New(t)

		// given
		birthYear := 1990
		sex := "male"
		weightKg := 72.5
		running := "intermediate"
		climbing := "advanced"

		// when
		body, _ := json.Marshal(map[string]any{
			"birthYear":          birthYear,
			"sex":                sex,
			"weightKg":           weightKg,
			"runningExperience":  running,
			"climbingExperience": climbing,
		})
		resp := doRequest(t, ts, http.MethodPut, "/api/profile", body)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		var athlete map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&athlete))
		r.Equal(float64(birthYear), athlete["birthYear"])
		r.Equal(sex, athlete["sex"])
		r.Equal(weightKg, athlete["weightKg"])

		// when: read back
		resp = doRequest(t, ts, http.MethodGet, "/api/profile", nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		athlete = map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&athlete))
		r.Equal(running, athlete["runningExperience"])
		r.Equal(climbing, athlete["climbingExperience"])
	})

	t.Run("profile rejects invalid sex", func(t *testing.T) {
		r := require.New(t)

		// when
		body, _ := json.Marshal(map[string]any{"sex": "alien"})
		resp := doRequest(t, ts, http.MethodPut, "/api/profile", body)

		// then
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("goal create, list, patch, delete", func(t *testing.T) {
		r := require.New(t)

		// when
		body, _ := json.Marshal(goalbuilder.NewInput(goalbuilder.WithBoulderGrade("7C"), goalbuilder.WithPrimary()))
		resp := doRequest(t, ts, http.MethodPost, "/api/goals", body)

		// then
		r.Equal(http.StatusCreated, resp.StatusCode)
		var goal map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&goal))
		goalID := goal["id"].(string)
		r.Equal("7C", goal["targetValue"])
		r.Equal(true, goal["isPrimary"])

		// when: invalid goal
		body, _ = json.Marshal(goalbuilder.NewInput(goalbuilder.WithBoulderGrade("9Z")))
		resp = doRequest(t, ts, http.MethodPost, "/api/goals", body)

		// then
		r.Equal(http.StatusBadRequest, resp.StatusCode)

		// when: patch status
		body, _ = json.Marshal(map[string]any{"status": "achieved"})
		resp = doRequest(t, ts, http.MethodPatch, "/api/goals/"+goalID, body)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		goal = map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&goal))
		r.Equal("achieved", goal["status"])

		// when: list
		resp = doRequest(t, ts, http.MethodGet, "/api/goals", nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		var goals []map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&goals))
		r.Len(goals, 1)

		// when: delete
		resp = doRequest(t, ts, http.MethodDelete, "/api/goals/"+goalID, nil)

		// then
		r.Equal(http.StatusNoContent, resp.StatusCode)
		resp = doRequest(t, ts, http.MethodGet, "/api/goals", nil)
		goals = []map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&goals))
		r.Empty(goals)
	})

	t.Run("goal with invalid id returns 400", func(t *testing.T) {
		r := require.New(t)

		// when: PATCH parses the id before touching the service
		body, _ := json.Marshal(map[string]any{"notes": "x"})
		resp := doRequest(t, ts, http.MethodPatch, "/api/goals/not-a-uuid", body)

		// then
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("plan creation versions and archives", func(t *testing.T) {
		r := require.New(t)

		// when
		body, _ := json.Marshal(planbuilder.NewInput())
		resp := doRequest(t, ts, http.MethodPost, "/api/plans", body)

		// then
		r.Equal(http.StatusCreated, resp.StatusCode)
		var plan map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&plan))
		firstID := plan["id"].(string)
		firstVersion := plan["version"].(float64)
		r.Equal("active", plan["status"])

		// when: create successor
		resp = doRequest(t, ts, http.MethodPost, "/api/plans", body)

		// then
		r.Equal(http.StatusCreated, resp.StatusCode)
		plan = map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&plan))
		r.Equal(firstVersion+1, plan["version"])

		// when: fetch the archived predecessor with sessions
		resp = doRequest(t, ts, http.MethodGet, "/api/plans/"+firstID, nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		plan = map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&plan))
		r.Equal("archived", plan["status"])
		weeks := plan["weeks"].([]any)
		r.Len(weeks, 4)
	})

	t.Run("climbing session CRUD", func(t *testing.T) {
		r := require.New(t)

		// when
		body, _ := json.Marshal(climbbuilder.NewInput())
		resp := doRequest(t, ts, http.MethodPost, "/api/climbing-sessions", body)

		// then
		r.Equal(http.StatusCreated, resp.StatusCode)
		var session map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&session))
		sessionID := session["id"].(string)
		r.Equal("boulder", session["discipline"])

		// when: list
		resp = doRequest(t, ts, http.MethodGet, "/api/climbing-sessions", nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		var sessions []map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&sessions))
		r.Len(sessions, 1)

		// when: update
		attempts := 20
		body, _ = json.Marshal(map[string]any{"attempts": attempts})
		resp = doRequest(t, ts, http.MethodPut, "/api/climbing-sessions/"+sessionID, body)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		session = map[string]any{}
		r.NoError(json.NewDecoder(resp.Body).Decode(&session))
		r.Equal(float64(attempts), session["attempts"])

		// when: delete
		resp = doRequest(t, ts, http.MethodDelete, "/api/climbing-sessions/"+sessionID, nil)

		// then
		r.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("chat session create and message history", func(t *testing.T) {
		r := require.New(t)

		// when
		resp := doRequest(t, ts, http.MethodPost, "/api/chat-sessions", []byte(`{"title":"First chat"}`))

		// then
		r.Equal(http.StatusCreated, resp.StatusCode)
		var session map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&session))
		sessionID := session["id"].(string)
		r.Equal("First chat", session["title"])

		// when: empty message history
		resp = doRequest(t, ts, http.MethodGet, "/api/chat-sessions/"+sessionID+"/messages", nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		var messages []map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&messages))
		r.Empty(messages)

		// when: list sessions
		resp = doRequest(t, ts, http.MethodGet, "/api/chat-sessions", nil)

		// then
		r.Equal(http.StatusOK, resp.StatusCode)
		var sessions []map[string]any
		r.NoError(json.NewDecoder(resp.Body).Decode(&sessions))
		r.Len(sessions, 1)
	})
}

func doRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	require.NoError(t, err)
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}
