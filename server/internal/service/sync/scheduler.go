package sync

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Run is the background scheduler: it syncs on the given interval until the
// context is cancelled. The first run starts shortly after boot.
func (s *Service) Run(ctx context.Context, athleteID uuid.UUID, interval time.Duration) {
	s.log.Info("garmin scheduler started", "interval", interval)
	defer s.log.Info("garmin scheduler stopped")

	const firstDelay = 30 * time.Second
	select {
	case <-ctx.Done():
		return
	case <-time.After(firstDelay):
	}

	s.syncAndLog(ctx, athleteID)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAndLog(ctx, athleteID)
		}
	}
}

func (s *Service) syncAndLog(ctx context.Context, athleteID uuid.UUID) {
	result, err := s.SyncNow(ctx, athleteID)
	if err != nil {
		s.log.Warn("scheduled sync finished with errors",
			"error", err,
			"activitiesSynced", result.ActivitiesSynced,
			"healthDaysSynced", result.HealthDaysSynced)
		return
	}
	s.log.Info("scheduled sync finished",
		"activitiesSynced", result.ActivitiesSynced,
		"healthDaysSynced", result.HealthDaysSynced)
}
