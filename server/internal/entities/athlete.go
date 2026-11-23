// Package entities holds domain types shared across the server, service and
// repository layers.
package entities

import (
	"time"

	"github.com/google/uuid"
)

// Athlete is the single user of the system (personal-first). Manual basics
// live here; Garmin-derived metrics are computed on read.
type Athlete struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	Name               string    `db:"name" json:"name"`
	IsDefault          bool      `db:"is_default" json:"-"`
	BirthYear          *int      `db:"birth_year" json:"birthYear,omitempty"`
	Sex                *string   `db:"sex" json:"sex,omitempty"`
	WeightKg           *float64  `db:"weight_kg" json:"weightKg,omitempty"`
	RunningExperience  *string   `db:"running_experience" json:"runningExperience,omitempty"`
	ClimbingExperience *string   `db:"climbing_experience" json:"climbingExperience,omitempty"`
	CreatedAt          time.Time `db:"created_at" json:"createdAt"`
}

// Experience levels shared by running and climbing.
const (
	ExperienceBeginner     = "beginner"
	ExperienceIntermediate = "intermediate"
	ExperienceAdvanced     = "advanced"
)

// Sex values for the athlete profile.
const (
	SexMale   = "male"
	SexFemale = "female"
	SexOther  = "other"
)

// UpdateAthleteInput is the validated payload for editing profile basics.
// Nil fields are kept unchanged.
type UpdateAthleteInput struct {
	BirthYear          *int     `json:"birthYear"`
	Sex                *string  `json:"sex"`
	WeightKg           *float64 `json:"weightKg"`
	RunningExperience  *string  `json:"runningExperience"`
	ClimbingExperience *string  `json:"climbingExperience"`
}
