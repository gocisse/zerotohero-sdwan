package mastery

import (
	"math"
	"testing"
	"time"

	"velocloud-academy/backend/internal/domain"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// seq builds events for one topic, oldest first, one hour apart.
func seq(topic string, partials ...float64) []domain.AnswerEvent {
	out := make([]domain.AnswerEvent, len(partials))
	for i, p := range partials {
		out[i] = domain.AnswerEvent{QuestionID: "q", Topic: topic, Partial: p, At: t0.Add(time.Duration(i) * time.Hour)}
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		events   []domain.AnswerEvent
		wantAcc  float64
		wantWeak bool
		answers  int
	}{
		{"all correct", seq("x", 1, 1, 1, 1), 100, false, 4},
		{"all wrong with 3 answers is weak", seq("x", 0, 0, 0), 0, true, 3},
		{"weak needs at least 3 answers", seq("x", 0, 0), 0, false, 2},
		{"exactly 70% is not weak (boundary)", seq("x", 0.7, 0.7, 0.7), 70, false, 3},
		{"just under 70% is weak", seq("x", 0.6, 0.6, 0.6), 60, true, 3},
		// weights oldest..newest = 1,2,3: wrong,right,right -> (0+2+3)/6
		{"recent answers weigh more: improving", seq("x", 0, 1, 1), 5.0 / 6 * 100, false, 3},
		// right,wrong,wrong -> (1+0+0)/6
		{"recent answers weigh more: declining", seq("x", 1, 0, 0), 1.0 / 6 * 100, true, 3},
		{"partial credit counts", seq("x", 0.5, 0.5, 0.5), 50, true, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compute(tt.events)
			if len(got) != 1 {
				t.Fatalf("topics = %+v", got)
			}
			if !approx(got[0].AccuracyPct, tt.wantAcc) || got[0].Weak != tt.wantWeak || got[0].Answers != tt.answers {
				t.Errorf("got %+v, want acc=%v weak=%v answers=%d", got[0], tt.wantAcc, tt.wantWeak, tt.answers)
			}
		})
	}
}

func TestComputeOnlyLastTenCount(t *testing.T) {
	// 5 old wrong answers followed by 10 correct ones: the old failures fall out of the window.
	events := seq("x", 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1)
	got := Compute(events)[0]
	if !approx(got.AccuracyPct, 100) || got.Answers != 15 || got.Weak {
		t.Errorf("got %+v", got)
	}
}

func TestComputeIgnoresInputOrder(t *testing.T) {
	events := seq("x", 0, 1, 1)
	events[0], events[2] = events[2], events[0] // shuffle; timestamps decide recency
	if got := Compute(events)[0]; !approx(got.AccuracyPct, 5.0/6*100) {
		t.Errorf("accuracy = %v", got.AccuracyPct)
	}
}

func TestComputeSortsWeakestFirstAndGroupsTopics(t *testing.T) {
	var events []domain.AnswerEvent
	events = append(events, seq("b-strong", 1, 1, 1)...)
	events = append(events, seq("a-weak", 0, 0, 1)...)
	events = append(events, seq("c-mid", 0, 1)...)
	got := Compute(events)
	if len(got) != 3 || got[0].Topic != "a-weak" || got[1].Topic != "c-mid" || got[2].Topic != "b-strong" {
		t.Fatalf("order = %+v", got)
	}
	if !got[0].Weak || got[1].Weak || got[2].Weak {
		t.Errorf("weak flags = %v %v %v", got[0].Weak, got[1].Weak, got[2].Weak)
	}
	if Compute(nil) == nil || len(Compute(nil)) != 0 {
		t.Error("Compute(nil) should return an empty, non-nil slice")
	}
}
