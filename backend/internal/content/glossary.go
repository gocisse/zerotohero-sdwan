package content

import (
	"regexp"
	"strings"
)

// GlossaryTerm is one entry of content/glossary.yaml.
type GlossaryTerm struct {
	ID        string   `yaml:"id" json:"id"`
	Term      string   `yaml:"term" json:"term"`
	Aliases   []string `yaml:"aliases" json:"aliases"`
	Short     string   `yaml:"short" json:"short"`
	Long      string   `yaml:"long" json:"long,omitempty"`
	Related   []string `yaml:"related" json:"related"`
	LessonRef string   `yaml:"lesson_ref" json:"lesson_ref,omitempty"`
}

// ---- MDX component references ----

var (
	fenceRe  = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")
	inlineRe = regexp.MustCompile("`[^`\n]*`")
	idAttrRe = regexp.MustCompile(`\bid\s*=\s*(?:"([^"]*)"|'([^']*)'|\{\s*"([^"]*)"\s*\}|\{\s*'([^']*)'\s*\})`)
)

// tagRef is one <Component ...> occurrence in a lesson body.
type tagRef struct {
	ID    string
	HasID bool
	Line  int // 1-based line within the lesson body
}

// blank replaces s with spaces, keeping newlines so line numbers stay correct.
func blank(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		return ' '
	}, s)
}

// scanTags finds <name ...> elements in MDX, ignoring fenced and inline code.
func scanTags(body, name string) []tagRef {
	body = fenceRe.ReplaceAllStringFunc(body, blank)
	body = inlineRe.ReplaceAllStringFunc(body, blank)
	tag := regexp.MustCompile(`<` + name + `\b([^>]*)>`)
	var out []tagRef
	for _, m := range tag.FindAllStringSubmatchIndex(body, -1) {
		ref := tagRef{Line: 1 + strings.Count(body[:m[0]], "\n")}
		attrs := body[m[2]:m[3]]
		if am := idAttrRe.FindStringSubmatch(attrs); am != nil {
			ref.HasID = true
			for _, g := range am[1:] {
				if g != "" {
					ref.ID = g
				}
			}
		}
		out = append(out, ref)
	}
	return out
}

func validateGlossary(errs *ErrorList, path string, terms []GlossaryTerm, lessonIDs map[string]string, anyBrokenLesson bool) {
	ids := map[string]bool{}
	names := map[string]string{} // lowercased term/alias -> term id
	for _, t := range terms {
		name := "term " + t.ID
		if t.ID == "" {
			name = "a term"
			errs.add(path, "%s: id is required", name)
		} else {
			if !idPattern.MatchString(t.ID) {
				errs.add(path, "%s: id must be lowercase letters, digits and single hyphens", name)
			}
			if ids[t.ID] {
				errs.add(path, "%s: duplicate term id", name)
			}
			ids[t.ID] = true
		}
		if strings.TrimSpace(t.Term) == "" {
			errs.add(path, "%s: term is required", name)
		}
		if strings.TrimSpace(t.Short) == "" {
			errs.add(path, "%s: short (a one-sentence definition) is required", name)
		}
		for _, n := range append([]string{t.Term}, t.Aliases...) {
			key := strings.ToLower(strings.TrimSpace(n))
			if key == "" {
				if n != t.Term {
					errs.add(path, "%s: aliases contains an empty alias", name)
				}
				continue
			}
			if other, dup := names[key]; dup && other != t.ID {
				errs.add(path, "%s: %q is also the name or alias of term %q (search would be ambiguous)", name, n, other)
			}
			names[key] = t.ID
		}
	}
	for _, t := range terms {
		for _, r := range t.Related {
			switch {
			case r == t.ID:
				errs.add(path, "term %s: lists itself as related", t.ID)
			case !ids[r]:
				errs.add(path, "term %s: related term %q does not exist", t.ID, r)
			}
		}
		if t.LessonRef != "" {
			if _, ok := lessonIDs[t.LessonRef]; !ok && !anyBrokenLesson {
				errs.add(path, "term %s: lesson_ref %q does not match any lesson id", t.ID, t.LessonRef)
			}
		}
	}
}
