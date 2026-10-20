package content

import (
	"io/fs"
	"path"
	"sort"
	"strings"

	"velocloud-academy/backend/internal/sim"
)

// TopologyDoc is one topologies/*.yaml file.
type TopologyDoc struct {
	Path   string
	T      *sim.Topology
	Broken bool
}

// Topology is a validated teaching topology plus its non-fatal findings.
type Topology struct {
	Path     string
	T        *sim.Topology
	Warnings []sim.Issue
}

// readTopologies parses content/topologies/*.yaml (strict YAML; syntax problems only).
func readTopologies(fsys fs.FS, b *Bundle, errs *ErrorList) {
	entries, _ := fs.ReadDir(fsys, "topologies")
	for _, e := range entries {
		p := path.Join("topologies", e.Name())
		switch {
		case ignored(e.Name()):
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml"):
			doc := TopologyDoc{Path: p}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				errs.add(p, "cannot read file: %v", unwrapPathErr(err))
				doc.Broken = true
			} else if t, err := sim.Decode(data); err != nil {
				errs.add(p, "%v", err)
				doc.Broken = true
			} else {
				doc.T = t
			}
			b.Topologies = append(b.Topologies, doc)
		default:
			b.Extra = append(b.Extra, p)
		}
	}
}

// validateTopologies applies the simulator's rules to every readable topology and reports ALL findings.
func validateTopologies(errs *ErrorList, docs []TopologyDoc) {
	ids := map[string]string{}
	for _, d := range docs {
		if d.Broken || d.T == nil {
			continue
		}
		stem := strings.TrimSuffix(path.Base(d.Path), ".yaml")
		if d.T.Meta.ID != "" && d.T.Meta.ID != stem {
			errs.add(d.Path, "meta.id %q must equal the file name %q", d.T.Meta.ID, stem)
		}
		if prev, dup := ids[d.T.Meta.ID]; dup && d.T.Meta.ID != "" {
			errs.add(d.Path, "duplicate topology id %q (also defined in %s)", d.T.Meta.ID, prev)
		} else {
			ids[d.T.Meta.ID] = d.Path
		}
		for _, is := range sim.Errors(sim.Validate(d.T)) {
			errs.add(d.Path, "%s: %s", is.Path, is.Message)
		}
	}
}

// buildTopologies keeps the validated topologies in file order with their warnings.
func buildTopologies(docs []TopologyDoc) []*Topology {
	out := []*Topology{}
	for _, d := range docs {
		if d.Broken || d.T == nil {
			continue
		}
		var warns []sim.Issue
		for _, is := range sim.Validate(d.T) {
			if is.Level == sim.LevelWarning {
				warns = append(warns, is)
			}
		}
		out = append(out, &Topology{Path: d.Path, T: d.T, Warnings: warns})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Topologies returns the validated topologies, ordered by file name.
func (c *Course) Topologies() []*Topology { return c.topologies }

// Topology returns the topology with the given id.
func (c *Course) Topology(id string) (*Topology, bool) {
	for _, t := range c.topologies {
		if t.T.Meta.ID == id {
			return t, true
		}
	}
	return nil, false
}
