-- Progress state for the single implicit local profile (no user_id by design).
CREATE TABLE lesson_progress (
    lesson_id    TEXT PRIMARY KEY,
    status       TEXT NOT NULL DEFAULT 'not_started'
                 CHECK (status IN ('not_started', 'in_progress', 'completed')),
    completed_at TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE quiz_attempts (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    section_id TEXT NOT NULL,
    score      INTEGER NOT NULL CHECK (score >= 0),
    total      INTEGER NOT NULL CHECK (total > 0),
    answers    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX quiz_attempts_section_idx ON quiz_attempts (section_id, created_at DESC);
