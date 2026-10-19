// Package mastery computes per-topic accuracy from answer history as pure functions.
package mastery

import (
	"sort"

	"velocloud-academy/backend/internal/domain"
)

const (
	// Window is how many of the most recent answers per topic count toward accuracy.
	Window = 10
	// WeakBelowPct and WeakMinAnswers define a weak topic: accuracy under the threshold
	// with at least that many answers on record.
	WeakBelowPct   = 70.0
	WeakMinAnswers = 3
)

// Topic is the mastery summary of one topic.
type Topic struct {
	Topic       string
	Answers     int     // all answers ever recorded on the topic
	AccuracyPct float64 // recency-weighted over the last Window answers
	Weak        bool
}

// Compute groups answers by topic. Within a topic the newest answer carries the largest weight:
// with n answers in the window the newest has weight n, the next n-1, ... the oldest 1.
// Accuracy uses each answer's partial score (1 or 0, except multi_choice partial credit).
// Result is sorted weakest first (lowest accuracy, then topic name).
func Compute(events []domain.AnswerEvent) []Topic {
	byTopic := map[string][]domain.AnswerEvent{}
	for _, e := range events {
		byTopic[e.Topic] = append(byTopic[e.Topic], e)
	}

	out := make([]Topic, 0, len(byTopic))
	for topic, evs := range byTopic {
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].At.After(evs[j].At) }) // newest first
		recent := evs
		if len(recent) > Window {
			recent = recent[:Window]
		}
		var sum, weights float64
		for i, e := range recent {
			w := float64(len(recent) - i)
			sum += w * e.Partial
			weights += w
		}
		acc := sum / weights * 100
		out = append(out, Topic{
			Topic:       topic,
			Answers:     len(evs),
			AccuracyPct: acc,
			Weak:        acc < WeakBelowPct-1e-9 && len(evs) >= WeakMinAnswers, // epsilon: 70.0000…01 must not flip the boundary
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AccuracyPct != out[j].AccuracyPct {
			return out[i].AccuracyPct < out[j].AccuracyPct
		}
		return out[i].Topic < out[j].Topic
	})
	return out
}
