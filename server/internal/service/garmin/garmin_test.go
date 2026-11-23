package garmin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("exchanges the API key for a bearer token and fetches activities", func(t *testing.T) {
		r := require.New(t)

		// given
		jwt := testJWT(t, time.Now().Add(2*time.Hour))
		var authHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case tokenPath:
				authHeader = req.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, jwt)
			case "/activitylist-service/activities/search/activities":
				if req.Header.Get("Authorization") != "Bearer "+jwt {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `[{"activityId":42,"activityName":"Morning run","activityType":{"typeKey":"running"},"startTimeLocal":"2026-11-20 07:30:00.0","duration":2400,"distance":8000,"averageHR":150,"maxHR":172}]`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := NewHTTPClientForURL(srv.URL, "test-api-key")

		// when
		activities, err := client.ListActivities(t.Context(), "2026-11-01", "2026-11-23")

		// then
		r.NoError(err)
		r.Equal("ApiKey test-api-key", authHeader)
		r.Len(activities, 1)
		r.Equal(int64(42), activities[0].ActivityID)
		r.Equal("running", activities[0].ActivityType.TypeKey)
		r.Equal(2400.0, activities[0].Duration)
	})

	t.Run("surfaces unauthorized errors from the token exchange", func(t *testing.T) {
		r := require.New(t)

		// given
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()
		client := NewHTTPClientForURL(srv.URL, "expired-key")

		// when
		_, err := client.ListActivities(t.Context(), "2026-11-01", "2026-11-23")

		// then
		var unauthorized *UnauthorizedError
		r.ErrorAs(err, &unauthorized)
	})

	t.Run("parses startTimeLocal and maps sport families", func(t *testing.T) {
		r := require.New(t)

		// given / when
		parsed, err := ParseStartTime("2026-11-23 07:30:00.0")

		// then
		r.NoError(err)
		r.Equal(2026, parsed.Year())
		r.Equal(7, parsed.Hour())

		r.Equal("run", SportFamily("running"))
		r.Equal("run", SportFamily("trail_running"))
		r.Equal("climb", SportFamily("bouldering"))
		r.Equal("climb", SportFamily("indoor_climbing"))
		r.Equal("bike", SportFamily("cycling"))
		r.Equal("other", SportFamily("yoga"))
	})
}

func testJWT(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]int64{"exp": expiresAt.Unix()})
	require.NoError(t, err)
	encode := func(part []byte) string {
		return base64.RawURLEncoding.EncodeToString(part)
	}
	return encode(header) + "." + encode(payload) + ".signature"
}
