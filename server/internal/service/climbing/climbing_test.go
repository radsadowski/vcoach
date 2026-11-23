package climbing_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/radosny/vcoach/server/internal/entities"
	domainerrors "github.com/radosny/vcoach/server/internal/errors"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/service/climbing"
	"github.com/radosny/vcoach/server/internal/testing/database"
	climbbuilder "github.com/radosny/vcoach/server/internal/testing/entities/climbing"
)

func mustUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}

func TestService(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// given
	db := database.SetupDB(t)
	repo := repository.New(db)
	svc := climbing.New(repo)
	ctx := context.Background()
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	r.NoError(err)

	t.Run("creates a valid boulder session", func(t *testing.T) {
		r := require.New(t)

		// when
		session, err := svc.Create(ctx, athleteID, climbbuilder.NewInput())

		// then
		r.NoError(err)
		r.Equal("boulder", session.Discipline)
		r.Equal("6A", session.HardestSent)
		r.Equal("6B", session.HardestAttempted)
		r.Equal(12, session.Attempts)
	})

	t.Run("validates session input", func(t *testing.T) {
		tests := map[string]struct {
			input   entities.CreateClimbingSessionInput
			wantErr string
		}{
			"invalid boulder grade": {
				input:   climbbuilder.NewInput(climbbuilder.WithSent("7Z")),
				wantErr: "hardestSent",
			},
			"sent harder than attempted": {
				input:   climbbuilder.NewInput(climbbuilder.WithSent("7A"), climbbuilder.WithAttempted("6B")),
				wantErr: "cannot be harder",
			},
			"rpe out of range": {
				input:   climbbuilder.NewInput(climbbuilder.WithRPE(11)),
				wantErr: "rpe",
			},
			"unknown discipline": {
				input:   climbbuilder.NewInput(climbbuilder.WithDiscipline("deep water solo")),
				wantErr: "discipline",
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				r := require.New(t)

				// when
				_, err := svc.Create(ctx, athleteID, tt.input)

				// then
				var validationErr *domainerrors.ValidationError
				r.ErrorAs(err, &validationErr)
				r.Contains(err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("update re-validates merged grades", func(t *testing.T) {
		r := require.New(t)

		// given
		session, err := svc.Create(ctx, athleteID, climbbuilder.NewInput())
		r.NoError(err)

		// when: switch to lead without fixing the grades
		discipline := "lead"
		_, err = svc.Update(ctx, athleteID, session.ID, entities.UpdateClimbingSessionInput{Discipline: &discipline})

		// then: boulder grades are invalid on the lead scale
		var validationErr *domainerrors.ValidationError
		r.ErrorAs(err, &validationErr)

		// when: fix grades for lead
		sent, attempted := "7a", "7a+"
		updated, err := svc.Update(ctx, athleteID, session.ID, entities.UpdateClimbingSessionInput{
			Discipline:       &discipline,
			HardestSent:      &sent,
			HardestAttempted: &attempted,
		})

		// then
		r.NoError(err)
		r.Equal("lead", updated.Discipline)
		r.Equal("7a", updated.HardestSent)
	})

	t.Run("lists sessions with date filter", func(t *testing.T) {
		r := require.New(t)

		// given
		before, err := svc.List(ctx, athleteID, nil, nil)
		r.NoError(err)
		session, err := svc.Create(ctx, athleteID, climbbuilder.NewInput())
		r.NoError(err)

		// when
		sessions, err := svc.List(ctx, athleteID, nil, nil)

		// then
		r.NoError(err)
		r.Len(sessions, len(before)+1)
		r.Contains(sessionIDs(sessions), session.ID.String())

		// when: filter to a window that excludes the session
		from, to := "2000-01-01", "2000-01-31"
		sessions, err = svc.List(ctx, athleteID, &from, &to)

		// then
		r.NoError(err)
		r.Empty(sessions)
	})

	t.Run("unknown session returns NotFoundError", func(t *testing.T) {
		r := require.New(t)

		// when
		_, err := svc.Get(ctx, athleteID, mustUUID("00000000-0000-0000-0000-000000000002"))

		// then
		var notFoundErr *domainerrors.NotFoundError
		r.ErrorAs(err, &notFoundErr)
	})

	t.Run("deletes a session", func(t *testing.T) {
		r := require.New(t)

		// given
		session, err := svc.Create(ctx, athleteID, climbbuilder.NewInput())
		r.NoError(err)

		// when
		err = svc.Delete(ctx, athleteID, session.ID)

		// then
		r.NoError(err)
		var notFoundErr *domainerrors.NotFoundError
		_, err = svc.Get(ctx, athleteID, session.ID)
		r.ErrorAs(err, &notFoundErr)
	})
}

func sessionIDs(sessions []entities.ClimbingSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID.String())
	}
	return ids
}
