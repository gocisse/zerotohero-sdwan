package sim

import (
	"errors"
	"fmt"
)

// Source kinds: where a value comes from.
const (
	SrcProfile      = "profile"
	SrcEdgeOverride = "edge-override"
	SrcEdgeOnly     = "edge-only"
	SrcCustomer     = "customer"
	SrcDefault      = "default"
	SrcUnset        = "unset"
)

// Source says which layer a value comes from and which Profile or Edge it is.
type Source struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// NotModelled is an explicit "the documentation does not say" marker returned instead of a guess.
type NotModelled struct {
	Rule   string `json:"rule"`
	Item   string `json:"item"`
	Reason string `json:"reason"`
	Basis  Basis  `json:"basis"`
}

func notModelled(ruleID, item string) NotModelled {
	r := RuleByID(ruleID)
	return NotModelled{Rule: ruleID, Item: item, Reason: r.Text, Basis: r.Basis()}
}

// EffectiveSetting is one device setting after the layer merge.
type EffectiveSetting struct {
	ID           string       `json:"id"`
	Label        string       `json:"label"`
	Section      string       `json:"section"`
	Value        any          `json:"value"`
	Source       Source       `json:"source"`
	ProfileValue any          `json:"profile_value,omitempty"`
	Overridable  bool         `json:"overridable"`
	Why          string       `json:"why"`
	Basis        Basis        `json:"basis"`
	Also         []Basis      `json:"also,omitempty"`
	NotModelled  *NotModelled `json:"not_modelled,omitempty"`
}

// EffectiveVLAN is a VLAN the Edge has.
type EffectiveVLAN struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Source   Source         `json:"source"`
	Settings map[string]any `json:"settings,omitempty"`
	Locked   bool           `json:"name_id_locked"`
	Basis    Basis          `json:"basis"`
	Also     []Basis        `json:"also,omitempty"`
}

// EffectiveInterface is an Edge interface with its effective segments.
type EffectiveInterface struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	VLAN           *int     `json:"vlan,omitempty"`
	Segments       []string `json:"segments"`
	SegmentsSource string   `json:"segments_source"` // "interface" or "profile-default"
	Link           string   `json:"link,omitempty"`
	Address        string   `json:"address,omitempty"`
	Override       bool     `json:"override"`
	Source         Source   `json:"source"`
	Basis          Basis    `json:"basis"`
	Also           []Basis  `json:"also,omitempty"`
}

// EffectiveSegment is a segment the Edge's Profile includes.
type EffectiveSegment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Basis Basis  `json:"basis"`
}

// EffectiveRule is a business rule as the Edge sees it.
type EffectiveRule struct {
	Rule         BusinessRule `json:"rule"`
	Owner        Source       `json:"owner"`
	SupersededBy string       `json:"superseded_by,omitempty"`
	Supersedes   []string     `json:"supersedes,omitempty"`
	// Rewritten is set when the Private-segment rule turned Internet Multipath into Direct.
	SteeringMode     string  `json:"effective_steering_mode"`
	Rewritten        bool    `json:"rewritten,omitempty"`
	EffectiveService string  `json:"effective_network_service,omitempty"`
	Basis            Basis   `json:"basis"`
	Also             []Basis `json:"also,omitempty"`
}

// SegmentPolicy is the Edge Business Policy tab for one segment: the two sections the real tab has.
type SegmentPolicy struct {
	Segment       EffectiveSegment `json:"segment"`
	EdgeOverrides []EffectiveRule  `json:"edge_overrides"`
	FromProfile   []EffectiveRule  `json:"from_profile"`
	Precedence    NotModelled      `json:"precedence"`
	Basis         Basis            `json:"basis"`
}

// EffectiveFirewallRule is a firewall rule as the Edge sees it.
type EffectiveFirewallRule struct {
	Rule  FirewallRule `json:"rule"`
	Owner Source       `json:"owner"`
	// Group is "edge-override" or "profile"; the Edge group has priority over the Profile group.
	Group string  `json:"group"`
	Basis Basis   `json:"basis"`
	Also  []Basis `json:"also,omitempty"`
}

// EffectiveFirewall is the Edge Firewall tab.
type EffectiveFirewall struct {
	Override bool                    `json:"override"`
	Stateful EffectiveSetting        `json:"stateful_firewall"`
	Rules    []EffectiveFirewallRule `json:"rules"`
	Order    NotModelled             `json:"evaluation_order"`
	Priority Basis                   `json:"group_priority"`
}

// CustomerSetting is an Enterprise-level value.
type CustomerSetting struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Value  any    `json:"value"`
	Source Source `json:"source"`
	Basis  Basis  `json:"basis"`
}

// EdgeRef identifies the Edge an Effective belongs to.
type EdgeRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Site        string `json:"site"`
	Role        string `json:"role"`
	Profile     string `json:"profile"`
	ProfileName string `json:"profile_name"`
}

// Effective is everything the engine knows about one Edge: values with sources and bases, plus the
// explicit not-modelled markers.
type Effective struct {
	Topology       string               `json:"topology"`
	Edge           EdgeRef              `json:"edge"`
	Customer       []CustomerSetting    `json:"customer"`
	Segments       []EffectiveSegment   `json:"segments"`
	Device         []EffectiveSetting   `json:"device"`
	VLANs          []EffectiveVLAN      `json:"vlans"`
	Interfaces     []EffectiveInterface `json:"interfaces"`
	BusinessPolicy []SegmentPolicy      `json:"business_policy"`
	Firewall       EffectiveFirewall    `json:"firewall"`
	NotModelled    []NotModelled        `json:"not_modelled"`
	// Reverted lists interfaces that reverted to the Profile after a Profile switch (empty outside what-if).
	Reverted []string `json:"reverted_interfaces,omitempty"`
}

// ErrNotFound is returned for an unknown edge id.
var ErrNotFound = errors.New("not found")

// FindEdge returns the Edge, its site id and its Profile.
func (t *Topology) FindEdge(id string) (*Edge, string, *Profile, error) {
	for si := range t.Sites {
		for ei := range t.Sites[si].Edges {
			e := &t.Sites[si].Edges[ei]
			if e.ID != id {
				continue
			}
			for pi := range t.Profiles {
				if t.Profiles[pi].ID == e.Profile {
					return e, t.Sites[si].ID, &t.Profiles[pi], nil
				}
			}
			return nil, "", nil, fmt.Errorf("edge %s references unknown profile %q", id, e.Profile)
		}
	}
	return nil, "", nil, fmt.Errorf("edge %q: %w", id, ErrNotFound)
}

func (t *Topology) segment(id string) Segment {
	for _, s := range t.Segments {
		if s.ID == id {
			if s.Name == "" && id == GlobalSegmentID {
				s.Name = "Global Segment"
			}
			return s
		}
	}
	if id == GlobalSegmentID { // rule: global-segment
		return Segment{ID: id, Name: "Global Segment", Type: SegRegular}
	}
	return Segment{ID: id, Name: id, Type: SegRegular}
}

// EffectiveConfig computes the effective configuration of one Edge. It is a pure function of the topology.
func EffectiveConfig(t *Topology, edgeID string) (*Effective, error) {
	e, siteID, prof, err := t.FindEdge(edgeID)
	if err != nil {
		return nil, err
	}
	ec := &Effective{
		Topology: t.Meta.ID,
		Edge:     EdgeRef{ID: e.ID, Name: e.Name, Site: siteID, Role: e.Role, Profile: prof.ID, ProfileName: prof.Name},
	}
	ec.Customer = customerSettings(t)
	ec.Device = effectiveDevice(t, e, prof)
	ec.Segments = effectiveSegments(t, prof)
	ec.VLANs = effectiveVLANs(e, prof)
	ec.Interfaces = effectiveInterfaces(t, e, prof)
	ec.BusinessPolicy = effectiveBusinessPolicy(t, e, prof)
	ec.Firewall = effectiveFirewall(t, e, prof, ec.Device)
	ec.NotModelled = []NotModelled{
		notModelled(RNMBizOrder, "Business policy evaluation order"),                             // rule: nm-biz-order
		notModelled(RNMFwOrder, "Firewall evaluation order and firewall versus business policy"), // rule: nm-fw-vs-bp-order
		notModelled(RNMSegmentToSegment, "Traffic between segments"),                             // rule: nm-segment-to-segment
		notModelled(RNMFwSegment, "Firewall rules per segment"),                                  // rule: nm-fw-segment-scope
		notModelled(RNMOverrideMarking, "How the real VCO marks an override"),                    // rule: nm-override-marking
		notModelled(RNMOverrideList, "Which other settings can be overridden"),                   // rule: nm-override-list
	}
	return ec, nil
}

func customerSettings(t *Topology) []CustomerSetting {
	c := t.Customer
	src := Source{Kind: SrcCustomer, ID: c.ID, Name: c.Name}
	def := Source{Kind: SrcDefault}
	out := []CustomerSetting{}
	// rule: segment-limit
	if c.MaxSegments != nil {
		out = append(out, CustomerSetting{"max_segments", "Maximum segments", *c.MaxSegments, src, basisOf(RSegmentLimit)})
	} else {
		out = append(out, CustomerSetting{"max_segments", "Maximum segments", DefaultMaxSegments, def, basisOf(RSegmentLimit)})
	}
	// rule: premium-default
	if c.PremiumService != nil {
		out = append(out, CustomerSetting{"premium_service", "Premium Service", *c.PremiumService, src, basisOf(RPremiumDefault)})
	} else {
		out = append(out, CustomerSetting{"premium_service", "Premium Service", true, def, basisOf(RPremiumDefault)})
	}
	return out
}

func profileSource(p *Profile) Source { return Source{Kind: SrcProfile, ID: p.ID, Name: p.Name} }
func edgeSource(e *Edge, kind string) Source {
	return Source{Kind: kind, ID: e.ID, Name: e.Name}
}

// effectiveDevice merges Profile and Edge device settings: Edge override wins, else the Profile value, else the
// documented default, else unset.
func effectiveDevice(t *Topology, e *Edge, p *Profile) []EffectiveSetting {
	out := make([]EffectiveSetting, 0, len(Settings))
	for _, def := range Settings {
		s := EffectiveSetting{ID: def.ID, Label: def.Label, Section: def.Section, Overridable: true}
		s.Also = append(s.Also, basisOf(RSettingOverridable)) // rule: setting-overridable
		if def.Kind == KindScalar {
			s.Also = append(s.Also, basisOf(RA33OpaqueValues)) // rule: assume-33
		}
		if !def.SectionKnown {
			s.Also = append(s.Also, basisOf(RA36SectionUnknown)) // rule: assume-36
		}
		ov, hasOv := e.Overrides.Device[def.ID]
		pv, hasPv := p.Device[def.ID]
		switch {
		case hasOv: // rule: layer-merge
			s.Value, s.Source, s.Basis = ov, edgeSource(e, SrcEdgeOverride), basisOf(RLayerMerge)
			if hasPv {
				s.ProfileValue = pv
				s.Why = fmt.Sprintf("Override is ticked on %s: its value %v replaces the %s value %v.", e.Name, ov, p.Name, pv)
			} else {
				s.Why = fmt.Sprintf("Override is ticked on %s with value %v; Profile %s sets nothing for it.", e.Name, ov, p.Name)
			}
		case hasPv: // rule: layer-merge
			s.Value, s.Source, s.Basis = pv, profileSource(p), basisOf(RLayerMerge)
			s.Why = fmt.Sprintf("%s does not override it, so it inherits the value from Profile %s.", e.Name, p.Name)
		case def.HasDefault: // rule: fw-stateful-default
			s.Value, s.Source, s.Basis = def.Default, Source{Kind: SrcDefault}, basisOf(def.DefaultRule)
			s.Why = "Neither the Edge nor its Profile sets it, so the documented default applies."
		default: // rule: nm-setting-defaults
			nm := notModelled(RNMDefaults, def.Label)
			s.Value, s.Source, s.Basis, s.NotModelled = nil, Source{Kind: SrcUnset}, nm.Basis, &nm
			s.Why = "Neither the Edge nor its Profile sets it; the documentation gives no default value."
		}
		out = append(out, s)
	}
	return out
}

func effectiveSegments(t *Topology, p *Profile) []EffectiveSegment {
	out := make([]EffectiveSegment, 0, len(p.Segments))
	for _, id := range p.Segments {
		s := t.segment(id)
		b := basisOf(RGlobalSegment) // rule: global-segment
		if id != GlobalSegmentID {
			b = basisOf(RProfileSegments) // rule: profile-segments
		}
		out = append(out, EffectiveSegment{ID: s.ID, Name: s.Name, Type: s.Type, Basis: b})
	}
	return out
}

func effectiveVLANs(e *Edge, p *Profile) []EffectiveVLAN {
	out := []EffectiveVLAN{}
	ovBy := map[int]VLANOverride{}
	for _, vo := range e.Overrides.VLANs {
		ovBy[vo.ID] = vo
	}
	for _, v := range p.VLANs {
		ev := EffectiveVLAN{ID: v.ID, Name: v.Name, Source: profileSource(p), Locked: true, Basis: basisOf(RProfileInterface)}
		ev.Also = append(ev.Also, basisOf(RVlanLocked))       // rule: vlan-name-id-locked
		if vo, ok := ovBy[v.ID]; ok && len(vo.Settings) > 0 { // rule: layer-merge
			ev.Settings = vo.Settings
			ev.Source = edgeSource(e, SrcEdgeOverride)
			ev.Basis = basisOf(RLayerMerge)
			ev.Also = append(ev.Also, basisOf(RA33OpaqueValues))
		}
		out = append(out, ev)
	}
	for _, v := range e.VLANs { // rule: assume-34
		out = append(out, EffectiveVLAN{ID: v.ID, Name: v.Name, Source: edgeSource(e, SrcEdgeOnly), Basis: basisOf(RA34EdgeVlans)})
	}
	return out
}

func effectiveInterfaces(t *Topology, e *Edge, p *Profile) []EffectiveInterface {
	out := make([]EffectiveInterface, 0, len(e.Interfaces))
	for _, in := range e.Interfaces { // rule: profile-interface-inherit
		ei := EffectiveInterface{ID: in.ID, Kind: in.Kind, VLAN: in.VLAN, Link: in.Link, Address: in.Address, Override: in.Override,
			Source: edgeSource(e, SrcEdgeOnly), Basis: basisOf(RProfileInterface)}
		if len(in.Segments) > 0 {
			ei.Segments, ei.SegmentsSource = in.Segments, "interface"
		} else { // rule: interface-all-segments-default
			ei.Segments, ei.SegmentsSource = append([]string{}, p.Segments...), "profile-default"
			ei.Also = append(ei.Also, basisOf(RInterfaceAllSegs))
		}
		switch {
		case in.Kind == IfLoopback: // rule: loopback-edge-only
			ei.Basis = basisOf(RLoopbackEdgeOnly)
		case in.Override: // rule: layer-merge
			ei.Source, ei.Basis = edgeSource(e, SrcEdgeOverride), basisOf(RLayerMerge)
		default:
			ei.Source = profileSource(p)
		}
		if in.Kind == IfLan { // rule: assume-40
			ei.Also = append(ei.Also, basisOf(RA40InterfaceVlans))
		}
		out = append(out, ei)
	}
	return out
}

func effectiveBusinessPolicy(t *Topology, e *Edge, p *Profile) []SegmentPolicy {
	segIDs := append([]string{}, p.Segments...)
	have := map[string]bool{}
	for _, s := range segIDs {
		have[s] = true
	}
	for _, r := range e.Overrides.BusinessRules { // rule: assume-38
		if !have[r.Segment] {
			have[r.Segment] = true
			segIDs = append(segIDs, r.Segment)
		}
	}
	var out []SegmentPolicy
	for _, sid := range segIDs {
		seg := t.segment(sid)
		es := EffectiveSegment{ID: seg.ID, Name: seg.Name, Type: seg.Type, Basis: basisOf(RBizSegmentAware)}
		sp := SegmentPolicy{Segment: es, EdgeOverrides: []EffectiveRule{}, FromProfile: []EffectiveRule{},
			Precedence: notModelled(RNMBizOrder, "Rule evaluation order"), Basis: basisOf(RBizSections)} // rule: biz-sections
		// Profile rules first, so Edge rules can mark them superseded.
		for _, r := range p.BusinessRules {
			if r.Segment != sid {
				continue
			}
			sp.FromProfile = append(sp.FromProfile, makeRule(r, profileSource(p), seg, RBizSegmentAware))
		}
		for _, r := range e.Overrides.BusinessRules { // rule: biz-supersede
			if r.Segment != sid {
				continue
			}
			er := makeRule(r, edgeSource(e, SrcEdgeOverride), seg, RBizSupersede)
			for i := range sp.FromProfile {
				if sp.FromProfile[i].Rule.Match == r.Match { // same match: the Edge rule supersedes the Profile rule
					sp.FromProfile[i].SupersededBy = r.ID
					sp.FromProfile[i].Also = append(sp.FromProfile[i].Also, basisOf(RBizSupersede), basisOf(RA41SameMatch)) // rule: assume-41
					er.Supersedes = append(er.Supersedes, sp.FromProfile[i].Rule.ID)
					er.Also = append(er.Also, basisOf(RA41SameMatch))
				}
			}
			if !hasString(p.Segments, sid) { // rule: assume-38
				er.Also = append(er.Also, basisOf(RA38SwitchEdgeRules))
			}
			sp.EdgeOverrides = append(sp.EdgeOverrides, er)
		}
		out = append(out, sp)
	}
	return out
}

func hasString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func makeRule(r BusinessRule, owner Source, seg Segment, basis string) EffectiveRule {
	er := EffectiveRule{Rule: r, Owner: owner, Basis: basisOf(basis)}
	er.Also = append(er.Also, basisOf(RA37AuthoredOrder)) // rule: assume-37
	er.EffectiveService = r.Action.NetworkService
	er.SteeringMode = r.Action.Steering.Mode
	if er.SteeringMode == "" { // rule: steering-auto-default
		er.SteeringMode = SteerAuto
	}
	if seg.Type == SegPrivate && r.Action.NetworkService == NetMultipath { // rule: private-segment-rewrite
		er.Rewritten, er.EffectiveService = true, NetDirect
		er.Also = append(er.Also, basisOf(RPrivateRewrite))
	}
	return er
}

func effectiveFirewall(t *Topology, e *Edge, p *Profile, device []EffectiveSetting) EffectiveFirewall {
	fw := EffectiveFirewall{Override: e.Overrides.Firewall.Override, Rules: []EffectiveFirewallRule{},
		Order: notModelled(RNMFwOrder, "Firewall rule evaluation order"), Priority: basisOf(RFwEdgePriority)}
	for _, d := range device {
		if d.ID == StatefulSetting {
			fw.Stateful = d
			fw.Stateful.Also = append(fw.Stateful.Also, basisOf(RA39StatefulSetting)) // rule: assume-39
		}
	}
	if fw.Override { // rule: fw-edge-priority
		for _, r := range e.Overrides.Firewall.Rules {
			fw.Rules = append(fw.Rules, EffectiveFirewallRule{Rule: r, Owner: edgeSource(e, SrcEdgeOverride), Group: "edge-override",
				Basis: basisOf(RFwEdgePriority), Also: []Basis{basisOf(RA37AuthoredOrder)}})
		}
	}
	for _, r := range p.FirewallRules { // rule: fw-override-tick
		fw.Rules = append(fw.Rules, EffectiveFirewallRule{Rule: r, Owner: profileSource(p), Group: "profile",
			Basis: basisOf(RFwOverrideTick), Also: []Basis{basisOf(RA37AuthoredOrder)}})
	}
	return fw
}
