// Package review implements Leitner spaced-repetition scheduling as pure functions.
// Time is always passed in, so callers inject a clock and tests use a fake one.
package review

import (
	"sort"
	"time"
)

const (
	MinBox = 1
	MaxBox = 5
)

const day = 24 * time.Hour

// intervals[box] is how long a question stays out of the queue after being answered
// correctly into that box: box1 = 1d, box2 = 3d, box3 = 7d, box4 = 14d, box5 = 30d.
var intervals = [MaxBox + 1]time.Duration{0, 1 * day, 3 * day, 7 * day, 14 * day, 30 * day}

// Interval returns the review interval for a box (clamped to 1..5).
func Interval(box int) time.Duration {
	return intervals[min(max(box, MinBox), MaxBox)]
}

// State is the Leitner state of one question.
type State struct {
	Box            int
	TimesCorrect   int
	TimesWrong     int
	LastResult     string // "correct" or "wrong"
	LastAnsweredAt time.Time
	NextDueAt      time.Time
}

const (
	ResultCorrect = "correct"
	ResultWrong   = "wrong"
)

// Next returns the state after answering. prev is nil the first time a question is answered
// (it starts in box 1). Correct moves up one box (capped at 5) and is due again after the new
// box's interval. Wrong returns to box 1 and is due immediately.
func Next(prev *State, correct bool, now time.Time) State {
	s := State{Box: MinBox}
	if prev != nil {
		s = *prev
	}
	s.LastAnsweredAt = now
	if correct {
		s.Box = min(s.Box+1, MaxBox)
		s.TimesCorrect++
		s.LastResult = ResultCorrect
		s.NextDueAt = now.Add(Interval(s.Box))
	} else {
		s.Box = MinBox
		s.TimesWrong++
		s.LastResult = ResultWrong
		s.NextDueAt = now
	}
	return s
}

// Entry is the scheduling view of one question.
type Entry struct {
	ID        string
	NextDueAt time.Time
}

// Queue is the result of selecting what to review now.
type Queue struct {
	IDs      []string   // due question ids, most overdue first, at most the requested limit
	DueCount int        // all due questions, before the limit
	Upcoming int        // questions not yet due
	NextDue  *time.Time // earliest due time among upcoming questions, if any
}

// SelectDue picks the questions with NextDueAt <= now, most overdue first, capped at limit.
func SelectDue(entries []Entry, now time.Time, limit int) Queue {
	var due []Entry
	var q Queue
	for _, e := range entries {
		if !e.NextDueAt.After(now) {
			due = append(due, e)
			continue
		}
		q.Upcoming++
		if q.NextDue == nil || e.NextDueAt.Before(*q.NextDue) {
			t := e.NextDueAt
			q.NextDue = &t
		}
	}
	sort.SliceStable(due, func(i, j int) bool {
		if !due[i].NextDueAt.Equal(due[j].NextDueAt) {
			return due[i].NextDueAt.Before(due[j].NextDueAt)
		}
		return due[i].ID < due[j].ID
	})
	q.DueCount = len(due)
	for i, e := range due {
		if i >= limit {
			break
		}
		q.IDs = append(q.IDs, e.ID)
	}
	return q
}
