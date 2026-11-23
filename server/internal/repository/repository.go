package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// queryer is the subset of *sqlx.DB / *sqlx.Tx methods the repository uses.
// It lets a Repository transparently wrap a transaction.
type queryer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Repository provides data access for the vcoach domain tables.
type Repository struct {
	db   queryer
	pool *sqlx.DB // nil when this Repository wraps a transaction
}

// New builds a Repository over the given sqlx database handle.
func New(db *sqlx.DB) *Repository {
	return &Repository{db: db, pool: db}
}

// DB exposes the underlying pool for tests. Panics if this Repository wraps
// a transaction — tests never run inside one.
func (r *Repository) DB() *sqlx.DB {
	if r.pool == nil {
		panic("repository wraps a transaction; no pool available")
	}
	return r.pool
}

// WithTx runs fn inside a database transaction. The fn receives a Repository
// bound to the transaction; an error rolls back, nil commits.
func (r *Repository) WithTx(ctx context.Context, fn func(txRepo *Repository) error) error {
	if r.pool == nil {
		// Already inside a transaction: join it.
		return fn(r)
	}
	tx, err := r.pool.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(&Repository{db: tx, pool: nil}); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("rollback after error %q: %w", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
