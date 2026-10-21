package content

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"velocloud-academy/backend/internal/sim"
)

// Observation files (content/reference/observations/*.md) record what the course owner saw in a real or lab VCO.
// Besides the claims table every fact sheet has, each file holds one or more observation blocks:
//
//	## Observation 1: <title>
//	- Date: 2026-10-20
//	- Release and build: 7.0.0.2 build R7002-... (as shown in the VCO)
//	- Screen / menu path: ...
//	- What was done: ...
//	- What was seen: ...
//	- Resolves: <fact-sheet name>: <exact substring of the row>      (or)      - Resolves: assumption <N>
//	- Effect: confirms | corrects | contradicts
//
// The validator below fails a block that has no date, no release or no resolvable reference, and flags every OBSERVED
// row (and every engine rule that rests on one) that no observation block supports.

// ObservationBlock is one parsed observation.
type ObservationBlock struct {
	File     string
	Line     int
	Title    string
	Date     string
	Release  string
	Screen   string
	Did      string
	Saw      string
	Effect   string
	Resolves []ObservationRef
}

// ObservationRef is one thing an observation resolves: a fact-sheet row or an assumption number.
type ObservationRef struct {
	Raw        string
	Sheet      string // fact sheet name without .md
	Substring  string
	Assumption int
}

// SheetRow is one claims-table row of a fact sheet.
type SheetRow struct {
	Sheet  string // file name without .md; observation files are "observations/<name>"
	File   string
	Line   int
	Claim  string
	Source string
	Status string
}

var (
	obsHeadingRe = regexp.MustCompile(`(?i)^##\s+observation\b\s*[0-9]*\s*[:.\-]?\s*(.*)$`)
	obsFieldRe   = regexp.MustCompile(`^\s*[-*]\s+([A-Za-z][A-Za-z /&]*?)\s*:\s*(.*)$`)
	obsDateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	obsAssumeRe  = regexp.MustCompile(`(?i)^assumption\s+(\d+)$`)
	// ObservationEffects are the allowed values of the Effect field.
	ObservationEffects = []string{"confirms", "corrects", "contradicts"}
)

// SheetName returns the fact sheet name of a reference path ("reference/bgp.md" -> "bgp"; the settings sheets in
// reference/config/ keep their own file name, for example "config-dns").
func SheetName(path string) string {
	n := strings.TrimPrefix(path, "reference/")
	n = strings.TrimPrefix(n, "config/")
	n = strings.TrimPrefix(n, "troubleshooting/")
	return strings.TrimSuffix(n, ".md")
}

// SheetRows parses every claims-table row of every reference document.
func SheetRows(refs []ReferenceDoc) []SheetRow {
	var out []SheetRow
	for _, doc := range refs {
		inTable := false
		for i, line := range strings.Split(strings.ReplaceAll(doc.Content, "\r\n", "\n"), "\n") {
			trim := strings.TrimSpace(line)
			switch {
			case claimsHeaderRe.MatchString(line):
				inTable = true
			case inTable && separatorRe.MatchString(line) && strings.Contains(trim, "-"):
			case inTable && strings.HasPrefix(trim, "|"):
				cells := strings.Split(strings.Trim(trim, "|"), "|")
				if len(cells) != 4 {
					continue
				}
				out = append(out, SheetRow{Sheet: SheetName(doc.Path), File: doc.Path, Line: i + 1, Claim: strings.TrimSpace(cells[0]),
					Source: strings.TrimSpace(cells[1]), Status: strings.TrimSpace(cells[3])})
			default:
				inTable = false
			}
		}
	}
	return out
}

func isObservationPath(p string) bool {
	return strings.HasPrefix(p, "reference/observations/") && !strings.HasSuffix(p, "/README.md")
}

// ParseObservations extracts the observation blocks of every observation file. Problems with the block structure
// itself are returned as errors keyed by file.
func ParseObservations(refs []ReferenceDoc) ([]ObservationBlock, []ObsProblem) {
	var blocks []ObservationBlock
	var probs []ObsProblem
	for _, doc := range refs {
		if !isObservationPath(doc.Path) {
			continue
		}
		var cur *ObservationBlock
		flush := func() {
			if cur != nil {
				blocks = append(blocks, *cur)
				cur = nil
			}
		}
		found := 0
		for i, line := range strings.Split(strings.ReplaceAll(doc.Content, "\r\n", "\n"), "\n") {
			if m := obsHeadingRe.FindStringSubmatch(line); m != nil {
				flush()
				found++
				cur = &ObservationBlock{File: doc.Path, Line: i + 1, Title: strings.TrimSpace(m[1])}
				continue
			}
			if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "###") {
				flush()
				continue
			}
			if cur == nil {
				continue
			}
			m := obsFieldRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(m[1]))
			val := strings.TrimSpace(m[2])
			switch {
			case key == "date":
				cur.Date = val
			case strings.HasPrefix(key, "release"):
				cur.Release = val
			case strings.HasPrefix(key, "screen") || strings.HasPrefix(key, "menu"):
				cur.Screen = val
			case strings.HasPrefix(key, "what was done") || key == "done" || key == "what i did":
				cur.Did = val
			case strings.HasPrefix(key, "what was seen") || key == "seen" || key == "what i saw":
				cur.Saw = val
			case key == "effect":
				cur.Effect = strings.ToLower(val)
			case key == "resolves":
				cur.Resolves = append(cur.Resolves, parseRef(val))
			}
		}
		flush()
		if found == 0 {
			probs = append(probs, ObsProblem{File: doc.Path, Message: "no observation block: add a `## Observation 1: <title>` section with the fields in reference/observations/README.md"})
		}
	}
	return blocks, probs
}

// ObsProblem is a finding of the observation validators.
type ObsProblem struct {
	File    string
	Message string
}

func parseRef(val string) ObservationRef {
	val = strings.TrimSpace(strings.Trim(val, "`"))
	ref := ObservationRef{Raw: val}
	if m := obsAssumeRe.FindStringSubmatch(val); m != nil {
		ref.Assumption, _ = strconv.Atoi(m[1])
		return ref
	}
	if i := strings.Index(val, ":"); i > 0 {
		ref.Sheet = strings.TrimSpace(val[:i])
		ref.Substring = strings.Trim(strings.TrimSpace(val[i+1:]), "\"'`")
	}
	return ref
}

// matchRows returns the rows of a sheet whose claim contains the substring.
func matchRows(rows []SheetRow, sheet, sub string) []SheetRow {
	var out []SheetRow
	for _, r := range rows {
		if r.Sheet == sheet && strings.Contains(r.Claim, sub) {
			out = append(out, r)
		}
	}
	return out
}

// CheckObservations runs both observation validators on parsed reference documents. assumptions is the set of
// assumption numbers listed in docs/SIMULATOR-MODEL.md; nil skips the check that an assumption number exists (the
// server container has no docs folder), the offline `make validate-content` passes it.
func CheckObservations(refs []ReferenceDoc, assumptions map[int]bool) []ObsProblem {
	blocks, probs := ParseObservations(refs)
	rows := SheetRows(refs)
	for _, b := range blocks {
		at := fmt.Sprintf("observation at line %d", b.Line)
		add := func(f string, a ...any) {
			probs = append(probs, ObsProblem{File: b.File, Message: at + ": " + fmt.Sprintf(f, a...)})
		}
		switch {
		case b.Date == "":
			add("missing Date (YYYY-MM-DD)")
		case !obsDateRe.MatchString(b.Date):
			add("Date %q is not YYYY-MM-DD", b.Date)
		default:
			if t, err := time.Parse("2006-01-02", b.Date); err != nil {
				add("Date %q is not a real date", b.Date)
			} else if t.After(time.Now().Add(48 * time.Hour)) {
				add("Date %q is in the future", b.Date)
			}
		}
		if b.Release == "" {
			add("missing Release and build (as shown in the VCO)")
		}
		if b.Screen == "" {
			add("missing Screen / menu path")
		}
		if b.Did == "" {
			add("missing What was done")
		}
		if b.Saw == "" {
			add("missing What was seen")
		}
		if !containsString(ObservationEffects, b.Effect) {
			add("Effect %q must be one of confirms, corrects, contradicts", b.Effect)
		}
		if len(b.Resolves) == 0 {
			add("missing Resolves: name a fact-sheet row (`<sheet>: <row text>`) or `assumption <N>`")
		}
		for _, ref := range b.Resolves {
			switch {
			case ref.Assumption > 0:
				if assumptions != nil && !assumptions[ref.Assumption] {
					add("Resolves %q: assumption %d is not listed in docs/SIMULATOR-MODEL.md", ref.Raw, ref.Assumption)
				}
			case ref.Sheet == "" || ref.Substring == "":
				add("Resolves %q is neither `<sheet>: <row text>` nor `assumption <N>`", ref.Raw)
			default:
				switch n := len(matchRows(rows, ref.Sheet, ref.Substring)); {
				case n == 0:
					add("Resolves %q: no row of reference/%s.md contains that text", ref.Raw, ref.Sheet)
				case n > 1:
					add("Resolves %q: %d rows of reference/%s.md contain that text; use a longer excerpt", ref.Raw, n, ref.Sheet)
				}
			}
		}
	}

	// Every OBSERVED row needs an observation block, dated like the row, that supports it.
	supported := func(r SheetRow) bool {
		d := strings.TrimPrefix(r.Source, "user observation ")
		for _, b := range blocks {
			if b.Date != d {
				continue
			}
			if b.File == r.File {
				return true
			}
			for _, ref := range b.Resolves {
				if ref.Sheet == r.Sheet && ref.Substring != "" && strings.Contains(r.Claim, ref.Substring) {
					return true
				}
			}
		}
		return false
	}
	for _, r := range rows {
		if r.Status == "OBSERVED" && !supported(r) {
			probs = append(probs, ObsProblem{File: r.File, Message: fmt.Sprintf("line %d: an OBSERVED row needs an observation block dated %s in reference/observations/ that resolves it (in its own file, or naming it in Resolves)",
				r.Line, strings.TrimPrefix(r.Source, "user observation "))})
		}
	}
	// Engine rules that rest on an OBSERVED row must be supported too.
	for _, rule := range sim.Rules {
		if rule.Status != sim.StatusVerified || rule.Sheet == "" {
			continue
		}
		for _, r := range matchRows(rows, rule.Sheet, rule.Claim) {
			if r.Status == "OBSERVED" && !supported(r) {
				probs = append(probs, ObsProblem{File: r.File, Message: fmt.Sprintf("engine rule %q rests on the OBSERVED row at line %d, which no observation file supports", rule.ID, r.Line)})
			}
		}
	}
	sort.SliceStable(probs, func(i, j int) bool { return probs[i].File < probs[j].File })
	return probs
}

var modelItemRe = regexp.MustCompile(`^(\d+)\. (.*)$`)

// ParseAssumptions returns the numbered assumptions of docs/SIMULATOR-MODEL.md: the numbered items under a heading
// that contains "assumption" (other numbered lists in the document, such as the build plan, are not assumptions).
func ParseAssumptions(model string) map[int]string {
	out := map[int]string{}
	in := false
	for _, line := range strings.Split(model, "\n") {
		if strings.HasPrefix(line, "#") {
			in = strings.Contains(strings.ToLower(line), "assumption")
			continue
		}
		if !in {
			continue
		}
		if m := modelItemRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			if _, dup := out[n]; !dup {
				out[n] = m[2]
			}
		}
	}
	return out
}

// ModelAssumptions returns the set of assumption numbers listed in the text of docs/SIMULATOR-MODEL.md.
func ModelAssumptions(model string) map[int]bool {
	out := map[int]bool{}
	for n := range ParseAssumptions(model) {
		out[n] = true
	}
	return out
}

// ReadReferences returns the fact sheets and observation files of a content directory (for tools; errors are
// reported by Load).
func ReadReferences(fsys fs.FS) []ReferenceDoc {
	var errs ErrorList
	return readReferences(fsys, &Bundle{}, &errs)
}
