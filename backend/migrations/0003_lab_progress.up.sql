-- Lab progress for the single implicit local profile (no user_id by design). Lab ids come from /content/labs: rows may
-- reference ids that no longer exist and are then ignored by the API (never deleted). Nothing here ever gates anything.
CREATE TABLE lab_progress (
    lab_id          TEXT PRIMARY KEY,
    status          TEXT NOT NULL CHECK (status IN ('started', 'completed', 'skipped')),
    attempts        INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    hints_used      INTEGER NOT NULL DEFAULT 0 CHECK (hints_used >= 0),
    last_attempt_at TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
