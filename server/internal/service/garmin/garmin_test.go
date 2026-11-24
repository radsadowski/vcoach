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

	t.Run("decodes the wrapped response shapes of the health endpoints", func(t *testing.T) {
		r := require.New(t)

		// given: the wire shapes Garmin actually returns (v2 dailysummaries
		// wraps in {"data": ...}, HRV in {"hrvSummaries": ...}, weight in
		// {"dailyWeightSummaries": [{...latestWeight: {weight}}]}, and HR
		// zones come as an array per sport)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch req.URL.Path {
			case tokenPath:
				fmt.Fprint(w, testJWT(t, time.Now().Add(2*time.Hour)))
			case "/usersummary-service/v2/dailysummaries":
				fmt.Fprint(w, `{"data":[{"calendarDate":"2026-11-20","heartRate":{"restingValue":52},"movement":{"steps":{"value":9800,"goal":13240}}}]}`)
			case "/hrv-service/hrv/daily/2026-11-20/2026-11-20":
				fmt.Fprint(w, `{"hrvSummaries":[{"calendarDate":"2026-11-20","lastNightAvg":64}]}`)
			case "/weight-service/weight/range/2026-11-20/2026-11-20":
				fmt.Fprint(w, `{"dailyWeightSummaries":[{"summaryDate":"2026-11-20","latestWeight":{"weight":72.4}}]}`)
			case "/biometric-service/heartRateZones/v2":
				fmt.Fprint(w, `[{"sport":"RUNNING","maxHeartRateUsed":185},{"sport":"DEFAULT","maxHeartRateUsed":190,"restingHeartRateUsed":50,"zone1Floor":100,"zone2Floor":120,"zone3Floor":145,"zone4Floor":165}]`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()
		client := NewHTTPClientForURL(srv.URL, "key")

		// when / then
		summaries, err := client.DailySummaries(t.Context(), "2026-11-20", "2026-11-20")
		r.NoError(err)
		r.Len(summaries, 1)
		r.InDelta(52, summaries[0].HeartRate.RestingValue, 0.001)
		r.InDelta(9800, summaries[0].Movement.Steps.Value, 0.001)

		hrv, err := client.HRVDaily(t.Context(), "2026-11-20", "2026-11-20")
		r.NoError(err)
		r.Len(hrv, 1)
		r.Equal(64, hrv[0].LastNightAvg)

		weights, err := client.WeightRange(t.Context(), "2026-11-20", "2026-11-20")
		r.NoError(err)
		r.Len(weights, 1)
		r.Equal("2026-11-20", weights[0].Date)
		r.InDelta(72.4, weights[0].Weight, 0.001)

		zones, err := client.HeartRateZones(t.Context())
		r.NoError(err)
		r.Equal("DEFAULT", zones.Sport)
		r.Equal(190, zones.MaxHeartRate)
		r.Equal(50, zones.RestingHR)
		r.Equal(120, zones.Zone2Floor)
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
