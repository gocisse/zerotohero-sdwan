// Package store is the persistence layer for progress state.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the Postgres pool. Progress queries are added in later steps.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Ping reports whether the database is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
