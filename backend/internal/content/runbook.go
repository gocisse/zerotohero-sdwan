package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// RunbookKinds are the allowed values of a runbook's kind (step 20).
var RunbookKinds = []string{"checklist", "cheatsheet", "template", "reference"}

// RunbookMeta is a runbook's YAML frontmatter. A runbook is a one-page MDX document under content/runbooks/.
type RunbookMeta struct {
	ID               string `yaml:"id"`
	Title            string `yaml:"title"`
	Summary          string `yaml:"summary"`
	Kind             string `yaml:"kind"`
	EstimatedMinutes int    `yaml:"estimated_minutes"`
}

// RunbookDoc is one runbooks/*.mdx file as read from disk.
type RunbookDoc struct {
	Path     string
	Meta     RunbookMeta
	Body     string
	BodyLine int
	Broken   bool
}

// Runbook is a validated runbook.
type Runbook struct {
	Meta RunbookMeta
	Path string
	Body string
}

// readRunbooks parses content/runbooks/*.mdx (frontmatter and body; syntax problems only).
func readRunbooks(fsys fs.FS, b *Bundle, errs *ErrorList) {
	entries, _ := fs.ReadDir(fsys, "runbooks")
	for _, e := range entries {
		p := path.Join("runbooks", e.Name())
		switch {
		case ignored(e.Name()) || e.Name() == "README.md":
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".mdx"):
			doc := RunbookDoc{Path: p}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				errs.add(p, "cannot read file: %v", unwrapPathErr(err))
				doc.Broken = true
			} else if front, body, offset, err := splitFrontmatter(data); err != nil {
				errs.add(p, "frontmatter: %v", err)
				doc.Broken = true
			} else if err := decodeStrict(front, &doc.Meta); err != nil {
				errs.add(p, "frontmatter: %v", err)
				doc.Broken = true
			} else {
				doc.Body, doc.BodyLine = body, offset
			}
			b.Runbooks = append(b.Runbooks, doc)
		default:
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			b.Extra = append(b.Extra, p+suffix)
		}
	}
}

// validateRunbooks reports every problem of every runbook together: missing fields, a bad kind, duplicate ids and unknown
// <Term> or <Diagram> ids.
func (v *validator) runbooks(anyBrokenDiagram, glossaryBroken bool) {
	seen := map[string]string{}
	for _, r := range v.b.Runbooks {
		if r.Broken {
			continue
		}
		p, m := r.Path, r.Meta
		v.requireID(p, "id", m.ID)
		if m.ID != "" {
			if prev, dup := seen[m.ID]; dup {
				v.errs.add(p, "duplicate runbook id %q (also defined in %s)", m.ID, prev)
			}
			seen[m.ID] = p
		}
		if strings.TrimSpace(m.Title) == "" {
			v.errs.add(p, "frontmatter: title is required")
		}
		if strings.TrimSpace(m.Summary) == "" {
			v.errs.add(p, "frontmatter: summary is required")
		}
		if m.EstimatedMinutes <= 0 {
			v.errs.add(p, "frontmatter: estimated_minutes must be a positive number")
		}
		ok := false
		for _, k := range RunbookKinds {
			ok = ok || m.Kind == k
		}
		if !ok {
			v.errs.add(p, "frontmatter: kind must be one of %s (got %q)", strings.Join(RunbookKinds, ", "), m.Kind)
		}
		if strings.TrimSpace(r.Body) == "" {
			v.errs.add(p, "runbook body is empty")
		}
		at := func(line int) string { return fmt.Sprintf("line %d", line+r.BodyLine) }
		for _, ref := range scanTags(r.Body, "Diagram") {
			switch {
			case !ref.HasID || ref.ID == "":
				v.errs.add(p, "%s: <Diagram> needs an id=\"...\" attribute", at(ref.Line))
			case v.diagramPath[ref.ID] == "" && !anyBrokenDiagram:
				v.errs.add(p, "%s: <Diagram id=%q> does not match any diagram in diagrams/", at(ref.Line), ref.ID)
			}
		}
		for _, ref := range scanTags(r.Body, "Term") {
			switch {
			case !ref.HasID || ref.ID == "":
				v.errs.add(p, "%s: <Term> needs an id=\"...\" attribute", at(ref.Line))
			case !v.termIDs[ref.ID] && !glossaryBroken:
				v.errs.add(p, "%s: <Term id=%q> does not match any term in glossary.yaml", at(ref.Line), ref.ID)
			}
		}
	}
}

func buildRunbooks(docs []RunbookDoc) []*Runbook {
	var out []*Runbook
	for _, d := range docs {
		out = append(out, &Runbook{Meta: d.Meta, Path: d.Path, Body: d.Body})
	}
	return out
}

// Runbooks returns the runbooks in file-name order.
func (c *Course) Runbooks() []*Runbook { return c.runbooks }

// Runbook returns the runbook with the given id.
func (c *Course) Runbook(id string) (*Runbook, bool) {
	for _, r := range c.runbooks {
		if r.Meta.ID == id {
			return r, true
		}
	}
	return nil, false
}
