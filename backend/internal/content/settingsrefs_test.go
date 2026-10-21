package content

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// termTagRe matches a Term tag with text, whose text is what the reader sees as the field name.
var termTagRe = regexp.MustCompile(`<Term\s+id="[^"]*">([^<]*)</Term>`)

// settingsFields returns the first-column cells of every table under a "## Settings reference" heading of a lesson body
// (including its "###" sub-sections), without the header and separator rows.
func settingsFields(body string) []string {
	var out []string
	in := false
	header := false
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "## "):
			in = strings.TrimSpace(strings.TrimPrefix(trim, "## ")) == "Settings reference"
			header = false
		case in && strings.HasPrefix(trim, "|"):
			cells := strings.Split(strings.Trim(trim, "|"), "|")
			first := strings.TrimSpace(cells[0])
			switch {
			case strings.HasPrefix(first, "---"):
			case !header:
				header = true // the table's header row
			default:
				first = strings.Trim(termTagRe.ReplaceAllString(first, "$1"), "`* ")
				if first != "" {
					out = append(out, first)
				}
			}
		case in && trim == "":
			header = false
		}
	}
	return out
}

func TestSettingsFieldsParse(t *testing.T) {
	body := "## Settings reference\n\n| Field | What |\n|---|---|\n| Name | x |\n| `Local IP` | y |\n| <Term id=\"x\">Linked field</Term> | y |\n\n### More\n\n| Field | What |\n|---|---|\n| **Max-hop** | z |\n\n## Next\n\n| A | B |\n|---|---|\n| Not me | q |\n"
	got := settingsFields(body)
	want := []string{"Name", "Local IP", "Linked field", "Max-hop"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

// Every field named in a "Settings reference" table of a Section 8 lesson must appear in a VERIFIED or OBSERVED row of a settings
// fact sheet (content/reference/config/), so a lesson cannot name a field the documentation was not read for. Unverified fields
// belong in the warning Callout instead.
func TestSection8SettingsFieldsExistInVerifiedSheetRows(t *testing.T) {
	fsys := os.DirFS("../../../content")
	c, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, r := range SheetRows(ReadReferences(fsys)) {
		if strings.HasPrefix(r.Sheet, "config-") && (r.Status == "VERIFIED" || r.Status == "OBSERVED") {
			rows = append(rows, r.Claim)
		}
	}
	if len(rows) < 100 {
		t.Fatalf("only %d verified settings rows found", len(rows))
	}
	// The lessons whose Settings reference tables are checked, with the number of lessons each section must have.
	want := map[string]int{"network-services-config": 8, "profile-policy-config": 7}
	checked, lessons := 0, 0
	for _, s := range c.Sections {
		if want[s.Meta.ID] == 0 {
			continue
		}
		for _, l := range s.Lessons {
			fields := settingsFields(l.Body)
			if len(fields) == 0 {
				t.Errorf("%s has no \"Settings reference\" table", l.Meta.ID)
				continue
			}
			lessons++
			for _, f := range fields {
				checked++
				found := false
				for _, r := range rows {
					if strings.Contains(r, f) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: field %q is in no VERIFIED or OBSERVED row of reference/config/", l.Meta.ID, f)
				}
			}
		}
	}
	if lessons != 15 || checked < 120 {
		t.Errorf("checked %d fields in %d lessons, want 15 lessons", checked, lessons)
	}
}

// ts-prefixed fact sheets are the troubleshooting sheets (content/reference/troubleshooting/).
func verifiedTroubleshootingRows(t *testing.T, c *Course) []string {
	t.Helper()
	fsys := os.DirFS("../../../content")
	var rows []string
	for _, r := range SheetRows(ReadReferences(fsys)) {
		if strings.HasPrefix(r.Sheet, "ts-") && (r.Status == "VERIFIED" || r.Status == "OBSERVED") {
			rows = append(rows, strings.ToLower(r.Claim))
		}
	}
	if len(rows) < 80 {
		t.Fatalf("only %d verified troubleshooting rows found", len(rows))
	}
	return rows
}

// Every name in a "Settings reference" table of a Section 10 lesson (a test, an option, an event, a threshold) must appear in a VERIFIED
// or OBSERVED row of a ts- fact sheet. Four lessons have such a table.
func TestSection10TablesExistInVerifiedSheetRows(t *testing.T) {
	c, err := Load(os.DirFS("../../../content"))
	if err != nil {
		t.Fatal(err)
	}
	rows := verifiedTroubleshootingRows(t, c)
	s, ok := c.Section("troubleshooting")
	if !ok {
		t.Fatal("section troubleshooting is missing")
	}
	withTable, checked := 0, 0
	for _, l := range s.Lessons {
		fields := settingsFields(l.Body)
		if len(fields) == 0 {
			continue
		}
		withTable++
		for _, f := range fields {
			checked++
			found := false
			for _, r := range rows {
				if strings.Contains(r, strings.ToLower(f)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: %q is in no VERIFIED or OBSERVED row of reference/troubleshooting/", l.Meta.ID, f)
			}
		}
	}
	if withTable != 4 || checked < 35 {
		t.Errorf("checked %d names in %d lessons, want 4 lessons and at least 35 names", checked, withTable)
	}
}

// The debug.py reference runbook lists options in its documented tables only if a VERIFIED or OBSERVED row of ts-edge-cli names them;
// everything else is under "Unconfirmed, fill from observation".
func TestDebugReferenceRunbookListsOnlyVerifiedOptions(t *testing.T) {
	c, err := Load(os.DirFS("../../../content"))
	if err != nil {
		t.Fatal(err)
	}
	rb, ok := c.Runbook("debug-script-reference")
	if !ok {
		t.Fatal("runbook debug-script-reference is missing")
	}
	rows := verifiedTroubleshootingRows(t, c)
	documented, unconfirmed := false, false
	inUnconfirmed := false
	for _, line := range strings.Split(rb.Body, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "## "):
			inUnconfirmed = strings.Contains(trim, "Unconfirmed")
			unconfirmed = unconfirmed || inUnconfirmed
		case !inUnconfirmed && strings.HasPrefix(trim, "| `--"):
			documented = true
			cells := strings.Split(strings.Trim(trim, "|"), "|")
			opt := strings.Trim(strings.TrimSpace(cells[0]), "`")
			found := false
			for _, r := range rows {
				if strings.Contains(r, strings.ToLower(opt)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("debug-script-reference lists %q as documented but no VERIFIED or OBSERVED row of a ts- sheet names it", opt)
			}
			if !strings.Contains(trim, "VERIFIED") && !strings.Contains(trim, "OBSERVED") {
				t.Errorf("debug-script-reference row for %q has no VERIFIED or OBSERVED status", opt)
			}
		}
	}
	if !documented || !unconfirmed {
		t.Errorf("the runbook needs documented option rows and an Unconfirmed section (documented=%v unconfirmed=%v)", documented, unconfirmed)
	}
}
