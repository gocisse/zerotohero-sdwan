package domain

import (
	"time"

	"velocloud-academy/backend/internal/review"
)

// Lesson progress statuses. "not_started" is implicit: a lesson without a row.
const (
	LessonNotStarted = "not_started"
	LessonVisited    = "visited"
	LessonCompleted  = "completed"
)

// Attempt is one recorded section quiz attempt.
type Attempt struct {
	ID        int64
	SectionID string
	Score     int
	Total     int
	CreatedAt time.Time
}

// AnswerRecord is one graded answer to persist with an attempt or review answer.
type AnswerRecord struct {
	QuestionID string
	Topic      string
	Difficulty int
	Correct    bool
	Partial    float64
	Given      any // string, []string or nil
}

// AttemptInput is everything needed to record a quiz attempt atomically.
type AttemptInput struct {
	SectionID string
	Score     int
	Total     int
	Answers   []AnswerRecord
	Now       time.Time
}

// LessonProgress is a stored lesson status.
type LessonProgress struct {
	LessonID    string
	Status      string
	CompletedAt *time.Time
	UpdatedAt   time.Time
}

// SectionState is the per-section state beyond lesson progress.
type SectionState struct {
	SectionID     string
	QuizSkippedAt *time.Time
	LastVisitedAt *time.Time
}

// QuestionState is the Leitner state of a question, with its topic.
type QuestionState struct {
	QuestionID string
	Topic      string
	review.State
}

// AnswerEvent is one historical answer (quiz or review), used for topic mastery.
type AnswerEvent struct {
	QuestionID string
	Topic      string
	Partial    float64
	At         time.Time
}

// Lab statuses. "not_started" is implicit: a lab without a row.
const (
	LabNotStarted = "not_started"
	LabStarted    = "started"
	LabCompleted  = "completed"
	LabSkipped    = "skipped"
)

// LabProgress is the stored state of one lab.
type LabProgress struct {
	LabID         string
	Status        string
	Attempts      int
	HintsUsed     int
	LastAttemptAt *time.Time
	CompletedAt   *time.Time
}

// Activity event kinds, derived from stored timestamps (see store.ActivityEvents for what each one can and cannot show).
const (
	ActLessonVisited   = "lesson_visited"
	ActLessonCompleted = "lesson_completed"
	ActQuizAttempt     = "quiz_attempt"
	ActReviewAnswer    = "review_answer"
	ActLabAttempt      = "lab_attempt"
	ActLabCompleted    = "lab_completed"
)

// ActivityEvent is one timestamped thing the learner did. Ref is the lesson, section, question or lab id.
type ActivityEvent struct {
	Kind string
	Ref  string
	At   time.Time
}
