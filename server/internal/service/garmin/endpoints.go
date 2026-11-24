package garmin

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// activityPageSize is the page size for the activity search endpoint. The
// service caps at roughly 45-50 per page; pages are fetched until empty.
const activityPageSize = 40

// ListActivities returns every activity between the inclusive dates
// (YYYY-MM-DD), paging through the search endpoint.
func (c *HTTPClient) ListActivities(ctx context.Context, startDate, endDate string) ([]Activity, error) {
	var all []Activity
	for offset := 0; ; offset += activityPageSize {
		params := url.Values{}
		params.Set("start", strconv.Itoa(offset))
		params.Set("limit", strconv.Itoa(activityPageSize))
		params.Set("startDate", startDate)
		params.Set("endDate", endDate)

		var page []Activity
		if err := c.get(ctx, "/activitylist-service/activities/search/activities?"+params.Encode(), &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < activityPageSize {
			return all, nil
		}
	}
}

// DailySummaries returns per-day wellness metrics (resting HR, steps).
func (c *HTTPClient) DailySummaries(ctx context.Context, startDate, endDate string) ([]DailySummary, error) {
	var response struct {
		Data []DailySummary `json:"data"`
	}
	if err := c.get(ctx, "/usersummary-service/v2/dailysummaries?startDate="+startDate+"&endDate="+endDate, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

// StressDaily returns the average stress level per day.
func (c *HTTPClient) StressDaily(ctx context.Context, startDate, endDate string) ([]DailyStress, error) {
	var out []DailyStress
	err := c.get(ctx, "/usersummary-service/stats/stress/daily/"+startDate+"/"+endDate, &out)
	return out, err
}

// HRVDaily returns nightly HRV averages per day.
func (c *HTTPClient) HRVDaily(ctx context.Context, startDate, endDate string) ([]DailyHRV, error) {
	var response struct {
		HRVSummaries []DailyHRV `json:"hrvSummaries"`
	}
	if err := c.get(ctx, "/hrv-service/hrv/daily/"+startDate+"/"+endDate, &response); err != nil {
		return nil, err
	}
	return response.HRVSummaries, nil
}

// SleepSummaries returns one sleep summary per night.
func (c *HTTPClient) SleepSummaries(ctx context.Context, startDate, endDate string) ([]SleepSummary, error) {
	var out []SleepSummary
	err := c.get(ctx, "/sleep-service/sleep/dailySleepSummariesByDate?startDate="+startDate+"&endDate="+endDate, &out)
	return out, err
}

// BodyBatteryDaily returns the end-of-day body battery charge.
func (c *HTTPClient) BodyBatteryDaily(ctx context.Context, startDate, endDate string) ([]DailyBodyBattery, error) {
	var out []DailyBodyBattery
	err := c.get(ctx, "/wellness-service/wellness/bodyBattery/reports/daily?startDate="+startDate+"&endDate="+endDate, &out)
	return out, err
}

// WeightRange returns weight measurements in the inclusive date range.
func (c *HTTPClient) WeightRange(ctx context.Context, startDate, endDate string) ([]WeightMeasurement, error) {
	var response struct {
		DailyWeightSummaries []struct {
			SummaryDate  string `json:"summaryDate"`
			LatestWeight struct {
				Weight float64 `json:"weight"`
			} `json:"latestWeight"`
		} `json:"dailyWeightSummaries"`
	}
	if err := c.get(ctx, "/weight-service/weight/range/"+startDate+"/"+endDate+"?includeAll=true", &response); err != nil {
		return nil, err
	}
	measurements := make([]WeightMeasurement, 0, len(response.DailyWeightSummaries))
	for _, summary := range response.DailyWeightSummaries {
		if summary.LatestWeight.Weight > 0 {
			measurements = append(measurements, WeightMeasurement{
				Date:   summary.SummaryDate,
				Weight: summary.LatestWeight.Weight,
			})
		}
	}
	return measurements, nil
}

// HeartRateZones returns the athlete's default heart-rate zones. Garmin
// returns one zone set per sport; DEFAULT applies globally.
func (c *HTTPClient) HeartRateZones(ctx context.Context) (HeartRateZones, error) {
	var all []HeartRateZones
	if err := c.get(ctx, "/biometric-service/heartRateZones/v2", &all); err != nil {
		return HeartRateZones{}, err
	}
	for _, zones := range all {
		if zones.Sport == "DEFAULT" {
			return zones, nil
		}
	}
	if len(all) > 0 {
		return all[0], nil
	}
	return HeartRateZones{}, nil
}

// VO2MaxDaily returns per-day VO2max estimates in the inclusive date range.
func (c *HTTPClient) VO2MaxDaily(ctx context.Context, startDate, endDate string) ([]VO2MaxMeasurement, error) {
	var out []VO2MaxMeasurement
	err := c.get(ctx, "/metrics-service/metrics/maxmet/daily/"+startDate+"/"+endDate, &out)
	return out, err
}

// ParseStartTime parses Garmin's startTimeLocal format
// ("2026-11-23 07:30:00.0") as a UTC timestamp.
func ParseStartTime(value string) (time.Time, error) {
	trimmed, _, _ := strings.Cut(value, ".")
	parsed, err := time.Parse("2006-01-02 15:04:05", trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("garmin: parse startTimeLocal %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

// SportFamily maps a Garmin typeKey to the sport family used for compliance
// matching and goal progress. Conservative by design: unmapped types land in
// "other" and never produce false matches.
func SportFamily(typeKey string) string {
	switch typeKey {
	case "running", "indoor_running", "treadmill_running", "trail_running",
		"track_running", "street_running", "ultra_running", "virtual_run":
		return "run"
	case "bouldering", "indoor_climbing", "rock_climbing", "ice_climbing",
		"indoor_bouldering", "deep_water_solo":
		return "climb"
	case "cycling", "indoor_cycling", "virtual_ride", "road_biking",
		"mountain_biking", "gravel_cycling", "track_cycling":
		return "bike"
	default:
		return "other"
	}
}
