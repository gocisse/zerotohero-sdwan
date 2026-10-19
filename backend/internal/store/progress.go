package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"velocloud-academy/backend/internal/domain"
	"velocloud-academy/backend/internal/review"
)

// jsonb marshals v for a jsonb parameter; nil becomes SQL NULL.
func jsonb(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

// RecordAttempt persists a quiz attempt, its per-question answers and the Leitner state of every
// question in ONE transaction. Rows may reference ids that later disappear from /content; they are
// never deleted here.
func (s *Store) RecordAttempt(ctx context.Context, in domain.AttemptInput) (domain.Attempt, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Attempt{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	given := make(map[string]any, len(in.Answers))
	for _, a := range in.Answers {
		given[a.QuestionID] = a.Given
	}
	att := domain.Attempt{SectionID: in.SectionID, Score: in.Score, Total: in.Total, CreatedAt: in.Now}
	if err := tx.QueryRow(ctx,
		`INSERT INTO quiz_attempts (section_id, score, total, answers, created_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		in.SectionID, in.Score, in.Total, jsonb(given), in.Now).Scan(&att.ID); err != nil {
		return domain.Attempt{}, fmt.Errorf("insert attempt: %w", err)
	}

	batch := &pgx.Batch{}
	for _, a := range in.Answers {
		batch.Queue(
			`INSERT INTO quiz_attempt_answers (attempt_id, question_id, topic, difficulty, correct, partial_score, given)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			att.ID, a.QuestionID, a.Topic, a.Difficulty, a.Correct, a.Partial, jsonb(a.Given))
	}
	if err := execBatch(ctx, tx, batch); err != nil {
		return domain.Attempt{}, fmt.Errorf("insert attempt answers: %w", err)
	}
	if err := applyLeitner(ctx, tx, in.Answers, in.Now); err != nil {
		return domain.Attempt{}, err
	}
	return att, tx.Commit(ctx)
}

// RecordReview logs review answers and updates Leitner state. It never creates quiz attempts.
func (s *Store) RecordReview(ctx context.Context, answers []domain.AnswerRecord, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(
			`INSERT INTO review_answers (question_id, topic, correct, partial_score, given, answered_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			a.QuestionID, a.Topic, a.Correct, a.Partial, jsonb(a.Given), now)
	}
	if err := execBatch(ctx, tx, batch); err != nil {
		return fmt.Errorf("insert review answers: %w", err)
	}
	if err := applyLeitner(ctx, tx, answers, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func execBatch(ctx context.Context, tx pgx.Tx, b *pgx.Batch) error {
	if b.Len() == 0 {
		return nil
	}
	br := tx.SendBatch(ctx, b)
	defer br.Close()
	for i := 0; i < b.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// applyLeitner moves each answered question between boxes (locking existing rows first).
func applyLeitner(ctx context.Context, tx pgx.Tx, answers []domain.AnswerRecord, now time.Time) error {
	if len(answers) == 0 {
		return nil
	}
	ids := make([]string, len(answers))
	for i, a := range answers {
		ids[i] = a.QuestionID
	}
	rows, err := tx.Query(ctx,
		`SELECT question_id, box, times_correct, times_wrong, last_result, last_answered_at, next_due_at
		 FROM question_state WHERE question_id = ANY($1) FOR UPDATE`, ids)
	if err != nil {
		return fmt.Errorf("load question state: %w", err)
	}
	prev := map[string]review.State{}
	for rows.Next() {
		var id string
		var st review.State
		if err := rows.Scan(&id, &st.Box, &st.TimesCorrect, &st.TimesWrong, &st.LastResult, &st.LastAnsweredAt, &st.NextDueAt); err != nil {
			rows.Close()
			return err
		}
		prev[id] = st
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, a := range answers {
		var p *review.State
		if st, ok := prev[a.QuestionID]; ok {
			p = &st
		}
		next := review.Next(p, a.Correct, now)
		batch.Queue(
			`INSERT INTO question_state (question_id, topic, box, times_correct, times_wrong, last_result, last_answered_at, next_due_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (question_id) DO UPDATE SET topic = EXCLUDED.topic, box = EXCLUDED.box,
			   times_correct = EXCLUDED.times_correct, times_wrong = EXCLUDED.times_wrong,
			   last_result = EXCLUDED.last_result, last_answered_at = EXCLUDED.last_answered_at,
			   next_due_at = EXCLUDED.next_due_at`,
			a.QuestionID, a.Topic, next.Box, next.TimesCorrect, next.TimesWrong, next.LastResult, next.LastAnsweredAt, next.NextDueAt)
	}
	if err := execBatch(ctx, tx, batch); err != nil {
		return fmt.Errorf("save question state: %w", err)
	}
	return nil
}

// ListAttempts returns a section's attempts, newest first.
func (s *Store) ListAttempts(ctx context.Context, sectionID string) ([]domain.Attempt, error) {
	return s.queryAttempts(ctx, `WHERE section_id = $1 ORDER BY created_at DESC, id DESC`, sectionID)
}

// AllAttempts returns every attempt, oldest first.
func (s *Store) AllAttempts(ctx context.Context) ([]domain.Attempt, error) {
	return s.queryAttempts(ctx, `ORDER BY created_at ASC, id ASC`)
}

func (s *Store) queryAttempts(ctx context.Context, tail string, args ...any) ([]domain.Attempt, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, section_id, score, total, created_at FROM quiz_attempts `+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Attempt{}
	for rows.Next() {
		var a domain.Attempt
		if err := rows.Scan(&a.ID, &a.SectionID, &a.Score, &a.Total, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SkipQuiz records that the section quiz was skipped. Idempotent: the first skip time is kept.
func (s *Store) SkipQuiz(ctx context.Context, sectionID string, now time.Time) (time.Time, error) {
	var at time.Time
	err := s.pool.QueryRow(ctx,
		`INSERT INTO section_state (section_id, quiz_skipped_at) VALUES ($1, $2)
		 ON CONFLICT (section_id) DO UPDATE
		   SET quiz_skipped_at = COALESCE(section_state.quiz_skipped_at, EXCLUDED.quiz_skipped_at)
		 RETURNING quiz_skipped_at`, sectionID, now).Scan(&at)
	return at, err
}

// SetLessonProgress marks a lesson visited or completed. 'completed' is sticky: it is never
// downgraded to 'visited'. Idempotent. Also stamps the section's last_visited_at.
func (s *Store) SetLessonProgress(ctx context.Context, lessonID, sectionID, status string, now time.Time) (domain.LessonProgress, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.LessonProgress{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	lp := domain.LessonProgress{LessonID: lessonID}
	if err := tx.QueryRow(ctx,
		`INSERT INTO lesson_progress (lesson_id, status, completed_at, updated_at)
		 VALUES ($1, $2::text, CASE WHEN $2::text = 'completed' THEN $3::timestamptz END, $3)
		 ON CONFLICT (lesson_id) DO UPDATE SET
		   status = CASE WHEN lesson_progress.status = 'completed' OR EXCLUDED.status = 'completed'
		                 THEN 'completed' ELSE 'visited' END,
		   completed_at = COALESCE(lesson_progress.completed_at, EXCLUDED.completed_at),
		   updated_at = EXCLUDED.updated_at
		 RETURNING status, completed_at`, lessonID, status, now).Scan(&lp.Status, &lp.CompletedAt); err != nil {
		return lp, fmt.Errorf("upsert lesson progress: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO section_state (section_id, last_visited_at) VALUES ($1, $2)
		 ON CONFLICT (section_id) DO UPDATE SET last_visited_at = EXCLUDED.last_visited_at`, sectionID, now); err != nil {
		return lp, fmt.Errorf("touch section: %w", err)
	}
	return lp, tx.Commit(ctx)
}

// LessonProgressAll returns every stored lesson status keyed by lesson id.
func (s *Store) LessonProgressAll(ctx context.Context) (map[string]domain.LessonProgress, error) {
	rows, err := s.pool.Query(ctx, `SELECT lesson_id, status, completed_at FROM lesson_progress`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.LessonProgress{}
	for rows.Next() {
		var lp domain.LessonProgress
		if err := rows.Scan(&lp.LessonID, &lp.Status, &lp.CompletedAt); err != nil {
			return nil, err
		}
		out[lp.LessonID] = lp
	}
	return out, rows.Err()
}

// SectionStates returns every stored section state keyed by section id.
func (s *Store) SectionStates(ctx context.Context) (map[string]domain.SectionState, error) {
	rows, err := s.pool.Query(ctx, `SELECT section_id, quiz_skipped_at, last_visited_at FROM section_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.SectionState{}
	for rows.Next() {
		var st domain.SectionState
		if err := rows.Scan(&st.SectionID, &st.QuizSkippedAt, &st.LastVisitedAt); err != nil {
			return nil, err
		}
		out[st.SectionID] = st
	}
	return out, rows.Err()
}

// QuestionStates returns every Leitner state row.
func (s *Store) QuestionStates(ctx context.Context) ([]domain.QuestionState, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT question_id, topic, box, times_correct, times_wrong, last_result, last_answered_at, next_due_at FROM question_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.QuestionState{}
	for rows.Next() {
		var q domain.QuestionState
		if err := rows.Scan(&q.QuestionID, &q.Topic, &q.Box, &q.TimesCorrect, &q.TimesWrong, &q.LastResult, &q.LastAnsweredAt, &q.NextDueAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// AnswerHistory returns every answer ever given (quiz and review), for topic mastery.
func (s *Store) AnswerHistory(ctx context.Context) ([]domain.AnswerEvent, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT a.question_id, a.topic, a.partial_score::float8, t.created_at
		   FROM quiz_attempt_answers a JOIN quiz_attempts t ON t.id = a.attempt_id
		 UNION ALL
		 SELECT question_id, topic, partial_score::float8, answered_at FROM review_answers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AnswerEvent{}
	for rows.Next() {
		var e domain.AnswerEvent
		if err := rows.Scan(&e.QuestionID, &e.Topic, &e.Partial, &e.At); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
