package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/server"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	// given
	log := discardLogger()
	router := server.NewRouter(server.Deps{Log: log})
	srv := httptest.NewServer(router)
	defer srv.Close()

	t.Run("healthz returns ok", func(t *testing.T) {
		r := require.New(t)

		// when
		resp, err := http.Get(srv.URL + "/healthz")

		// then
		r.NoError(err)
		defer resp.Body.Close()
		r.Equal(http.StatusOK, resp.StatusCode)
		r.Equal("application/json", resp.Header.Get("Content-Type"))

		var body map[string]string
		r.NoError(json.NewDecoder(resp.Body).Decode(&body))
		r.Equal("ok", body["status"])
	})

	t.Run("unknown path returns 404", func(t *testing.T) {
		r := require.New(t)

		// when
		resp, err := http.Get(srv.URL + "/nope")

		// then
		r.NoError(err)
		defer resp.Body.Close()
		r.Equal(http.StatusNotFound, resp.StatusCode)
	})

	t.Run("method not allowed returns 405", func(t *testing.T) {
		r := require.New(t)

		// when
		resp, err := http.Post(srv.URL+"/healthz", "application/json", nil)

		// then
		r.NoError(err)
		defer resp.Body.Close()
		r.Equal(http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
