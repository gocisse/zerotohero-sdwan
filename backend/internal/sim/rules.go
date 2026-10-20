package sim

import "fmt"

// Status says how well a behaviour is backed.
type Status string

const (
	// StatusVerified: a VERIFIED or OBSERVED row of a fact sheet in content/reference/ states it.
	StatusVerified Status = "verified"
	// StatusAssumption: not documented; a numbered assumption in docs/SIMULATOR-MODEL.md. Always shown as such.
	StatusAssumption Status = "assumption"
	// StatusNotModelled: documented as unknown (a VERIFY row) or out of scope; the engine returns a marker, never a guess.
	StatusNotModelled Status = "not-modelled"
)

// Rule is one behaviour the engine implements (or deliberately refuses to implement). Code that implements a
// rule carries a "// rule: <id>" comment; rules_test.go checks that every comment names a registered rule, every
// registered rule is cited, every verified rule's Claim is a row of Sheet with status VERIFIED or OBSERVED, every
// not-modelled rule's Claim is a VERIFY row, and every assumption number exists in docs/SIMULATOR-MODEL.md.
type Rule struct {
	ID         string
	Status     Status
	Sheet      string // fact sheet file name without .md (verified and not-modelled rules)
	Claim      string // exact substring of the sheet row
	Assumption int    // assumption number (assumption rules)
	Text       string // what the engine does, in one line
}

// Basis is the evidence attached to every value the engine returns.
type Basis struct {
	Status Status `json:"status"`
	Rule   string `json:"rule"`
	// Ref is the fact sheet name ("edge-overrides") or "assumption 5".
	Ref  string `json:"ref"`
	Text string `json:"text"`
}

// Basis returns the evidence for the rule.
func (r Rule) Basis() Basis {
	ref := r.Sheet
	if r.Status == StatusAssumption {
		ref = fmt.Sprintf("assumption %d", r.Assumption)
	}
	return Basis{Status: r.Status, Rule: r.ID, Ref: ref, Text: r.Text}
}

// Rule ids.
const (
	RLayerMerge         = "layer-merge"
	RSettingOverridable = "setting-overridable"
	RVlanLocked         = "vlan-name-id-locked"
	RLoopbackEdgeOnly   = "loopback-edge-only"
	RLoopbackUnique     = "loopback-unique"
	RProfileInterface   = "profile-interface-inherit"
	RInterfaceAllSegs   = "interface-all-segments-default"
	RBizSections        = "biz-sections"
	RBizSupersede       = "biz-supersede"
	RBizSegmentAware    = "biz-segment-aware"
	RFwEdgePriority     = "fw-edge-priority"
	RFwOverrideTick     = "fw-override-tick"
	RFwStatefulDefault  = "fw-stateful-default"
	RFwActionsStateful  = "fw-reject-skip-stateful"
	RFwProtocols        = "fw-protocols"
	RPrivateRewrite     = "private-segment-rewrite"
	RGlobalSegment      = "global-segment"
	RSegmentLimit       = "segment-limit"
	RVlanLimit          = "edge-vlan-limit"
	RSwitchVlans        = "profile-switch-vlans"
	RSteeringScope      = "steering-scope"
	RBackhaulInternet   = "backhaul-internet-only"
	RServiceClassCustom = "service-class-custom"
	RSteeringAuto       = "steering-auto-default"
	ROneProfile         = "one-profile-per-edge"
	RPremiumDefault     = "premium-default"
	RLinkGroups         = "link-transport-groups"
	RProfileSegments    = "profile-segments"
	RNMBizOrder         = "nm-biz-order"
	RNMFwOrder          = "nm-fw-vs-bp-order"
	RNMSegmentToSegment = "nm-segment-to-segment"
	RNMOverrideMarking  = "nm-override-marking"
	RNMOverrideList     = "nm-override-list"
	RNMSwitchBGP        = "nm-profile-switch-bgp"
	RNMSwitchOverrides  = "nm-profile-switch-overrides"
	RNMFwSegment        = "nm-fw-segment-scope"
	RNMDefaults         = "nm-setting-defaults"
	RA5ProfileEditSkips = "assume-5"
	RA8RevertToProfile  = "assume-8"
	RA12InterfaceSegs   = "assume-12"
	RA33OpaqueValues    = "assume-33"
	RA34EdgeVlans       = "assume-34"
	RA35SwitchVlanByID  = "assume-35"
	RA36SectionUnknown  = "assume-36"
	RA37AuthoredOrder   = "assume-37"
	RA38SwitchEdgeRules = "assume-38"
	RA39StatefulSetting = "assume-39"
	RA40InterfaceVlans  = "assume-40"
	RA41SameMatch       = "assume-41"
)

// Rules is the registry. Add a rule here before citing it in code.
var baseRules = []Rule{
	{ID: RLayerMerge, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Some Device-tab settings are inherited", Text: "A setting is inherited from the Profile; ticking Override lets the Edge edit it."},
	{ID: RSettingOverridable, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Examples that offer Override", Text: "This setting is one of those documented as offering Override."},
	{ID: RVlanLocked, Status: StatusVerified, Sheet: "edge-overrides", Claim: "The system prevents overriding the Profile VLAN", Text: "A Profile VLAN's name and ID cannot be overridden."},
	{ID: RLoopbackEdgeOnly, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Loopback interfaces apply exclusively", Text: "Loopback interfaces exist only on Edges."},
	{ID: RLoopbackUnique, Status: StatusVerified, Sheet: "segments", Claim: "Loopback interface IDs must be unique", Text: "Loopback ids are unique across segments within an Edge."},
	{ID: RProfileInterface, Status: StatusVerified, Sheet: "profiles", Claim: "Interface settings of a Profile are applied", Text: "Profile interface settings reach the Edges and can be overridden for one Edge."},
	{ID: RInterfaceAllSegs, Status: StatusVerified, Sheet: "segments", Claim: "On an Edge, interface settings by default", Text: "An interface applies to all segments unless Segments is set."},
	{ID: RBizSections, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Edge page for Business Policy has a section", Text: "The Edge Business Policy tab has Rules From Profile and Edge Overrides."},
	{ID: RBizSupersede, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Business policy rules created at the Profile level", Text: "A Profile rule cannot be edited on the Edge; an identical Edge rule with new parameters supersedes it."},
	{ID: RBizSegmentAware, Status: StatusVerified, Sheet: "business-policy", Claim: "Business policy rules are segment aware", Text: "Business rules belong to a segment."},
	{ID: RFwEdgePriority, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Firewall: rules created at the Profile level", Text: "Edge override firewall rules take priority over inherited Profile rules."},
	{ID: RFwOverrideTick, Status: StatusVerified, Sheet: "firewall", Claim: "Edges inherit firewall rules and settings", Text: "Edges inherit Profile firewall rules; the Edge Override checkbox lets an Edge change them."},
	{ID: RFwStatefulDefault, Status: StatusVerified, Sheet: "firewall", Claim: "By default Orchestrator deactivates Stateful", Text: "Stateful Firewall is off by default."},
	{ID: RFwActionsStateful, Status: StatusVerified, Sheet: "firewall", Claim: "Actions: Allow, Drop", Text: "Reject and Skip need Stateful Firewall."},
	{ID: RFwProtocols, Status: StatusVerified, Sheet: "firewall", Claim: "Protocols supported in firewall rules", Text: "Firewall protocols are GRE, ICMP, TCP and UDP."},
	{ID: RPrivateRewrite, Status: StatusVerified, Sheet: "segments", Claim: "For Private segments the Orchestrator", Text: "On a Private segment every Internet Multipath policy becomes Direct."},
	{ID: RGlobalSegment, Status: StatusVerified, Sheet: "segments", Claim: "There is a Global Segment", Text: "The Global Segment always exists; its type is Regular or Private."},
	{ID: RSegmentLimit, Status: StatusVerified, Sheet: "segments", Claim: "Default maximum segments for an Enterprise", Text: "The default maximum is 16 segments per Enterprise."},
	{ID: RVlanLimit, Status: StatusVerified, Sheet: "segments", Claim: "An Edge can have up to 32 VLANs", Text: "An Edge has up to 32 VLANs across 16 segments."},
	{ID: RSwitchVlans, Status: StatusVerified, Sheet: "edge-overrides", Claim: "On a Profile change, the target profile removes", Text: "A Profile change removes VLANs inherited from the old Profile; an interface tied to a removed VLAN reverts to the new Profile."},
	{ID: RSteeringScope, Status: StatusVerified, Sheet: "edge-overrides", Claim: "Interface and WAN Link link steering is allowed only", Text: "Interface and WAN Link steering exist only at Edge level; Transport Group at both."},
	{ID: RBackhaulInternet, Status: StatusVerified, Sheet: "business-policy", Claim: "Internet Backhaul applies only", Text: "Internet Backhaul applies only to Internet destinations."},
	{ID: RServiceClassCustom, Status: StatusVerified, Sheet: "applications-qos", Claim: "Service Class cannot be changed", Text: "A rule's Service Class applies to custom applications only."},
	{ID: RSteeringAuto, Status: StatusVerified, Sheet: "dmpo-steering", Claim: "Auto (default for applications)", Text: "Link steering defaults to Auto."},
	{ID: ROneProfile, Status: StatusVerified, Sheet: "profiles", Claim: "When an Edge is provisioned", Text: "An Edge has one Profile, chosen at provisioning."},
	{ID: RPremiumDefault, Status: StatusVerified, Sheet: "dmpo", Claim: "Premium Service is on by default", Text: "Premium Service is on by default for a customer."},
	{ID: RLinkGroups, Status: StatusVerified, Sheet: "dmpo-vco-ui", Claim: "Transport Groups view (Public Wired", Text: "The Links tab groups: Public Wired, Public Wireless, Private Wired."},
	{ID: RProfileSegments, Status: StatusVerified, Sheet: "segments", Claim: "Segments are chosen for a Profile", Text: "A Profile includes segments chosen with Change Profile Segments."},

	{ID: RNMBizOrder, Status: StatusNotModelled, Sheet: "business-policy", Claim: "Whether evaluation is first-match", Text: "not modelled: whether business rules are first-match, how ties resolve and how Profile and Edge rules interleave beyond supersession are not stated."},
	{ID: RNMFwOrder, Status: StatusNotModelled, Sheet: "firewall", Claim: "Rule evaluation order, the default action", Text: "not modelled: firewall rule evaluation order, the no-match action and firewall versus business policy order are not stated."},
	{ID: RNMSegmentToSegment, Status: StatusNotModelled, Sheet: "segments", Claim: "Whether traffic between two segments", Text: "not modelled: whether traffic between segments is blocked or allowed is not stated."},
	{ID: RNMOverrideMarking, Status: StatusNotModelled, Sheet: "edge-overrides", Claim: "How the UI visually marks an overridden", Text: "not modelled: how the real VCO marks an overridden setting. The badge here is this simulator's own."},
	{ID: RNMOverrideList, Status: StatusNotModelled, Sheet: "edge-overrides", Claim: "A complete list of which settings", Text: "not modelled: the complete list of overridable settings is not documented; only settings documented as overridable are included."},
	{ID: RNMSwitchBGP, Status: StatusNotModelled, Sheet: "profiles", Claim: "Changing an Edge's Profile: only the BGP", Text: "not modelled: removal of Profile-level BGP filters on a Profile switch (routing is step 12)."},
	{ID: RNMSwitchOverrides, Status: StatusNotModelled, Sheet: "profiles", Claim: "Changing an Edge's Profile: only the BGP", Text: "not modelled: the general effect of a Profile switch on other overrides is not stated."},
	{ID: RNMFwSegment, Status: StatusNotModelled, Sheet: "firewall", Claim: "Whether general firewall rules are segment scoped", Text: "not modelled: whether general firewall rules are scoped per segment."},
	{ID: RNMDefaults, Status: StatusNotModelled, Sheet: "edge-overrides", Claim: "A complete list of which settings", Text: "not modelled: this setting has no documented default value."},

	{ID: RA5ProfileEditSkips, Status: StatusAssumption, Assumption: 5, Text: "A later Profile change reaches an overriding Edge only for settings it did not override."},
	{ID: RA8RevertToProfile, Status: StatusAssumption, Assumption: 8, Text: "Removing an override restores the Profile value."},
	{ID: RA12InterfaceSegs, Status: StatusAssumption, Assumption: 12, Text: "An interface's segments must be among the segments of the Edge's Profile."},
	{ID: RA33OpaqueValues, Status: StatusAssumption, Assumption: 33, Text: "Setting values are opaque scalars: shown and compared, their meaning is not modelled."},
	{ID: RA34EdgeVlans, Status: StatusAssumption, Assumption: 34, Text: "An Edge may define its own VLANs beside inherited Profile VLANs; the 32-VLAN limit counts both."},
	{ID: RA35SwitchVlanByID, Status: StatusAssumption, Assumption: 35, Text: "On a Profile switch an interface is tied to a removed VLAN when its VLAN id is not defined by the new Profile."},
	{ID: RA36SectionUnknown, Status: StatusAssumption, Assumption: 36, Text: "Settings whose Device tab section was not read are grouped as 'Other (section not confirmed)'."},
	{ID: RA37AuthoredOrder, Status: StatusAssumption, Assumption: 37, Text: "Rules inside one group are listed in authored order; this is a listing, not an evaluation order."},
	{ID: RA38SwitchEdgeRules, Status: StatusAssumption, Assumption: 38, Text: "After a Profile switch the Edge's own rules are kept; rules on a segment the new Profile lacks are flagged."},
	{ID: RA39StatefulSetting, Status: StatusAssumption, Assumption: 39, Text: "Stateful Firewall is a Profile setting (default off) that an Edge may override."},
	{ID: RA41SameMatch, Status: StatusAssumption, Assumption: 41, Text: "An Edge rule supersedes a Profile rule when both are on the same segment with the same match (the guides say an identical rule with new parameters)."},
	{ID: RA40InterfaceVlans, Status: StatusAssumption, Assumption: 40, Text: "A LAN interface references a VLAN the Edge has (inherited or its own); other kinds carry no VLAN."},
}

var ruleByID = func() map[string]Rule {
	m := make(map[string]Rule, len(Rules))
	for _, r := range Rules {
		m[r.ID] = r
	}
	return m
}()

// RuleByID returns a registered rule. It panics on an unknown id: that is a programming error that
// rules_test.go catches (every cited id must be registered).
func RuleByID(id string) Rule {
	r, ok := ruleByID[id]
	if !ok {
		panic("sim: unregistered rule " + id)
	}
	return r
}

// basisOf is shorthand for RuleByID(id).Basis().
func basisOf(id string) Basis { return RuleByID(id).Basis() }
