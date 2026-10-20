package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Patch is one change a lab applies to a COPY of a topology. Paths use "/" between names and "[id]" to pick the
// array element whose "id" is id, for example edges[spoke-1-edge]/overrides/business_rules[spoke-1-video]/action.
// The alias edges[<id>] at the root finds an Edge in any site. Ops: set (creates missing objects), remove, append.
type Patch struct {
	Op    string `yaml:"op" json:"op"`
	Path  string `yaml:"path" json:"path"`
	Value any    `yaml:"value" json:"value,omitempty"`
}

// PatchError describes a patch that cannot be applied.
type PatchError struct{ Message string }

func (e *PatchError) Error() string { return e.Message }

// ApplyPatches returns a deep copy of the topology with the patches applied, in order. The original is never
// changed. Unknown fields and type mismatches are errors, so a typo in a lab cannot silently do nothing.
func ApplyPatches(t *Topology, patches []Patch) (*Topology, error) {
	raw, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for i, p := range patches {
		if err := applyOne(doc, p); err != nil {
			return nil, &PatchError{fmt.Sprintf("patch %d (%s %s): %v", i+1, p.Op, p.Path, err)}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	var nt Topology
	if err := dec.Decode(&nt); err != nil {
		return nil, &PatchError{"the patched topology is not a valid document: " + err.Error()}
	}
	return &nt, nil
}

type pathStep struct{ name, id string }

func parsePath(p string) ([]pathStep, error) {
	var steps []pathStep
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return nil, fmt.Errorf("empty path segment")
		}
		st := pathStep{name: seg}
		if i := strings.Index(seg, "["); i >= 0 {
			if !strings.HasSuffix(seg, "]") {
				return nil, fmt.Errorf("unterminated [ in %q", seg)
			}
			st.name, st.id = seg[:i], seg[i+1:len(seg)-1]
		}
		steps = append(steps, st)
	}
	return steps, nil
}

func findByID(list []any, id string) int {
	for i, x := range list {
		if m, ok := x.(map[string]any); ok && m["id"] == id {
			return i
		}
	}
	return -1
}

// edgeLoc finds an Edge object of the root document.
func edgeLoc(doc map[string]any, id string) map[string]any {
	sites, _ := doc["sites"].([]any)
	for _, s := range sites {
		sm, _ := s.(map[string]any)
		edges, _ := sm["edges"].([]any)
		if i := findByID(edges, id); i >= 0 {
			return edges[i].(map[string]any)
		}
	}
	return nil
}

func applyOne(doc map[string]any, p Patch) error {
	steps, err := parsePath(p.Path)
	if err != nil {
		return err
	}
	var cur any = doc
	for i, st := range steps {
		last := i == len(steps)-1
		m, ok := cur.(map[string]any)
		if !ok {
			return fmt.Errorf("%q is not an object", steps[i-1].name)
		}
		// Root alias for Edges.
		if i == 0 && st.name == "edges" && st.id != "" {
			e := edgeLoc(doc, st.id)
			if e == nil {
				return fmt.Errorf("no Edge %q", st.id)
			}
			if last {
				return fmt.Errorf("an Edge itself cannot be replaced or removed")
			}
			cur = e
			continue
		}
		if st.id == "" {
			if last {
				return finish(m, st.name, p)
			}
			next, ok := m[st.name]
			if !ok || next == nil {
				if p.Op != "set" && p.Op != "append" {
					return fmt.Errorf("%q does not exist", st.name)
				}
				next = map[string]any{}
				m[st.name] = next
			}
			cur = next
			continue
		}
		list, ok := m[st.name].([]any)
		if !ok {
			return fmt.Errorf("%q is not a list", st.name)
		}
		idx := findByID(list, st.id)
		if idx < 0 {
			return fmt.Errorf("%q has no element with id %q", st.name, st.id)
		}
		if last {
			switch p.Op {
			case "remove":
				m[st.name] = append(append([]any{}, list[:idx]...), list[idx+1:]...)
				return nil
			case "set":
				v, err := normalise(p.Value)
				if err != nil {
					return err
				}
				list[idx] = v
				return nil
			}
			return fmt.Errorf("op %q cannot target a list element", p.Op)
		}
		cur = list[idx]
	}
	return nil
}

func normalise(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func finish(m map[string]any, name string, p Patch) error {
	switch p.Op {
	case "set":
		v, err := normalise(p.Value)
		if err != nil {
			return err
		}
		m[name] = v
	case "remove":
		if _, ok := m[name]; !ok {
			return fmt.Errorf("%q does not exist", name)
		}
		delete(m, name)
	case "append":
		v, err := normalise(p.Value)
		if err != nil {
			return err
		}
		list, _ := m[name].([]any)
		m[name] = append(list, v)
	default:
		return fmt.Errorf("unknown op %q (set, remove or append)", p.Op)
	}
	return nil
}
