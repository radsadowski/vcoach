// Package database provides testcontainers-backed PostgreSQL helpers for
// service and server tests. Tests run against the real repository so business
// logic is verified end-to-end, per the kubecast testing conventions.
package database

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/radosny/vcoach/server/db/migrations"
)

// SetupDB starts a throwaway PostgreSQL 17 container, applies all migrations
// and returns a connected pool. The container is torn down when the test
// finishes. Each test gets its own database, so tests run in parallel.
func SetupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("vcoach"),
		postgres.WithUsername("vcoach"),
		postgres.WithPassword("vcoach"),
	)
	require.NoError(t, err)

	connStr, err := waitForConnectionString(ctx, container)
	require.NoError(t, err)

	db, err := connectWithRetry(ctx, connStr)
	require.NoError(t, err)

	source, err := iofs.New(migrations.FS, ".")
	require.NoError(t, err)
	migrator, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	require.NoError(t, err)
	require.NoError(t, migrator.Up())
	srcErr, dbErr := migrator.Close()
	require.NoError(t, srcErr)
	require.NoError(t, dbErr)

	t.Cleanup(func() {
		_ = db.Close()
		_ = container.Terminate(context.Background())
	})
	return db
}

// waitForConnectionString retries ConnectionString briefly: right after
// startup, Docker may not have published the mapped port yet, which shows up
// as the flaky `port "5432/tcp" not found` error.
func waitForConnectionString(ctx context.Context, container *postgres.PostgresContainer) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err == nil {
			return connStr, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return "", lastErr
}

// connectWithRetry retries the initial connection: the postgres entrypoint
// restarts the server once during first-time initialization, and a connection
// opened in that window is reset by the shutting-down temp server.
func connectWithRetry(ctx context.Context, connStr string) (*sqlx.DB, error) {
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		db, err := sqlx.Connect("pgx", connStr)
		if err == nil {
			return db, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return nil, lastErr
}
