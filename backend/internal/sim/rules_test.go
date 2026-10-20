package sim

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var citeRe = regexp.MustCompile(`//\s*rule:\s*([a-z0-9-]+)`)

// Convention: every behaviour in this package carries a "// rule: <id>" comment naming a registered Rule.
func TestEveryCitationNamesARegisteredRule(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	cited := map[string]int{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range citeRe.FindAllStringSubmatch(string(data), -1) {
			cited[m[1]]++
			if _, ok := ruleByID[m[1]]; !ok {
				t.Errorf("%s cites unregistered rule %q", f, m[1])
			}
		}
	}
	for _, r := range Rules {
		if cited[r.ID] == 0 {
			t.Errorf("rule %q is registered but no code cites it (\"// rule: %s\")", r.ID, r.ID)
		}
	}
}

// Every rule constant must be registered and every registered rule must have text and a unique id.
func TestRegistryIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	nums := map[int]string{}
	for _, r := range Rules {
		if seen[r.ID] {
			t.Errorf("duplicate rule id %s", r.ID)
		}
		seen[r.ID] = true
		if r.Text == "" {
			t.Errorf("%s: empty text", r.ID)
		}
		switch r.Status {
		case StatusVerified, StatusNotModelled:
			if r.Sheet == "" || r.Claim == "" {
				t.Errorf("%s: needs sheet and claim", r.ID)
			}
		case StatusAssumption:
			if r.Assumption == 0 {
				t.Errorf("%s: needs an assumption number", r.ID)
			}
			if prev, dup := nums[r.Assumption]; dup {
				t.Errorf("%s and %s share assumption %d", r.ID, prev, r.Assumption)
			}
			nums[r.Assumption] = r.ID
			if !strings.HasPrefix(r.ID, "assume-") {
				t.Errorf("%s: assumption rule ids are assume-N", r.ID)
			}
		default:
			t.Errorf("%s: bad status %q", r.ID, r.Status)
		}
	}
}

// Verified rules must point at a real fact sheet row whose status is VERIFIED or OBSERVED; not-modelled rules at a
// VERIFY row; assumption numbers must exist in docs/SIMULATOR-MODEL.md.
func TestRulesMatchFactSheetsAndAssumptions(t *testing.T) {
	model, err := os.ReadFile("../../../docs/SIMULATOR-MODEL.md")
	if err != nil {
		t.Fatal(err)
	}
	assumptionRe := regexp.MustCompile(`(?m)^(\d+)\. `)
	have := map[string]bool{}
	for _, m := range assumptionRe.FindAllStringSubmatch(string(model), -1) {
		have[m[1]] = true
	}
	for _, r := range Rules {
		if r.Status == StatusAssumption {
			if !have[itoa(r.Assumption)] {
				t.Errorf("%s: assumption %d is not listed in docs/SIMULATOR-MODEL.md", r.ID, r.Assumption)
			}
			continue
		}
		data, err := os.ReadFile("../../../content/reference/" + r.Sheet + ".md")
		if err != nil {
			t.Errorf("%s: sheet %s: %v", r.ID, r.Sheet, err)
			continue
		}
		var status string
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "|") && strings.Contains(line, r.Claim) {
				cells := strings.Split(strings.Trim(line, "| "), "|")
				status = strings.TrimSpace(cells[len(cells)-1])
				break
			}
		}
		switch {
		case status == "":
			t.Errorf("%s: no row in %s contains %q", r.ID, r.Sheet, r.Claim)
		case r.Status == StatusVerified && status != "VERIFIED" && status != "OBSERVED":
			t.Errorf("%s: row in %s has status %s, want VERIFIED or OBSERVED", r.ID, r.Sheet, status)
		case r.Status == StatusNotModelled && status != "VERIFY":
			t.Errorf("%s: row in %s has status %s, want VERIFY", r.ID, r.Sheet, status)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
