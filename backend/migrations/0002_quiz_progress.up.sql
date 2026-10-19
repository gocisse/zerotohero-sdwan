-- Per-question results of a quiz attempt. Content ids are the source of truth:
-- rows may reference ids that no longer exist in /content and are then ignored by the API (never deleted).
CREATE TABLE quiz_attempt_answers (
    attempt_id    BIGINT NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id   TEXT NOT NULL,
    topic         TEXT NOT NULL,
    difficulty    INTEGER NOT NULL,
    correct       BOOLEAN NOT NULL,
    partial_score NUMERIC(4,3) NOT NULL CHECK (partial_score >= 0 AND partial_score <= 1),
    given         JSONB,
    PRIMARY KEY (attempt_id, question_id)
);

-- Leitner state, one row per question id.
CREATE TABLE question_state (
    question_id      TEXT PRIMARY KEY,
    topic            TEXT NOT NULL,
    box              INTEGER NOT NULL DEFAULT 1 CHECK (box BETWEEN 1 AND 5),
    times_correct    INTEGER NOT NULL DEFAULT 0,
    times_wrong      INTEGER NOT NULL DEFAULT 0,
    last_result      TEXT NOT NULL CHECK (last_result IN ('correct', 'wrong')),
    last_answered_at TIMESTAMPTZ NOT NULL,
    next_due_at      TIMESTAMPTZ NOT NULL
);
CREATE INDEX question_state_due_idx ON question_state (next_due_at);

-- Answers given in review sessions. Kept separate from quiz_attempts on purpose:
-- review never creates quiz attempts, but it still counts toward topic mastery.
CREATE TABLE review_answers (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id   TEXT NOT NULL,
    topic         TEXT NOT NULL,
    correct       BOOLEAN NOT NULL,
    partial_score NUMERIC(4,3) NOT NULL CHECK (partial_score >= 0 AND partial_score <= 1),
    given         JSONB,
    answered_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE section_state (
    section_id      TEXT PRIMARY KEY,
    quiz_skipped_at TIMESTAMPTZ,
    last_visited_at TIMESTAMPTZ
);

-- Lesson statuses become 'visited' | 'completed' (plus the implicit 'not_started').
UPDATE lesson_progress SET status = 'visited' WHERE status = 'in_progress';
ALTER TABLE lesson_progress DROP CONSTRAINT lesson_progress_status_check;
ALTER TABLE lesson_progress ADD CONSTRAINT lesson_progress_status_check
    CHECK (status IN ('not_started', 'visited', 'completed'));
