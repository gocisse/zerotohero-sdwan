package content

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"velocloud-academy/backend/internal/sim"
)

// LabDoc is one labs/*.yaml file.
type LabDoc struct {
	Path   string
	Lab    *sim.Lab
	Broken bool
}

// Lab is a validated lab.
type Lab struct {
	Path string
	Lab  *sim.Lab
}

// readLabs parses content/labs/*.yaml (strict YAML; syntax problems only).
func readLabs(fsys fs.FS, b *Bundle, errs *ErrorList) {
	entries, _ := fs.ReadDir(fsys, "labs")
	for _, e := range entries {
		p := path.Join("labs", e.Name())
		switch {
		case ignored(e.Name()) || e.Name() == "README.md":
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml"):
			doc := LabDoc{Path: p}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				errs.add(p, "cannot read file: %v", unwrapPathErr(err))
				doc.Broken = true
			} else {
				var l sim.Lab
				dec := yaml.NewDecoder(bytes.NewReader(data))
				dec.KnownFields(true)
				if err := dec.Decode(&l); err != nil {
					if errors.Is(err, io.EOF) {
						errs.add(p, "file is empty")
					} else {
						errs.add(p, "%v", err)
					}
					doc.Broken = true
				} else {
					doc.Lab = &l
				}
			}
			b.Labs = append(b.Labs, doc)
		default:
			b.Extra = append(b.Extra, p)
		}
	}
}

type labCtx struct {
	topologies map[string]*sim.Topology
	lessons    map[string]string
}

func (c labCtx) Topology(id string) *sim.Topology { return c.topologies[id] }
func (c labCtx) HasLesson(id string) bool         { _, ok := c.lessons[id]; return ok }

// validateLabs applies the lab rules to every readable lab and reports ALL findings, like the other validators.
func validateLabs(errs *ErrorList, docs []LabDoc, topos []TopologyDoc, refs []ReferenceDoc, lessonPath map[string]string, lessonsBroken bool) {
	confirmed := []string{}
	for _, r := range SheetRows(refs) {
		if r.Status == "VERIFIED" || r.Status == "OBSERVED" {
			confirmed = append(confirmed, r.Claim)
		}
	}
	ctx := labCtx{topologies: map[string]*sim.Topology{}, lessons: lessonPath}
	for _, d := range topos {
		if !d.Broken && d.T != nil && len(sim.Errors(sim.Validate(d.T))) == 0 {
			ctx.topologies[d.T.Meta.ID] = d.T
		}
	}
	ids := map[string]string{}
	for _, d := range docs {
		if d.Broken || d.Lab == nil {
			continue
		}
		stem := strings.TrimSuffix(path.Base(d.Path), ".yaml")
		if d.Lab.ID != "" && d.Lab.ID != stem {
			errs.add(d.Path, "id %q must equal the file name %q", d.Lab.ID, stem)
		}
		if prev, dup := ids[d.Lab.ID]; dup && d.Lab.ID != "" {
			errs.add(d.Path, "duplicate lab id %q (also defined in %s)", d.Lab.ID, prev)
		} else {
			ids[d.Lab.ID] = d.Path
		}
		lessons := ctx
		if lessonsBroken {
			// A lesson that failed to parse could be the cause of an unknown-lesson error: do not report it twice.
			lessons.lessons = map[string]string{}
			for _, g := range d.Lab.Goals {
				lessons.lessons[g] = ""
			}
		}
		for _, msg := range sim.ValidateLab(d.Lab, lessons) {
			errs.add(d.Path, "%s", msg)
		}
		for i, db := range d.Lab.Debrief {
			if db.Unverified || strings.TrimSpace(db.Where) == "" {
				continue
			}
			found := false
			for _, c := range confirmed {
				if strings.Contains(c, db.Where) {
					found = true
					break
				}
			}
			if !found {
				errs.add(d.Path, "debrief[%d].where: %q is not wording of any VERIFIED or OBSERVED fact-sheet row; use the guides' menu names or mark the line unverified with a note", i, db.Where)
			}
		}
	}
}

func buildLabs(docs []LabDoc) []*Lab {
	out := []*Lab{}
	for _, d := range docs {
		if !d.Broken && d.Lab != nil {
			out = append(out, &Lab{Path: d.Path, Lab: d.Lab})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Labs returns the validated labs, ordered by file name.
func (c *Course) Labs() []*Lab { return c.labs }

// Lab returns the lab with the given id.
func (c *Course) Lab(id string) (*Lab, bool) {
	for _, l := range c.labs {
		if l.Lab.ID == id {
			return l, true
		}
	}
	return nil, false
}
