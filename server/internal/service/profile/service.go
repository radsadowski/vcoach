// Package profile implements athlete profile basics: manual fields with
// validation. Garmin-derived metrics are added on top in service/profile.
package profile

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
)

// nowUTC is the clock used for derived-metric windows.
func nowUTC() time.Time {
	return time.Now().UTC()
}

// Service holds the profile business logic dependencies.
type Service struct {
	repo *repository.Repository
	log  *slog.Logger
}

// New builds a profile Service.
func New(repo *repository.Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Get returns the athlete profile basics.
func (s *Service) Get(ctx context.Context, athleteID uuid.UUID) (entities.Athlete, error) {
	athlete, err := s.repo.GetAthlete(ctx, athleteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Athlete{}, domainerrors.NewNotFoundError("athlete %s not found", athleteID)
		}
		return entities.Athlete{}, err
	}
	return athlete, nil
}

// UpdateBasics validates and updates manual profile basics.
func (s *Service) UpdateBasics(ctx context.Context, athleteID uuid.UUID, input entities.UpdateAthleteInput) (entities.Athlete, error) {
	if err := validateBasics(input); err != nil {
		return entities.Athlete{}, err
	}

	athlete, err := s.repo.UpdateAthleteBasics(ctx, athleteID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Athlete{}, domainerrors.NewNotFoundError("athlete %s not found", athleteID)
		}
		return entities.Athlete{}, err
	}
	return athlete, nil
}

func validateBasics(input entities.UpdateAthleteInput) error {
	if input.BirthYear != nil {
		currentYear := time.Now().UTC().Year()
		if *input.BirthYear < 1900 || *input.BirthYear > currentYear {
			return domainerrors.NewValidationError("birthYear must be between 1900 and %d, got %d", currentYear, *input.BirthYear)
		}
	}
	if input.Sex != nil && !validSex(*input.Sex) {
		return domainerrors.NewValidationError("sex must be one of male, female, other, got %q", *input.Sex)
	}
	if input.WeightKg != nil && (*input.WeightKg < 30 || *input.WeightKg > 300) {
		return domainerrors.NewValidationError("weightKg must be between 30 and 300, got %.1f", *input.WeightKg)
	}
	if input.RunningExperience != nil && !validExperience(*input.RunningExperience) {
		return domainerrors.NewValidationError("runningExperience must be one of beginner, intermediate, advanced, got %q", *input.RunningExperience)
	}
	if input.ClimbingExperience != nil && !validExperience(*input.ClimbingExperience) {
		return domainerrors.NewValidationError("climbingExperience must be one of beginner, intermediate, advanced, got %q", *input.ClimbingExperience)
	}
	return nil
}

func validSex(sex string) bool {
	return sex == entities.SexMale || sex == entities.SexFemale || sex == entities.SexOther
}

func validExperience(level string) bool {
	return level == entities.ExperienceBeginner ||
		level == entities.ExperienceIntermediate ||
		level == entities.ExperienceAdvanced
}
