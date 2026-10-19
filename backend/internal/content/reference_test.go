package content

import (
	"strings"
	"testing"
)

const goodSheet = `# Topic

| claim | source URL | doc version | status |
|---|---|---|---|
| A verified claim. | https://example.com/a | 7.0 | VERIFIED |
| A secondary claim. | https://example.com/b | n/a | SECONDARY |
| Could not confirm. | none | n/a | VERIFY |

Some prose between tables.

| Claim | Source URL | Doc version | Status |
| --- | --- | --- | --- |
| Second table. | https://example.com/c | 7.0 | VERIFIED |
`

func TestValidateReference(t *testing.T) {
	tests := []struct {
		name, path, content, wantMsg string
	}{
		{"valid sheet with two tables", "reference/a.md", goodSheet, ""},
		{"no claims table", "reference/a.md", "# Notes only\n\nNo table here.\n", "no claims table"},
		{"other table header", "reference/a.md", "| fact | source |\n|---|---|\n| x | y |\n", "no claims table"},
		{"wrong cell count", "reference/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| only | three | cells |\n", "exactly 4 cells"},
		{"unknown status", "reference/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | https://e.com | 7.0 | CONFIRMED |\n", `status "CONFIRMED"`},
		{"verified without url", "reference/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | trust me | 7.0 | VERIFIED |\n", "must cite a source URL"},
		{"secondary without url", "reference/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | — | 7.0 | SECONDARY |\n", "must cite a source URL"},
		{"empty claim", "reference/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n|  | https://e.com | 7.0 | VERIFIED |\n", "claim text is empty"},
		{"observed with dated source", "reference/observations/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | user observation 2026-10-20 | 7.0 | OBSERVED |\n", ""},
		{"observed without date", "reference/observations/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | I saw it | 7.0 | OBSERVED |\n", "user observation YYYY-MM-DD"},
		{"observed with a url is refused", "reference/observations/a.md", "| claim | source URL | doc version | status |\n|---|---|---|---|\n| x | https://e.com | 7.0 | OBSERVED |\n", "user observation YYYY-MM-DD"},
		{"not markdown", "reference/a.txt", goodSheet, "must be Markdown"},
		{"CRLF line endings are fine", "reference/a.md", strings.ReplaceAll(goodSheet, "\n", "\r\n"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errs ErrorList
			validateReference(&errs, ReferenceDoc{Path: tt.path, Content: tt.content})
			if tt.wantMsg == "" {
				if len(errs) != 0 {
					t.Fatalf("expected valid, got %v", errs)
				}
				return
			}
			for _, e := range errs {
				if e.File == tt.path && strings.Contains(e.Message, tt.wantMsg) {
					return
				}
			}
			t.Fatalf("no error containing %q; got %v", tt.wantMsg, errs)
		})
	}
}

func TestReferenceFolderIsNotAnOrphanAndIsValidated(t *testing.T) {
	b := validBundle()
	b.References = []ReferenceDoc{{Path: "reference/ok.md", Content: goodSheet}, {Path: "reference/bad.md", Content: "# no table\n"}}
	errs := Validate(b)
	if len(errs) != 1 || errs[0].File != "reference/bad.md" {
		t.Fatalf("want exactly one error for reference/bad.md, got %v", errs)
	}
}
