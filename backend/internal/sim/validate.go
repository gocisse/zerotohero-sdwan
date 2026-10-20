package sim

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Issue levels.
const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Issue is one validation finding. Path points into the topology, for example "profiles[branch].device".
type Issue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (i Issue) String() string { return i.Path + ": " + i.Message }

var (
	slugRe  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	ifaceRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

// DefaultMaxSegments is the documented default maximum segments per Enterprise.
const DefaultMaxSegments = 16 // rule: segment-limit

// SystemMaxSegments is the figure the guide gives for the raised maximum (128). Above it only a warning is given
// because the real limit is a system property that can be raised.
const SystemMaxSegments = 128 // rule: segment-limit

// MaxEdgeVLANs and MaxEdgeSegments are the documented per-Edge limits.
const (
	MaxEdgeVLANs    = 32 // rule: edge-vlan-limit
	MaxEdgeSegments = 16 // rule: edge-vlan-limit
)

// MaxSegments is the customer's maximum number of segments.
func (c Customer) MaxSegmentsOrDefault() int {
	if c.MaxSegments != nil {
		return *c.MaxSegments
	}
	return DefaultMaxSegments
}

// Errors returns only the error-level issues.
func Errors(issues []Issue) []Issue {
	var out []Issue
	for _, i := range issues {
		if i.Level == LevelError {
			out = append(out, i)
		}
	}
	return out
}

type validator struct {
	t      *Topology
	issues []Issue

	segs     map[string]Segment
	profiles map[string]*Profile
	links    map[string]*WanLink
	ruleIDs  map[string]string // rule id -> where defined
	linkUse  map[string]string // link id -> interface path
}

// Validate checks a decoded topology and returns EVERY problem found, in a stable order. It never panics on
// malformed input (nil maps, empty lists, unknown references).
func Validate(t *Topology) []Issue {
	v := &validator{t: t, segs: map[string]Segment{}, profiles: map[string]*Profile{}, links: map[string]*WanLink{},
		ruleIDs: map[string]string{}, linkUse: map[string]string{}}
	v.meta()
	v.customer()
	v.segments()
	v.gateways()
	v.wanLinks()
	v.profilesCheck()
	v.sites()
	v.unusedLinks()
	v.overlay()
	v.routing()
	return v.issues
}

func (v *validator) errf(path, format string, args ...any) {
	v.issues = append(v.issues, Issue{LevelError, path, fmt.Sprintf(format, args...)})
}

func (v *validator) warnf(path, format string, args ...any) {
	v.issues = append(v.issues, Issue{LevelWarning, path, fmt.Sprintf(format, args...)})
}

func (v *validator) meta() {
	t := v.t
	if t.Version != SchemaVersion {
		v.errf("version", "must be %d (this simulator reads topology schema %d)", SchemaVersion, SchemaVersion)
	}
	if !slugRe.MatchString(t.Meta.ID) {
		v.errf("meta.id", "must be a lowercase slug (letters, digits, single hyphens), got %q", t.Meta.ID)
	}
	if strings.TrimSpace(t.Meta.Title) == "" {
		v.errf("meta.title", "is required")
	}
	if strings.TrimSpace(t.Meta.Summary) == "" {
		v.errf("meta.summary", "is required")
	}
	if !t.Meta.Teaching {
		v.errf("meta.teaching", "must be true: every topology is a labelled teaching topology, never a real network")
	}
}

func (v *validator) customer() {
	c := v.t.Customer
	if !slugRe.MatchString(c.ID) {
		v.errf("customer.id", "must be a lowercase slug, got %q", c.ID)
	}
	if strings.TrimSpace(c.Name) == "" {
		v.errf("customer.name", "is required")
	}
	if c.MaxSegments != nil {
		switch {
		case *c.MaxSegments < 1:
			v.errf("customer.max_segments", "must be at least 1")
		case *c.MaxSegments > SystemMaxSegments:
			// rule: segment-limit
			v.warnf("customer.max_segments", "%d is above the 128 the guide gives for the raised maximum", *c.MaxSegments)
		}
	}
}

func (v *validator) segments() {
	// rule: global-segment
	v.segs[GlobalSegmentID] = Segment{ID: GlobalSegmentID, Name: "Global Segment", Type: SegRegular}
	seen := map[string]bool{}
	for i, s := range v.t.Segments {
		p := fmt.Sprintf("segments[%d]", i)
		if s.ID != "" {
			p = fmt.Sprintf("segments[%s]", s.ID)
		}
		if !slugRe.MatchString(s.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", s.ID)
			continue
		}
		if seen[s.ID] {
			v.errf(p, "duplicate segment id %q", s.ID)
			continue
		}
		seen[s.ID] = true
		if strings.TrimSpace(s.Name) == "" && s.ID != GlobalSegmentID {
			v.errf(p+".name", "is required")
		}
		if s.Type != SegRegular && s.Type != SegPrivate {
			v.errf(p+".type", "must be %q or %q, got %q", SegRegular, SegPrivate, s.Type)
		}
		if s.ServiceVLAN != nil && *s.ServiceVLAN < 1 {
			v.errf(p+".service_vlan", "must be a positive VLAN id")
		}
		if s.ID == GlobalSegmentID && s.Name == "" {
			s.Name = "Global Segment"
		}
		v.segs[s.ID] = s
	}
	// rule: segment-limit
	if n := len(v.segs); n > v.t.Customer.MaxSegmentsOrDefault() {
		v.errf("segments", "%d segments (including the Global Segment) exceed the customer's maximum of %d", n, v.t.Customer.MaxSegmentsOrDefault())
	}
}

func (v *validator) gateways() {
	seen := map[string]bool{}
	for i, g := range v.t.Gateways {
		p := fmt.Sprintf("gateways[%d]", i)
		if !slugRe.MatchString(g.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", g.ID)
			continue
		}
		p = fmt.Sprintf("gateways[%s]", g.ID)
		if seen[g.ID] {
			v.errf(p, "duplicate gateway id %q", g.ID)
		}
		seen[g.ID] = true
		if strings.TrimSpace(g.Region) == "" {
			v.errf(p+".region", "is required")
		}
	}
}

func (v *validator) wanLinks() {
	for i := range v.t.Links {
		l := &v.t.Links[i]
		p := fmt.Sprintf("links[%d]", i)
		if !slugRe.MatchString(l.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", l.ID)
			continue
		}
		p = fmt.Sprintf("links[%s]", l.ID)
		if _, dup := v.links[l.ID]; dup {
			v.errf(p, "duplicate link id %q", l.ID)
			continue
		}
		v.links[l.ID] = l
		if l.Type != LinkPublic && l.Type != LinkPrivate {
			v.errf(p+".type", "must be %q or %q, got %q", LinkPublic, LinkPrivate, l.Type)
		}
		if l.Medium != MediumWired && l.Medium != MediumWireless {
			v.errf(p+".medium", "must be %q or %q, got %q", MediumWired, MediumWireless, l.Medium)
		}
		if l.UpMbps <= 0 || l.DownMbps <= 0 {
			v.errf(p, "up_mbps and down_mbps must be positive")
		}
		if l.Type == LinkPrivate && l.Medium == MediumWireless {
			// rule: link-transport-groups
			v.warnf(p, "private wireless is not among the Links tab transport groups (Public Wired, Public Wireless, Private Wired)")
		}
	}
}

func (v *validator) profilesCheck() {
	maxSeg := v.t.Customer.MaxSegmentsOrDefault()
	for i := range v.t.Profiles {
		pr := &v.t.Profiles[i]
		p := fmt.Sprintf("profiles[%d]", i)
		if !slugRe.MatchString(pr.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", pr.ID)
			continue
		}
		p = fmt.Sprintf("profiles[%s]", pr.ID)
		if _, dup := v.profiles[pr.ID]; dup {
			v.errf(p, "duplicate profile id %q", pr.ID)
			continue
		}
		v.profiles[pr.ID] = pr
		if strings.TrimSpace(pr.Name) == "" {
			v.errf(p+".name", "is required")
		}
		if len(pr.Segments) == 0 {
			v.errf(p+".segments", "must list at least one segment (the Global Segment is always available)")
		}
		seen := map[string]bool{}
		for _, sid := range pr.Segments {
			if _, ok := v.segs[sid]; !ok {
				v.errf(p+".segments", "unknown segment %q", sid)
			}
			if seen[sid] {
				v.errf(p+".segments", "segment %q listed twice", sid)
			}
			seen[sid] = true
		}
		if len(pr.Segments) > maxSeg { // rule: segment-limit
			v.errf(p+".segments", "%d segments exceed the customer's maximum of %d", len(pr.Segments), maxSeg)
		}
		v.deviceSettings(p+".device", pr.Device)
		v.vlans(p+".vlans", pr.VLANs, nil)
		v.rules(p+".business_rules", pr.BusinessRules, "", pr.Segments, nil)
		v.fwRules(p+".firewall_rules", pr.FirewallRules)
	}
	// Reject and Skip are valid only with Stateful Firewall: check each Profile on its own setting.
	for i := range v.t.Profiles {
		pr := &v.t.Profiles[i]
		on := statefulOn(pr.Device, nil)
		for j, r := range pr.FirewallRules {
			if (r.Action == FwReject || r.Action == FwSkip) && !on { // rule: fw-reject-skip-stateful
				v.errf(fmt.Sprintf("profiles[%s].firewall_rules[%d]", pr.ID, j), "action %q needs %s turned on in the Profile (rule %s)", r.Action, StatefulSetting, r.ID)
			}
		}
	}
}

// statefulOn resolves the Stateful Firewall setting: Edge override, else Profile, else the documented default (off).
func statefulOn(profile, edge map[string]any) bool {
	for _, m := range []map[string]any{edge, profile} {
		if b, ok := m[StatefulSetting].(bool); ok { // rule: layer-merge
			return b
		}
	}
	return false // rule: fw-stateful-default
}

func (v *validator) deviceSettings(path string, m map[string]any) {
	ids := make([]string, 0, len(m))
	for k := range m {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for _, id := range ids {
		def, ok := SettingByID(id)
		if !ok {
			v.errf(path+"."+id, "unknown setting; known settings: %s", strings.Join(SettingIDs(), ", "))
			continue
		}
		if msg := def.checkValue(m[id]); msg != "" {
			v.errf(path+"."+id, "%s", msg)
		}
	}
}

// vlans validates a VLAN list. extra holds ids that already exist (the Profile's VLANs for an Edge's own list).
func (v *validator) vlans(path string, list []VLAN, extra map[int]bool) {
	seen := map[int]bool{}
	for i, vl := range list {
		p := fmt.Sprintf("%s[%d]", path, i)
		if vl.ID < 1 || vl.ID > 4094 {
			v.errf(p+".id", "VLAN id must be between 1 and 4094, got %d", vl.ID)
			continue
		}
		if strings.TrimSpace(vl.Name) == "" {
			v.errf(p+".name", "is required")
		}
		if seen[vl.ID] || extra[vl.ID] {
			v.errf(p, "duplicate VLAN id %d", vl.ID)
		}
		seen[vl.ID] = true
	}
}

// rules validates business rules. owner "" means a Profile rule; otherwise the Edge id (Edge-owned).
// allowedSegs are the segments the owner's Profile includes; edge is non-nil for Edge-owned rules.
func (v *validator) rules(path string, list []BusinessRule, owner string, allowedSegs []string, edge *Edge) {
	allowed := map[string]bool{}
	for _, s := range allowedSegs {
		allowed[s] = true
	}
	for i, r := range list {
		p := fmt.Sprintf("%s[%d]", path, i)
		if !slugRe.MatchString(r.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", r.ID)
			continue
		}
		p = fmt.Sprintf("%s[%s]", path, r.ID)
		if where, dup := v.ruleIDs[r.ID]; dup {
			v.errf(p, "duplicate rule id %q (also defined at %s)", r.ID, where)
		} else {
			v.ruleIDs[r.ID] = p
		}
		if _, ok := v.segs[r.Segment]; !ok {
			v.errf(p+".segment", "unknown segment %q", r.Segment) // rule: biz-segment-aware
		} else if !allowed[r.Segment] {
			v.errf(p+".segment", "segment %q is not part of the Profile's segments", r.Segment) // rule: biz-segment-aware
		}
		switch r.Match.Protocol {
		case "", "gre", "icmp", "tcp", "udp":
		default:
			v.errf(p+".match.protocol", "must be one of gre, icmp, tcp, udp, got %q", r.Match.Protocol)
		}
		a := r.Action
		switch a.Priority {
		case "", "high", "normal", "low":
		default:
			v.errf(p+".action.priority", "must be high, normal or low, got %q", a.Priority)
		}
		switch a.NetworkService {
		case "", NetDirect, NetMultipath, NetBackhaul:
		default:
			v.errf(p+".action.network_service", "must be direct, multipath or internet_backhaul, got %q", a.NetworkService)
		}
		if a.NetworkService == NetBackhaul && r.Match.Destination != DestInternet { // rule: backhaul-internet-only
			v.errf(p+".action.network_service", "internet_backhaul applies only when the destination is %q", DestInternet)
		}
		if a.ServiceClass != "" { // rule: service-class-custom
			switch a.ServiceClass {
			case "real_time", "transactional", "bulk":
			default:
				v.errf(p+".action.service_class", "must be real_time, transactional or bulk, got %q", a.ServiceClass)
			}
			if !r.Match.CustomApplication {
				v.errf(p+".action.service_class", "a Service Class can be set only for a custom application (match.custom_application: true)")
			}
		}
		v.steering(p+".action.steering", a.Steering, owner, edge)
	}
}

func (v *validator) steering(p string, s Steering, owner string, edge *Edge) {
	switch s.Mode {
	case "", SteerAuto:
		if s.Group != "" || s.Interface != "" || s.Link != "" {
			v.errf(p, "group, interface and link apply only to transport_group, interface and wan_link modes")
		}
		if s.Strictness != "" {
			v.errf(p+".strictness", "Mandatory, Preferred or Available is chosen only with transport_group, interface or wan_link steering")
		}
	case SteerGroup:
		switch s.Group {
		case "public_wired", "public_wireless", "private_wired": // rule: link-transport-groups
		default:
			v.errf(p+".group", "must be public_wired, public_wireless or private_wired, got %q", s.Group)
		}
	case SteerInterface, SteerWanLink:
		if owner == "" { // rule: steering-scope
			v.errf(p+".mode", "%s steering is allowed only on Edge-level rules, not on a Profile rule", s.Mode)
			return
		}
		if edge == nil {
			return
		}
		if s.Mode == SteerInterface {
			if !edgeHasInterface(edge, s.Interface) {
				v.errf(p+".interface", "Edge %s has no interface %q", edge.ID, s.Interface)
			}
		} else {
			if !edgeHasLink(edge, s.Link) {
				v.errf(p+".link", "Edge %s has no WAN interface using link %q", edge.ID, s.Link)
			}
		}
	default:
		v.errf(p+".mode", "must be auto, transport_group, interface or wan_link, got %q", s.Mode)
	}
	switch s.Strictness {
	case "", "mandatory", "preferred", "available":
	default:
		v.errf(p+".strictness", "must be mandatory, preferred or available, got %q", s.Strictness)
	}
}

func edgeHasInterface(e *Edge, id string) bool {
	for _, i := range e.Interfaces {
		if i.ID == id {
			return true
		}
	}
	return false
}

func edgeHasLink(e *Edge, id string) bool {
	for _, i := range e.Interfaces {
		if i.Kind == IfWan && i.Link == id {
			return true
		}
	}
	return false
}

func (v *validator) fwRules(path string, list []FirewallRule) {
	for i, r := range list {
		p := fmt.Sprintf("%s[%d]", path, i)
		if !slugRe.MatchString(r.ID) {
			v.errf(p+".id", "must be a lowercase slug, got %q", r.ID)
			continue
		}
		p = fmt.Sprintf("%s[%s]", path, r.ID)
		if where, dup := v.ruleIDs[r.ID]; dup {
			v.errf(p, "duplicate rule id %q (also defined at %s)", r.ID, where)
		} else {
			v.ruleIDs[r.ID] = p
		}
		switch r.Match.Protocol { // rule: fw-protocols
		case "", "gre", "icmp", "tcp", "udp":
		default:
			v.errf(p+".match.protocol", "must be one of gre, icmp, tcp, udp, got %q", r.Match.Protocol)
		}
		switch r.Action {
		case FwAllow, FwDrop, FwReject, FwSkip:
		default:
			v.errf(p+".action", "must be allow, drop, reject or skip, got %q", r.Action)
		}
	}
}

func (v *validator) sites() {
	siteSeen := map[string]bool{}
	edgeSeen := map[string]string{}
	regions := map[string]bool{}
	for _, g := range v.t.Gateways {
		regions[g.Region] = true
	}
	for si := range v.t.Sites {
		s := &v.t.Sites[si]
		sp := fmt.Sprintf("sites[%d]", si)
		if !slugRe.MatchString(s.ID) {
			v.errf(sp+".id", "must be a lowercase slug, got %q", s.ID)
		} else {
			sp = fmt.Sprintf("sites[%s]", s.ID)
			if siteSeen[s.ID] {
				v.errf(sp, "duplicate site id %q", s.ID)
			}
			siteSeen[s.ID] = true
		}
		if strings.TrimSpace(s.Region) == "" {
			v.errf(sp+".region", "is required")
		} else if len(v.t.Gateways) > 0 && !regions[s.Region] {
			v.warnf(sp+".region", "no Gateway is defined in region %q", s.Region)
		}
		if len(s.Edges) == 0 {
			v.errf(sp+".edges", "a site needs at least one Edge")
		}
		for ei := range s.Edges {
			v.edge(sp, ei, &s.Edges[ei], edgeSeen)
		}
	}
}

func (v *validator) edge(sitePath string, ei int, e *Edge, seen map[string]string) {
	p := fmt.Sprintf("%s.edges[%d]", sitePath, ei)
	if !slugRe.MatchString(e.ID) {
		v.errf(p+".id", "must be a lowercase slug, got %q", e.ID)
		return
	}
	p = fmt.Sprintf("edges[%s]", e.ID)
	if where, dup := seen[e.ID]; dup {
		v.errf(p, "duplicate edge id %q (also defined at %s)", e.ID, where)
	} else {
		seen[e.ID] = p
	}
	if strings.TrimSpace(e.Name) == "" {
		v.errf(p+".name", "is required")
	}
	if e.Role != RoleBranch && e.Role != RoleHub {
		v.errf(p+".role", "must be %q or %q, got %q", RoleBranch, RoleHub, e.Role)
	}
	prof, ok := v.profiles[e.Profile] // rule: one-profile-per-edge
	if !ok {
		v.errf(p+".profile", "unknown profile %q (an Edge has exactly one Profile)", e.Profile)
	}

	// VLANs the Edge has: inherited from the Profile plus its own (assumption 34).
	vlanIDs := map[int]bool{}
	profVLAN := map[int]bool{}
	if ok {
		for _, vl := range prof.VLANs {
			vlanIDs[vl.ID] = true
			profVLAN[vl.ID] = true
		}
	}
	v.vlans(p+".vlans", e.VLANs, profVLAN) // rule: assume-34
	for _, vl := range e.VLANs {
		vlanIDs[vl.ID] = true
	}
	if len(vlanIDs) > MaxEdgeVLANs { // rule: edge-vlan-limit
		v.errf(p+".vlans", "%d VLANs (inherited and own) exceed the documented limit of %d per Edge", len(vlanIDs), MaxEdgeVLANs)
	}

	var profSegs []string
	if ok {
		profSegs = prof.Segments
	}
	inProf := map[string]bool{}
	for _, s := range profSegs {
		inProf[s] = true
	}
	v.interfaces(p, e, vlanIDs, inProf, ok)

	// Overrides.
	o := e.Overrides
	v.deviceSettings(p+".overrides.device", o.Device) // rule: setting-overridable
	seenV := map[int]bool{}
	for i, vo := range o.VLANs {
		vp := fmt.Sprintf("%s.overrides.vlans[%d]", p, i)
		if !profVLAN[vo.ID] {
			v.errf(vp+".id", "VLAN %d is not a VLAN of Profile %q; only inherited Profile VLANs can be overridden", vo.ID, e.Profile)
		}
		if seenV[vo.ID] {
			v.errf(vp, "VLAN %d overridden twice", vo.ID)
		}
		seenV[vo.ID] = true
		for k := range vo.Settings {
			if k == "id" || k == "name" { // rule: vlan-name-id-locked
				v.errf(vp+".settings."+k, "the Profile VLAN %s cannot be overridden", k)
			}
		}
	}
	v.rules(p+".overrides.business_rules", o.BusinessRules, e.ID, profSegs, e)
	if len(o.Firewall.Rules) > 0 && !o.Firewall.Override { // rule: fw-override-tick
		v.errf(p+".overrides.firewall", "Edge firewall rules need override: true (the Edge Override checkbox)")
	}
	v.fwRules(p+".overrides.firewall.rules", o.Firewall.Rules)
	if ok {
		on := statefulOn(prof.Device, o.Device)
		check := func(path string, list []FirewallRule) {
			for j, r := range list {
				if (r.Action == FwReject || r.Action == FwSkip) && !on { // rule: fw-reject-skip-stateful
					v.errf(fmt.Sprintf("%s[%d]", path, j), "action %q needs %s turned on for Edge %s (rule %s)", r.Action, StatefulSetting, e.ID, r.ID)
				}
			}
		}
		check(p+".overrides.firewall.rules", o.Firewall.Rules)
	}
}

func (v *validator) interfaces(p string, e *Edge, vlanIDs map[int]bool, inProf map[string]bool, haveProfile bool) {
	seen := map[string]bool{}
	loops := map[string]bool{}
	wan := 0
	segsUsed := map[string]bool{}
	for i, in := range e.Interfaces {
		ip := fmt.Sprintf("%s.interfaces[%d]", p, i)
		if !ifaceRe.MatchString(in.ID) {
			v.errf(ip+".id", "must be lowercase letters, digits, hyphens or underscores, got %q", in.ID)
			continue
		}
		ip = fmt.Sprintf("%s.interfaces[%s]", p, in.ID)
		if seen[in.ID] {
			v.errf(ip, "duplicate interface id %q on Edge %s", in.ID, e.ID)
		}
		seen[in.ID] = true
		switch in.Kind {
		case IfWan:
			wan++
			if in.Link == "" {
				v.errf(ip+".link", "a WAN interface needs a link")
			} else if _, ok := v.links[in.Link]; !ok {
				v.errf(ip+".link", "unknown link %q", in.Link)
			} else if where, used := v.linkUse[in.Link]; used {
				v.errf(ip+".link", "link %q is already used by %s (a link belongs to one WAN interface)", in.Link, where)
			} else {
				v.linkUse[in.Link] = ip
			}
		case IfLan, IfRouted, IfLoopback:
			if in.Link != "" {
				v.errf(ip+".link", "only WAN interfaces carry a link (kind is %q)", in.Kind)
			}
		default:
			v.errf(ip+".kind", "must be wan, lan, routed or loopback, got %q", in.Kind)
		}
		if in.Kind == IfLoopback { // rule: loopback-edge-only
			if loops[in.ID] { // rule: loopback-unique
				v.errf(ip, "loopback id %q repeated", in.ID)
			}
			loops[in.ID] = true
		}
		// rule: assume-40
		if in.Kind == IfLan {
			switch {
			case in.VLAN == nil:
				v.errf(ip+".vlan", "a LAN interface needs a VLAN the Edge has (assumption 40)")
			case !vlanIDs[*in.VLAN]:
				v.errf(ip+".vlan", "VLAN %d is neither inherited from Profile %q nor defined on Edge %s", *in.VLAN, e.Profile, e.ID)
			}
		} else if in.VLAN != nil {
			v.errf(ip+".vlan", "only LAN interfaces carry a VLAN (kind is %q; assumption 40)", in.Kind)
		}
		// rule: interface-all-segments-default
		sseen := map[string]bool{}
		for _, sid := range in.Segments {
			if _, ok := v.segs[sid]; !ok {
				v.errf(ip+".segments", "unknown segment %q", sid)
				continue
			}
			if sseen[sid] {
				v.errf(ip+".segments", "segment %q listed twice", sid)
			}
			sseen[sid] = true
			segsUsed[sid] = true
			if haveProfile && !inProf[sid] { // rule: assume-12
				v.errf(ip+".segments", "segment %q is not part of Profile %q (assumption 12)", sid, e.Profile)
			}
		}
	}
	if wan == 0 {
		v.errf(p+".interfaces", "an Edge needs at least one WAN interface")
	}
	if len(segsUsed) > MaxEdgeSegments { // rule: edge-vlan-limit
		v.errf(p+".interfaces", "%d segments in use exceed the documented limit of %d per Edge", len(segsUsed), MaxEdgeSegments)
	}
}

func (v *validator) unusedLinks() {
	ids := make([]string, 0, len(v.links))
	for id := range v.links {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, used := v.linkUse[id]; !used {
			v.warnf("links["+id+"]", "link is not used by any WAN interface")
		}
	}
}
