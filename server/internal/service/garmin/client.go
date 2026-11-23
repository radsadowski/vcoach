// Package garmin is the only module that knows Garmin's Personal API wire
// format: token exchange, endpoint paths and response decoding. Everything
// downstream consumes the decoded types.
package garmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BaseURL is the Garmin Personal API host.
const BaseURL = "https://personal.connectapi.garmin.com"

// Client reads the athlete's own data from the Garmin Personal API.
//
//go:generate mockgen -destination ./mock/mocks.go -package mock . Client
type Client interface {
	ListActivities(ctx context.Context, startDate, endDate string) ([]Activity, error)
	DailySummaries(ctx context.Context, startDate, endDate string) ([]DailySummary, error)
	StressDaily(ctx context.Context, startDate, endDate string) ([]DailyStress, error)
	HRVDaily(ctx context.Context, startDate, endDate string) ([]DailyHRV, error)
	SleepSummaries(ctx context.Context, startDate, endDate string) ([]SleepSummary, error)
	BodyBatteryDaily(ctx context.Context, startDate, endDate string) ([]DailyBodyBattery, error)
	WeightRange(ctx context.Context, startDate, endDate string) ([]WeightMeasurement, error)
	HeartRateZones(ctx context.Context) (HeartRateZones, error)
}

// Activity is one workout from the activity list.
type Activity struct {
	ActivityID     int64           `json:"activityId"`
	ActivityName   string          `json:"activityName"`
	ActivityType   ActivityTypeRef `json:"activityType"`
	StartTimeLocal string          `json:"startTimeLocal"`
	Duration       float64         `json:"duration"`
	Distance       float64         `json:"distance"`
	AverageHR      float64         `json:"averageHR"`
	MaxHR          float64         `json:"maxHR"`
	VO2Max         float64         `json:"vO2MaxValue"`
}

// ActivityTypeRef is the activity type reference on an activity.
type ActivityTypeRef struct {
	TypeKey string `json:"typeKey"`
}

// DailySummary holds per-day wellness metrics (usersummary-service).
type DailySummary struct {
	CalendarDate     string `json:"calendarDate"`
	RestingHeartRate int    `json:"restingHeartRate"`
	Steps            int    `json:"steps"`
}

// DailyStress holds the average daily stress level.
type DailyStress struct {
	CalendarDate       string `json:"calendarDate"`
	AverageStressLevel int    `json:"averageStressLevel"`
}

// DailyHRV holds nightly HRV averages.
type DailyHRV struct {
	CalendarDate string `json:"calendarDate"`
	LastNightAvg int    `json:"lastNightAvg"`
}

// SleepSummary holds one night of sleep (sleep-service).
type SleepSummary struct {
	CalendarDate     string           `json:"calendarDate"`
	SleepTimeSeconds float64          `json:"sleepTimeSeconds"`
	SleepScores      SleepScoreBlocks `json:"sleepScores"`
}

// SleepScoreBlocks is the nested score structure of a sleep summary.
type SleepScoreBlocks struct {
	Overall SleepScoreValue `json:"overall"`
}

// SleepScoreValue is one sleep score component.
type SleepScoreValue struct {
	Value int `json:"value"`
}

// DailyBodyBattery holds the end-of-day body battery charge.
type DailyBodyBattery struct {
	CalendarDate string `json:"calendarDate"`
	Charged      int    `json:"charged"`
}

// WeightMeasurement is one weight sample.
type WeightMeasurement struct {
	Date   string  `json:"date"`
	Weight float64 `json:"weight"`
}

// HeartRateZones is the athlete's configured heart-rate zone set.
type HeartRateZones struct {
	MaxHeartRate int `json:"maxHeartRate"`
	RestingHR    int `json:"restingHr"`
	LactateHR    int `json:"lactateHr"`
	Zone1Floor   int `json:"zone1Floor"`
	Zone2Floor   int `json:"zone2Floor"`
	Zone3Floor   int `json:"zone3Floor"`
	Zone4Floor   int `json:"zone4Floor"`
}

// HTTPClient implements Client against the live Personal API.
type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client

	tokenMu sync.Mutex
	token   *BearerToken
}

// NewHTTPClient builds a client for the given personal API key.
func NewHTTPClient(apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL: BaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewHTTPClientForURL builds a client against a custom base URL (tests).
func NewHTTPClientForURL(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPClient) get(ctx context.Context, path string, out any) error {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("garmin GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return &UnauthorizedError{}
	}
	if resp.StatusCode != http.StatusOK {
		return &HTTPStatusError{Path: path, StatusCode: resp.StatusCode}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("garmin decode %s: %w", path, err)
	}
	return nil
}

// UnauthorizedError means the API key or bearer token was rejected — the
// key may have expired (Personal API keys live 30 days to 1 year).
type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
	return "garmin: unauthorized (API key expired or token rejected)"
}

// HTTPStatusError is any unexpected status from the Personal API.
type HTTPStatusError struct {
	Path       string
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("garmin: %s returned status %d", e.Path, e.StatusCode)
}
