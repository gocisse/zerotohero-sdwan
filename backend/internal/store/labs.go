package store

import (
	"context"
	"fmt"
	"time"

	"velocloud-academy/backend/internal/domain"
)

// LabProgressAll returns every stored lab state keyed by lab id.
func (s *Store) LabProgressAll(ctx context.Context) (map[string]domain.LabProgress, error) {
	rows, err := s.pool.Query(ctx, `SELECT lab_id, status, attempts, hints_used, last_attempt_at, completed_at FROM lab_progress`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.LabProgress{}
	for rows.Next() {
		var lp domain.LabProgress
		if err := rows.Scan(&lp.LabID, &lp.Status, &lp.Attempts, &lp.HintsUsed, &lp.LastAttemptAt, &lp.CompletedAt); err != nil {
			return nil, err
		}
		out[lp.LabID] = lp
	}
	return out, rows.Err()
}

const labUpsert = `
INSERT INTO lab_progress (lab_id, status, attempts, hints_used, last_attempt_at, completed_at, updated_at)
VALUES ($1, $2::text, $3, $4, $5, CASE WHEN $2::text = 'completed' THEN $6::timestamptz END, $6)
ON CONFLICT (lab_id) DO UPDATE SET
  status = CASE WHEN lab_progress.status = 'completed' THEN 'completed' ELSE EXCLUDED.status END,
  attempts = lab_progress.attempts + EXCLUDED.attempts,
  hints_used = GREATEST(lab_progress.hints_used, EXCLUDED.hints_used),
  last_attempt_at = COALESCE(EXCLUDED.last_attempt_at, lab_progress.last_attempt_at),
  completed_at = COALESCE(lab_progress.completed_at, EXCLUDED.completed_at),
  updated_at = EXCLUDED.updated_at
RETURNING lab_id, status, attempts, hints_used, last_attempt_at, completed_at`

func (s *Store) labWrite(ctx context.Context, labID, status string, attempts, hints int, attemptAt *time.Time, now time.Time) (domain.LabProgress, error) {
	var lp domain.LabProgress
	err := s.pool.QueryRow(ctx, labUpsert, labID, status, attempts, hints, attemptAt, now).
		Scan(&lp.LabID, &lp.Status, &lp.Attempts, &lp.HintsUsed, &lp.LastAttemptAt, &lp.CompletedAt)
	if err != nil {
		return lp, fmt.Errorf("write lab progress: %w", err)
	}
	return lp, nil
}

// SetLabStatus sets started, completed or skipped. 'completed' is sticky (never downgraded). Idempotent.
func (s *Store) SetLabStatus(ctx context.Context, labID, status string, now time.Time) (domain.LabProgress, error) {
	return s.labWrite(ctx, labID, status, 0, 0, nil, now)
}

// RecordLabAttempt counts one checked task and starts the lab if it was not started.
func (s *Store) RecordLabAttempt(ctx context.Context, labID string, now time.Time) (domain.LabProgress, error) {
	return s.labWrite(ctx, labID, domain.LabStarted, 1, 0, &now, now)
}

// UseLabHint records that hints up to n were revealed (never decreases) and starts the lab if needed.
func (s *Store) UseLabHint(ctx context.Context, labID string, n int, now time.Time) (domain.LabProgress, error) {
	return s.labWrite(ctx, labID, domain.LabStarted, 0, n, nil, now)
}
