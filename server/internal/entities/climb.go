package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ClimbingSession is a manually logged climbing workout. Goal progress for
// boulder/lead goals is computed from these records.
type ClimbingSession struct {
	ID               uuid.UUID `db:"id" json:"id"`
	AthleteID        uuid.UUID `db:"athlete_id" json:"athleteId"`
	SessionDate      Date      `db:"session_date" json:"sessionDate"`
	Venue            string    `db:"venue" json:"venue"`
	Discipline       string    `db:"discipline" json:"discipline"`
	HardestSent      string    `db:"hardest_sent" json:"hardestSent"`
	HardestAttempted string    `db:"hardest_attempted" json:"hardestAttempted"`
	Attempts         int       `db:"attempts" json:"attempts"`
	RPE              *int      `db:"rpe" json:"rpe,omitempty"`
	Notes            string    `db:"notes" json:"notes"`
	CreatedAt        time.Time `db:"created_at" json:"createdAt"`
}

// CreateClimbingSessionInput is the validated payload for logging a session.
type CreateClimbingSessionInput struct {
	SessionDate      Date   `json:"sessionDate"`
	Venue            string `json:"venue"`
	Discipline       string `json:"discipline"`
	HardestSent      string `json:"hardestSent"`
	HardestAttempted string `json:"hardestAttempted"`
	Attempts         int    `json:"attempts"`
	RPE              *int   `json:"rpe,omitempty"`
	Notes            string `json:"notes"`
}

// UpdateClimbingSessionInput is the validated payload for editing a session.
type UpdateClimbingSessionInput struct {
	SessionDate      *Date   `json:"sessionDate,omitempty"`
	Venue            *string `json:"venue,omitempty"`
	Discipline       *string `json:"discipline,omitempty"`
	HardestSent      *string `json:"hardestSent,omitempty"`
	HardestAttempted *string `json:"hardestAttempted,omitempty"`
	Attempts         *int    `json:"attempts,omitempty"`
	RPE              *int    `json:"rpe,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

// Boulder grades on the Font scale, in ascending order. The slice index is
// the rank used to compare grades and compute goal progress.
var boulderGrades = []string{
	"3", "4", "4+", "5", "5+", "6A", "6A+", "6B", "6B+", "6C", "6C+",
	"7A", "7A+", "7B", "7B+", "7C", "7C+", "8A", "8A+", "8B", "8B+",
	"8C", "8C+", "9A",
}

// Lead grades on the Font route scale, in ascending order.
var leadGrades = []string{
	"4a", "4b", "4c", "5a", "5a+", "5b", "5b+", "5c", "5c+",
	"6a", "6a+", "6b", "6b+", "6c", "6c+",
	"7a", "7a+", "7b", "7b+", "7c", "7c+",
	"8a", "8a+", "8b", "8b+", "8c", "8c+",
	"9a", "9a+", "9b", "9b+", "9c",
}

// BoulderGradeRank returns the rank of a Font boulder grade (0 = easiest).
func BoulderGradeRank(grade string) (int, error) {
	return gradeRank(boulderGrades, grade, "boulder")
}

// LeadGradeRank returns the rank of a Font lead grade (0 = easiest).
func LeadGradeRank(grade string) (int, error) {
	return gradeRank(leadGrades, grade, "lead")
}

// ValidBoulderGrade reports whether the grade is a known Font boulder grade.
func ValidBoulderGrade(grade string) bool {
	_, err := BoulderGradeRank(grade)
	return err == nil
}

// ValidLeadGrade reports whether the grade is a known Font lead grade.
func ValidLeadGrade(grade string) bool {
	_, err := LeadGradeRank(grade)
	return err == nil
}

func gradeRank(scale []string, grade string, discipline string) (int, error) {
	for i, g := range scale {
		if g == grade {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown %s grade %q", discipline, grade)
}
