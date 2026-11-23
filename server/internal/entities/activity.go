package entities

import (
	"time"

	"github.com/google/uuid"
)

// Activity is a workout synced from Garmin. Raw payload is retained for
// re-normalization when the connector changes.
type Activity struct {
	ID          uuid.UUID `db:"id" json:"id"`
	AthleteID   uuid.UUID `db:"athlete_id" json:"athleteId"`
	ExternalID  string    `db:"external_id" json:"externalId"`
	Sport       string    `db:"sport" json:"sport"`
	SportFamily string    `db:"sport_family" json:"sportFamily"`
	StartTime   time.Time `db:"start_time" json:"startTime"`
	DurationSec int       `db:"duration_sec" json:"durationSec"`
	DistanceM   *float64  `db:"distance_m" json:"distanceM,omitempty"`
	AvgHR       *int      `db:"avg_hr" json:"avgHR,omitempty"`
	MaxHR       *int      `db:"max_hr" json:"maxHR,omitempty"`
	VO2Max      *float64  `db:"vo2max" json:"vo2max,omitempty"`
	Raw         []byte    `db:"raw" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// DailyHealth holds one day of Garmin health metrics.
type DailyHealth struct {
	ID          uuid.UUID `db:"id" json:"id"`
	AthleteID   uuid.UUID `db:"athlete_id" json:"athleteId"`
	Day         Date      `db:"day" json:"day"`
	SleepScore  *int      `db:"sleep_score" json:"sleepScore,omitempty"`
	SleepHours  *float64  `db:"sleep_hours" json:"sleepHours,omitempty"`
	HRVMS       *int      `db:"hrv_ms" json:"hrvMs,omitempty"`
	RestingHR   *int      `db:"resting_hr" json:"restingHr,omitempty"`
	BodyBattery *int      `db:"body_battery" json:"bodyBattery,omitempty"`
	StressAvg   *int      `db:"stress_avg" json:"stressAvg,omitempty"`
	Steps       *int      `db:"steps" json:"steps,omitempty"`
	WeightKg    *float64  `db:"weight_kg" json:"weightKg,omitempty"`
	Raw         []byte    `db:"raw" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}
