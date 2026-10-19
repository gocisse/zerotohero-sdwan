ALTER TABLE lesson_progress DROP CONSTRAINT lesson_progress_status_check;
UPDATE lesson_progress SET status = 'in_progress' WHERE status = 'visited';
ALTER TABLE lesson_progress ADD CONSTRAINT lesson_progress_status_check
    CHECK (status IN ('not_started', 'in_progress', 'completed'));
DROP TABLE IF EXISTS section_state;
DROP TABLE IF EXISTS review_answers;
DROP TABLE IF EXISTS question_state;
DROP TABLE IF EXISTS quiz_attempt_answers;
