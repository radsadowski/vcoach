package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/radosny/vcoach/server/db/migrations"
	_ "github.com/radosny/vcoach/server/docs"
	"github.com/radosny/vcoach/server/internal/config"
	"github.com/radosny/vcoach/server/internal/repository"
	"github.com/radosny/vcoach/server/internal/server"
	chatsvc "github.com/radosny/vcoach/server/internal/service/chat"
	climbingsvc "github.com/radosny/vcoach/server/internal/service/climbing"
	"github.com/radosny/vcoach/server/internal/service/coach"
	"github.com/radosny/vcoach/server/internal/service/garmin"
	goalsvc "github.com/radosny/vcoach/server/internal/service/goals"
	plansvc "github.com/radosny/vcoach/server/internal/service/plans"
	profilesvc "github.com/radosny/vcoach/server/internal/service/profile"
	syncsvc "github.com/radosny/vcoach/server/internal/service/sync"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := migrateDB(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	db, err := sqlx.Connect("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	repo := repository.New(db)
	garminClient := garmin.NewHTTPClient(cfg.GarminAPIKey)
	syncService := syncsvc.New(repo, garminClient, log)

	// Resolve the default athlete up front so the scheduler has an ID; this
	// also creates the athlete row on first boot.
	athleteID, err := repo.GetOrCreateDefaultAthlete(ctx)
	if err != nil {
		return fmt.Errorf("resolve default athlete: %w", err)
	}
	go syncService.Run(ctx, athleteID, cfg.SyncInterval)

	llm := coach.NewOpenAICompatibleClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	profileService := profilesvc.New(repo, log)
	coachService := coach.New(coach.Dependencies{
		Repo:     repo,
		Log:      log,
		LLM:      llm,
		Goals:    goalsvc.New(repo),
		Plans:    plansvc.New(repo),
		Climbing: climbingsvc.New(repo),
		Profile:  profileService,
	})

	handler := server.NewRouter(server.Deps{
		Log:      log,
		Athletes: repo,
		Profile:  profileService,
		Goals:    goalsvc.New(repo),
		Plans:    plansvc.New(repo),
		Climbing: climbingsvc.New(repo),
		Chat:     chatsvc.New(repo),
		Sync:     syncService,
		Coach:    coachService,
	})

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", httpSrv.Addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	}
}

func migrateDB(databaseURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
