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
