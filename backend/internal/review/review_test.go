package review

import (
	"testing"
	"time"
)

// fakeClock is an injectable clock: tests advance it explicitly.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func TestInterval(t *testing.T) {
	want := map[int]time.Duration{1: 24 * time.Hour, 2: 72 * time.Hour, 3: 7 * 24 * time.Hour, 4: 14 * 24 * time.Hour, 5: 30 * 24 * time.Hour, 0: 24 * time.Hour, 9: 30 * 24 * time.Hour}
	for box, d := range want {
		if got := Interval(box); got != d {
			t.Errorf("Interval(%d) = %v, want %v", box, got, d)
		}
	}
}

func TestNext(t *testing.T) {
	clock := newClock()
	now := clock.Now()
	at := func(box int) *State { return &State{Box: box, TimesCorrect: 2, TimesWrong: 1} }

	tests := []struct {
		name        string
		prev        *State
		correct     bool
		wantBox     int
		wantDueIn   time.Duration
		wantCorrect int
		wantWrong   int
		wantResult  string
	}{
		{"first answer correct moves box1 -> box2", nil, true, 2, 3 * 24 * time.Hour, 1, 0, "correct"},
		{"first answer wrong stays box1, due immediately", nil, false, 1, 0, 0, 1, "wrong"},
		{"promotion 2 -> 3", at(2), true, 3, 7 * 24 * time.Hour, 3, 1, "correct"},
		{"promotion 3 -> 4", at(3), true, 4, 14 * 24 * time.Hour, 3, 1, "correct"},
		{"promotion 4 -> 5", at(4), true, 5, 30 * 24 * time.Hour, 3, 1, "correct"},
		{"cap at box 5", at(5), true, 5, 30 * 24 * time.Hour, 3, 1, "correct"},
		{"demotion from box 4 to 1, due immediately", at(4), false, 1, 0, 2, 2, "wrong"},
		{"demotion from box 5 to 1", at(5), false, 1, 0, 2, 2, "wrong"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before State
			if tt.prev != nil {
				before = *tt.prev
			}
			got := Next(tt.prev, tt.correct, clock.Now())
			if got.Box != tt.wantBox || got.TimesCorrect != tt.wantCorrect || got.TimesWrong != tt.wantWrong || got.LastResult != tt.wantResult {
				t.Errorf("got %+v", got)
			}
			if want := now.Add(tt.wantDueIn); !got.NextDueAt.Equal(want) {
				t.Errorf("NextDueAt = %v, want %v", got.NextDueAt, want)
			}
			if !got.LastAnsweredAt.Equal(now) {
				t.Errorf("LastAnsweredAt = %v", got.LastAnsweredAt)
			}
			if tt.prev != nil && *tt.prev != before {
				t.Error("Next must not mutate prev")
			}
		})
	}
}

func TestNextSequenceWithClock(t *testing.T) {
	clock := newClock()
	s := Next(nil, true, clock.Now()) // box 2, due +3d
	clock.Advance(3 * 24 * time.Hour)
	s = Next(&s, true, clock.Now()) // box 3, due +7d from the new now
	if s.Box != 3 || !s.NextDueAt.Equal(clock.Now().Add(7*24*time.Hour)) {
		t.Fatalf("got %+v", s)
	}
	clock.Advance(time.Hour)
	s = Next(&s, false, clock.Now())
	if s.Box != 1 || !s.NextDueAt.Equal(clock.Now()) {
		t.Fatalf("wrong answer must reset to box 1, due now: %+v", s)
	}
}

func TestSelectDue(t *testing.T) {
	now := newClock().Now()
	h := func(d time.Duration) time.Time { return now.Add(d) }
	entries := []Entry{
		{"c", h(-1 * time.Hour)},
		{"a", h(-5 * time.Hour)},
		{"b", h(-5 * time.Hour)},
		{"now", h(0)},
		{"later", h(48 * time.Hour)},
		{"soon", h(2 * time.Hour)},
	}

	t.Run("most overdue first, ties by id, upcoming and next due", func(t *testing.T) {
		q := SelectDue(entries, now, 15)
		want := []string{"a", "b", "c", "now"}
		if len(q.IDs) != len(want) {
			t.Fatalf("ids = %v", q.IDs)
		}
		for i := range want {
			if q.IDs[i] != want[i] {
				t.Fatalf("ids = %v, want %v", q.IDs, want)
			}
		}
		if q.DueCount != 4 || q.Upcoming != 2 || q.NextDue == nil || !q.NextDue.Equal(h(2*time.Hour)) {
			t.Errorf("queue = %+v", q)
		}
	})
	t.Run("limit caps ids but not the due count", func(t *testing.T) {
		q := SelectDue(entries, now, 2)
		if len(q.IDs) != 2 || q.DueCount != 4 {
			t.Errorf("queue = %+v", q)
		}
	})
	t.Run("nothing due", func(t *testing.T) {
		q := SelectDue([]Entry{{"x", h(time.Hour)}}, now, 15)
		if q.DueCount != 0 || len(q.IDs) != 0 || q.NextDue == nil {
			t.Errorf("queue = %+v", q)
		}
	})
	t.Run("empty", func(t *testing.T) {
		if q := SelectDue(nil, now, 15); q.DueCount != 0 || q.Upcoming != 0 || q.NextDue != nil {
			t.Errorf("queue = %+v", q)
		}
	})
}
