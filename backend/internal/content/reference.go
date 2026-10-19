package content

import (
	"regexp"
	"strings"
)

// Reference fact sheets (content/reference/*.md) record, per claim, where it came from and how well it is
// confirmed. Every sheet other than README.md must contain at least one claims table whose header is
// "claim | source URL | doc version | status"; rows must have exactly four cells and a known status.
var (
	claimsHeaderRe = regexp.MustCompile(`(?i)^\s*\|\s*claim\s*\|\s*source url\s*\|\s*doc version\s*\|\s*status\s*\|\s*$`)
	separatorRe    = regexp.MustCompile(`^\s*\|?\s*:?-{3,}`)
	// ReferenceStatuses are the allowed values of the status column.
	ReferenceStatuses = []string{"VERIFIED", "SECONDARY", "VERIFY", "OBSERVED"}
	// observationSourceRe is the only accepted source for an OBSERVED row: a dated user observation.
	observationSourceRe = regexp.MustCompile(`^user observation \d{4}-\d{2}-\d{2}$`)
)

func validateReference(errs *ErrorList, doc ReferenceDoc) {
	if !strings.HasSuffix(doc.Path, ".md") {
		errs.add(doc.Path, "reference files must be Markdown (.md)")
		return
	}
	headers := 0
	inTable := false
	for i, line := range strings.Split(strings.ReplaceAll(doc.Content, "\r\n", "\n"), "\n") {
		n := i + 1
		trim := strings.TrimSpace(line)
		switch {
		case claimsHeaderRe.MatchString(line):
			headers++
			inTable = true
		case inTable && separatorRe.MatchString(line) && strings.Contains(trim, "-"):
			// header separator row
		case inTable && strings.HasPrefix(trim, "|"):
			cells := strings.Split(strings.Trim(trim, "|"), "|")
			if len(cells) != 4 {
				errs.add(doc.Path, "line %d: claims table row needs exactly 4 cells (claim | source URL | doc version | status), got %d", n, len(cells))
				continue
			}
			status := strings.TrimSpace(cells[3])
			if !containsString(ReferenceStatuses, status) {
				errs.add(doc.Path, "line %d: status %q must be one of VERIFIED, SECONDARY, VERIFY, OBSERVED", n, status)
				continue
			}
			src := strings.TrimSpace(cells[1])
			if status == "OBSERVED" {
				if !observationSourceRe.MatchString(src) {
					errs.add(doc.Path, "line %d: an OBSERVED claim must have the source \"user observation YYYY-MM-DD\" (got %q)", n, src)
				}
			} else if status != "VERIFY" && !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
				errs.add(doc.Path, "line %d: a %s claim must cite a source URL (got %q); use VERIFY if there is no source", n, status, src)
			}
			if strings.TrimSpace(cells[0]) == "" {
				errs.add(doc.Path, "line %d: claim text is empty", n)
			}
		default:
			inTable = false
		}
	}
	if headers == 0 {
		errs.add(doc.Path, "no claims table: add a table with the header `| claim | source URL | doc version | status |`")
	}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
