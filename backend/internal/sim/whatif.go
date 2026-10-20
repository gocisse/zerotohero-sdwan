package sim

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Change types accepted by WhatIf.
const (
	ChSetProfileSetting    = "set_profile_setting"
	ChRemoveProfileSetting = "remove_profile_setting"
	ChSetEdgeOverride      = "set_edge_override"
	ChRemoveEdgeOverride   = "remove_edge_override"
	ChAssignProfile        = "assign_profile"
)

// Change is one hypothetical change. Fields used depend on Type.
type Change struct {
	Type    string `json:"type"`
	Profile string `json:"profile,omitempty"` // profile to change, or the target Profile for assign_profile
	Edge    string `json:"edge,omitempty"`
	Setting string `json:"setting,omitempty"`
	Value   any    `json:"value,omitempty"`
}

// ChangeError is returned for a change that cannot be applied (unknown ids, wrong value type...).
type ChangeError struct{ Message string }

func (e *ChangeError) Error() string { return e.Message }

// Note is an explanation attached to an Edge's diff, with its evidence.
type Note struct {
	Text  string `json:"text"`
	Basis Basis  `json:"basis"`
}

// DiffItem is one effective value that differs before and after.
type DiffItem struct {
	Key          string  `json:"key"`
	Kind         string  `json:"kind"`
	Label        string  `json:"label"`
	Before       any     `json:"before"`
	After        any     `json:"after"`
	BeforeSource *Source `json:"before_source,omitempty"`
	AfterSource  *Source `json:"after_source,omitempty"`
	Basis        Basis   `json:"basis"`
}

// EdgeDiff is the effect of the change on one Edge.
type EdgeDiff struct {
	EdgeID   string `json:"edge_id"`
	EdgeName string `json:"edge_name"`
	Profile  string `json:"profile"`
	// Status is "changed", "kept-override" (the Edge overrides the changed setting) or "unchanged".
	Status   string     `json:"status"`
	Items    []DiffItem `json:"items"`
	Notes    []Note     `json:"notes"`
	Reverted []string   `json:"reverted_interfaces,omitempty"`
}

// WhatIfResult is the diff of effective configuration for every Edge.
type WhatIfResult struct {
	Change      Change        `json:"change"`
	Summary     string        `json:"summary"`
	Edges       []EdgeDiff    `json:"edges"`
	Issues      []Issue       `json:"issues"`
	NotModelled []NotModelled `json:"not_modelled"`
}

// WhatIf applies a hypothetical change to a copy of the topology and diffs every Edge's effective
// configuration. The input topology is never modified.
func WhatIf(t *Topology, ch Change) (*WhatIfResult, error) {
	before, err := clone(t)
	if err != nil {
		return nil, err
	}
	after, err := clone(t)
	if err != nil {
		return nil, err
	}
	ap, err := applyChange(after, ch)
	if err != nil {
		return nil, err
	}

	res := &WhatIfResult{Change: ch, Summary: ap.summary, Edges: []EdgeDiff{}, Issues: []Issue{}, NotModelled: []NotModelled{}}
	for si := range after.Sites {
		for ei := range after.Sites[si].Edges {
			id := after.Sites[si].Edges[ei].ID
			b, err1 := EffectiveConfig(before, id)
			a, err2 := EffectiveConfig(after, id)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("what-if: edge %s: %v %v", id, err1, err2)
			}
			d := EdgeDiff{EdgeID: id, EdgeName: a.Edge.Name, Profile: a.Edge.Profile, Items: diffEffective(b, a), Notes: ap.notes[id], Reverted: ap.reverted[id]}
			if d.Notes == nil {
				d.Notes = []Note{}
			}
			switch {
			case len(d.Items) > 0:
				d.Status = "changed"
			case ap.kept[id]:
				d.Status = "kept-override"
			default:
				d.Status = "unchanged"
			}
			res.Edges = append(res.Edges, d)
		}
	}
	existing := map[string]bool{}
	for _, i := range Validate(before) {
		existing[i.String()] = true
	}
	for _, i := range Validate(after) {
		if existing[i.String()] || revertedVlanIssue(ap, i) {
			continue
		}
		res.Issues = append(res.Issues, i)
	}
	if ap.notModelled != nil {
		res.NotModelled = ap.notModelled
	}
	return res, nil
}

// revertedVlanIssue reports whether i is the "LAN interface needs a VLAN" finding for an interface that a Profile switch
// just reverted: losing the VLAN is the documented effect, not an authoring mistake. // rule: profile-switch-vlans
func revertedVlanIssue(ap *applied, i Issue) bool {
	for edge, ifaces := range ap.reverted {
		for _, id := range ifaces {
			if strings.HasPrefix(i.Path, "edges["+edge+"].interfaces["+id+"].vlan") {
				return true
			}
		}
	}
	return false
}

type applied struct {
	summary     string
	notes       map[string][]Note
	kept        map[string]bool
	reverted    map[string][]string
	notModelled []NotModelled
}

func (a *applied) note(edge, text string, b Basis) {
	a.notes[edge] = append(a.notes[edge], Note{Text: text, Basis: b})
}

func clone(t *Topology) (*Topology, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var out Topology
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *Topology) profileByID(id string) *Profile {
	for i := range t.Profiles {
		if t.Profiles[i].ID == id {
			return &t.Profiles[i]
		}
	}
	return nil
}

func applyChange(t *Topology, ch Change) (*applied, error) {
	ap := &applied{notes: map[string][]Note{}, kept: map[string]bool{}, reverted: map[string][]string{}}
	checkSetting := func() (SettingDef, error) {
		def, ok := SettingByID(ch.Setting)
		if !ok {
			return def, &ChangeError{fmt.Sprintf("unknown setting %q; known settings: %v", ch.Setting, SettingIDs())}
		}
		if ch.Type == ChSetProfileSetting || ch.Type == ChSetEdgeOverride {
			if msg := def.checkValue(ch.Value); msg != "" {
				return def, &ChangeError{msg}
			}
		}
		return def, nil
	}
	switch ch.Type {
	case ChSetProfileSetting, ChRemoveProfileSetting:
		p := t.profileByID(ch.Profile)
		if p == nil {
			return nil, &ChangeError{fmt.Sprintf("unknown profile %q", ch.Profile)}
		}
		def, err := checkSetting()
		if err != nil {
			return nil, err
		}
		if p.Device == nil {
			p.Device = map[string]any{}
		}
		if ch.Type == ChSetProfileSetting {
			p.Device[ch.Setting] = ch.Value
			ap.summary = fmt.Sprintf("Profile %s: set %s to %v", p.Name, def.Label, ch.Value)
		} else {
			delete(p.Device, ch.Setting)
			ap.summary = fmt.Sprintf("Profile %s: remove %s", p.Name, def.Label)
		}
		for si := range t.Sites {
			for ei := range t.Sites[si].Edges {
				e := &t.Sites[si].Edges[ei]
				if e.Profile != p.ID {
					continue
				}
				if _, ov := e.Overrides.Device[ch.Setting]; ov { // rule: assume-5
					ap.kept[e.ID] = true
					ap.note(e.ID, fmt.Sprintf("%s overrides %s, so the Profile change is assumed not to reach it.", e.Name, def.Label), basisOf(RA5ProfileEditSkips))
				}
			}
		}
	case ChSetEdgeOverride, ChRemoveEdgeOverride:
		e, _, _, err := t.FindEdge(ch.Edge)
		if err != nil {
			return nil, &ChangeError{err.Error()}
		}
		def, err := checkSetting()
		if err != nil {
			return nil, err
		}
		if ch.Type == ChSetEdgeOverride { // rule: layer-merge
			if e.Overrides.Device == nil {
				e.Overrides.Device = map[string]any{}
			}
			e.Overrides.Device[ch.Setting] = ch.Value
			ap.summary = fmt.Sprintf("%s: override %s with %v", e.Name, def.Label, ch.Value)
		} else {
			if _, ok := e.Overrides.Device[ch.Setting]; !ok {
				return nil, &ChangeError{fmt.Sprintf("%s has no override for %s", e.Name, def.Label)}
			}
			delete(e.Overrides.Device, ch.Setting) // rule: assume-8
			ap.summary = fmt.Sprintf("%s: remove the override of %s", e.Name, def.Label)
			ap.note(e.ID, "The setting falls back to the Profile value (how a real override is reverted is not documented).", basisOf(RA8RevertToProfile))
		}
	case ChAssignProfile:
		e, _, old, err := t.FindEdge(ch.Edge)
		if err != nil {
			return nil, &ChangeError{err.Error()}
		}
		np := t.profileByID(ch.Profile)
		if np == nil {
			return nil, &ChangeError{fmt.Sprintf("unknown profile %q", ch.Profile)}
		}
		if np.ID == old.ID {
			return nil, &ChangeError{fmt.Sprintf("%s already uses Profile %s", e.Name, np.Name)}
		}
		ap.summary = fmt.Sprintf("%s: switch from Profile %s to Profile %s", e.Name, old.Name, np.Name)
		switchProfile(ap, e, old, np)
	default:
		return nil, &ChangeError{fmt.Sprintf("unknown change type %q", ch.Type)}
	}
	return ap, nil
}

// switchProfile applies the documented side effects of moving an Edge to another Profile.
func switchProfile(ap *applied, e *Edge, old, np *Profile) {
	newIDs := map[int]bool{}
	for _, v := range np.VLANs {
		newIDs[v.ID] = true
	}
	removed := map[int]bool{}
	for _, v := range old.VLANs { // rule: profile-switch-vlans
		if !newIDs[v.ID] { // rule: assume-35
			removed[v.ID] = true
		}
	}
	if len(removed) > 0 {
		ids := make([]int, 0, len(removed))
		for id := range removed {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		ap.note(e.ID, fmt.Sprintf("VLANs inherited from %s and not defined by %s are removed: %v.", old.Name, np.Name, ids), basisOf(RSwitchVlans))
		ap.note(e.ID, "VLANs are matched by id (a VLAN id the new Profile also defines is kept).", basisOf(RA35SwitchVlanByID))
	}
	for i := range e.Interfaces {
		in := &e.Interfaces[i]
		if in.VLAN != nil && removed[*in.VLAN] { // rule: profile-switch-vlans
			ap.reverted[e.ID] = append(ap.reverted[e.ID], in.ID)
			ap.note(e.ID, fmt.Sprintf("Interface %s was tied to removed VLAN %d, so it reverts to the new Profile's configuration, even if it was overridden.", in.ID, *in.VLAN), basisOf(RSwitchVlans))
			in.VLAN, in.Override = nil, false
		}
	}
	kept := e.Overrides.VLANs[:0]
	for _, vo := range e.Overrides.VLANs {
		if !removed[vo.ID] {
			kept = append(kept, vo)
		}
	}
	e.Overrides.VLANs = kept
	e.Profile = np.ID
	// rule: nm-profile-switch-bgp
	nm := notModelled(RNMSwitchBGP, "Profile-level BGP filters on a Profile switch")
	ap.notModelled = append(ap.notModelled, nm)
	ap.note(e.ID, "Profile-level BGP filters tied to neighbours would be removed (routing is not modelled yet).", nm.Basis)
	if len(e.Overrides.Device) > 0 || len(e.Overrides.BusinessRules) > 0 || e.Overrides.Firewall.Override { // rule: nm-profile-switch-overrides
		nm2 := notModelled(RNMSwitchOverrides, "Edge overrides after a Profile switch")
		ap.notModelled = append(ap.notModelled, nm2)
		ap.note(e.ID, "The Edge's other overrides are shown unchanged; what a real Profile switch does to them is not documented.", nm2.Basis)
	}
	for _, r := range e.Overrides.BusinessRules { // rule: assume-38
		if !hasString(np.Segments, r.Segment) {
			ap.note(e.ID, fmt.Sprintf("Edge rule %s is on segment %s, which Profile %s does not include; it is kept and flagged.", r.ID, r.Segment, np.Name), basisOf(RA38SwitchEdgeRules))
		}
	}
}

type flatItem struct {
	Key, Kind, Label string
	Value            any
	Source           *Source
	Basis            Basis
}

func flatten(ec *Effective) map[string]flatItem {
	m := map[string]flatItem{}
	put := func(f flatItem) { m[f.Key] = f }
	put(flatItem{Key: "profile", Kind: "profile", Label: "Profile", Value: ec.Edge.Profile, Basis: basisOf(ROneProfile)})
	ids := make([]string, 0, len(ec.Segments))
	for _, s := range ec.Segments {
		ids = append(ids, s.ID)
	}
	put(flatItem{Key: "segments", Kind: "segments", Label: "Segments", Value: ids, Basis: basisOf(RGlobalSegment)})
	for _, d := range ec.Device {
		src := d.Source
		put(flatItem{Key: "device:" + d.ID, Kind: "setting", Label: d.Label, Value: d.Value, Source: &src, Basis: d.Basis})
	}
	for _, v := range ec.VLANs {
		src := v.Source
		put(flatItem{Key: fmt.Sprintf("vlan:%d", v.ID), Kind: "vlan", Label: fmt.Sprintf("VLAN %d", v.ID),
			Value: map[string]any{"name": v.Name, "settings": v.Settings}, Source: &src, Basis: v.Basis})
	}
	for _, in := range ec.Interfaces {
		src := in.Source
		put(flatItem{Key: "interface:" + in.ID, Kind: "interface", Label: "Interface " + in.ID,
			Value: map[string]any{"vlan": in.VLAN, "segments": in.Segments, "override": in.Override}, Source: &src, Basis: in.Basis})
	}
	for _, sp := range ec.BusinessPolicy {
		for _, grp := range [][]EffectiveRule{sp.EdgeOverrides, sp.FromProfile} {
			for _, r := range grp {
				src := r.Owner
				put(flatItem{Key: "biz:" + sp.Segment.ID + ":" + r.Rule.ID, Kind: "business-rule", Label: "Business rule " + r.Rule.ID + " (" + sp.Segment.ID + ")",
					Value: map[string]any{"action": r.Rule.Action, "service": r.EffectiveService, "superseded_by": r.SupersededBy}, Source: &src, Basis: r.Basis})
			}
		}
	}
	m2 := ec.Firewall
	put(flatItem{Key: "fw:override", Kind: "firewall", Label: "Firewall Override", Value: m2.Override, Basis: basisOf(RFwOverrideTick)})
	for i, r := range m2.Rules {
		src := r.Owner
		put(flatItem{Key: "fw:" + r.Rule.ID, Kind: "firewall-rule", Label: "Firewall rule " + r.Rule.ID,
			Value: map[string]any{"action": r.Rule.Action, "group": r.Group, "position": i + 1}, Source: &src, Basis: r.Basis})
	}
	return m
}

func jsonKey(v any) string { b, _ := json.Marshal(v); return string(b) }

func diffEffective(b, a *Effective) []DiffItem {
	fb, fa := flatten(b), flatten(a)
	keys := map[string]bool{}
	for k := range fb {
		keys[k] = true
	}
	for k := range fa {
		keys[k] = true
	}
	order := make([]string, 0, len(keys))
	for k := range keys {
		order = append(order, k)
	}
	sort.Strings(order)
	items := []DiffItem{}
	for _, k := range order {
		x, hx := fb[k]
		y, hy := fa[k]
		sameValue := hx && hy && jsonKey(x.Value) == jsonKey(y.Value)
		// Settings and VLANs also report a change of source (same value, now inherited from another Profile).
		// Interfaces and rules are compared by value only: the layer they belong to does not change.
		if sameValue && (jsonKey(x.Source) == jsonKey(y.Source) || (x.Kind != "setting" && x.Kind != "vlan")) {
			continue
		}
		it := DiffItem{Key: k}
		switch {
		case hx && hy:
			it.Kind, it.Label, it.Before, it.After, it.BeforeSource, it.AfterSource, it.Basis = y.Kind, y.Label, x.Value, y.Value, x.Source, y.Source, y.Basis
		case hy:
			it.Kind, it.Label, it.After, it.AfterSource, it.Basis = y.Kind, y.Label, y.Value, y.Source, y.Basis
		default:
			it.Kind, it.Label, it.Before, it.BeforeSource, it.Basis = x.Kind, x.Label, x.Value, x.Source, x.Basis
		}
		items = append(items, it)
	}
	return items
}
