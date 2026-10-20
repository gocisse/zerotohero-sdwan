package content

import "strings"

// candidateMarker tags a line in a lesson's "Check this in your own VCO or site" box: a thing the course owner can look at in a
// real VCO or site, with the documented expectation and where the documentation may differ. Such lines are counted and listed
// by make observations-report; they are not observations and settle nothing.
const candidateMarker = "[observation candidate]"

// ObservationCandidate is one marked line of a lesson.
type ObservationCandidate struct {
	Lesson string
	Screen string // what to open or do (text before "Expect:", without a leading "Open:")
	Expect string // the documented expectation
	Differ string // where the documentation may differ ("" when the line has none)
}

// ParseCandidates returns the observation candidates of one lesson body, in order.
func ParseCandidates(lessonID, body string) []ObservationCandidate {
	var out []ObservationCandidate
	for _, line := range strings.Split(body, "\n") {
		i := strings.Index(line, candidateMarker)
		if i < 0 {
			continue
		}
		rest := strings.TrimSpace(strings.Trim(strings.TrimSpace(line[i+len(candidateMarker):]), "*"))
		c := ObservationCandidate{Lesson: lessonID}
		if j := strings.Index(rest, "Docs may differ:"); j >= 0 {
			c.Differ = strings.TrimSpace(rest[j+len("Docs may differ:"):])
			rest = strings.TrimSpace(rest[:j])
		}
		if j := strings.Index(rest, "Expect:"); j >= 0 {
			c.Expect = strings.TrimSpace(rest[j+len("Expect:"):])
			rest = strings.TrimSpace(rest[:j])
		}
		c.Screen = strings.TrimSpace(strings.TrimPrefix(rest, "Open:"))
		out = append(out, c)
	}
	return out
}
