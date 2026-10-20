package sim

import (
	"fmt"
	"sort"
	"strings"
)

// Lost Reason codes the model reports; every one is a code the guides name (see rules_routing.go).
const (
	LRConnFlag      = "LR_CONN_FLAG"
	LRSwanVsVelo    = "LR_NP_SWAN_VS_VELO"
	LRExtOSPFInter  = "LR_EXT_OSPF_INTER"
	LRExtBGPRoute   = "LR_EXT_BGP_RT"
	LRExtMetricType = "LR_EXT_METRIC_TYPE"
	LRExtMetricVal  = "LR_EXT_METRIC_VAL"
	LRBGPLocalPref  = "LR_BGP_LOCAL_PREF"
	LRBGPASPathLen  = "LR_BGP_ASPATH_LEN"
	LRBGPMetric     = "LR_BGP_METRIC"
	LRLogicalID     = "LR_LOGICAL_ID"
)

// classes of the order of operations (after the longest prefix).
const (
	clsConnected = iota
	clsLocalStatic
	clsNSDLocal
	clsNSDRemote
	clsRemoteEdge
	clsDynamic
	clsLastResort
)

type cmpRes struct {
	diff  int
	code  string
	text  string
	basis Basis
	also  []Basis
	nm    string
}

func (s *rtState) classOf(r *Route) (cls, sub int) {
	switch r.Source {
	case SrcConnected:
		return clsConnected, 0
	case SrcLocalStatic:
		wan := 0
		if r.owner.static.Interface != "" {
			wan = 1
		}
		if !r.owner.static.Preferred { // rule: rt-static-last-resort
			return clsLastResort, wan
		}
		return clsLocalStatic, wan
	case SrcNSDLocal:
		return clsNSDLocal, 0
	case SrcNSDRemote:
		return clsNSDRemote, 0
	case SrcRemoteConnected:
		return clsRemoteEdge, 0
	case SrcRemoteStatic: // rule: rt-remote-order
		return clsRemoteEdge, 1
	}
	return clsDynamic, typePos(r.RouteType)
}

// typePos orders dynamic route types: O 1001, O IA 1002, eBGP 1003, iBGP 1004, then OE1 and OE2 after BGP.
func typePos(t string) int {
	switch t {
	case TypeO:
		return 1
	case TypeOIA:
		return 2
	case TypeEB:
		return 3
	case TypeIB:
		return 4
	case TypeOE1:
		return 5
	}
	return 6
}

func isOSPFType(t string) bool { return t == TypeO || t == TypeOIA || t == TypeOE1 || t == TypeOE2 }
func isBGPType(t string) bool  { return t == TypeEB || t == TypeIB }

func (s *rtState) label(r *Route) string {
	switch r.Source {
	case SrcConnected:
		return "the connected route on " + r.Via
	case SrcLocalStatic:
		return "the static route " + r.Via
	case SrcNSDLocal:
		return "the Non SD-WAN route via " + r.Via
	case SrcOSPF:
		return "the OSPF " + r.RouteType + " route"
	case SrcBGP:
		return "the " + r.RouteType + " route from " + strings.TrimPrefix(r.NextHop, "BGP neighbor ")
	}
	return "the " + strings.TrimPrefix(r.Source, "remote-") + " route from " + r.OriginName
}

func lp(r *Route) int {
	if r.LocalPreference == nil {
		return 100 // rule: assume-75
	}
	return *r.LocalPreference
}

func (s *rtState) hubIndex(y *edgeInfo, originEdge string) int {
	for i, h := range y.CVPN.B2H.Hubs {
		if h == originEdge {
			return i
		}
		if cl := s.c.clusters[h]; cl != nil && hasString(cl.Members, originEdge) {
			return i
		}
	}
	return -1
}

// compare orders two candidates of the same prefix on one Edge: negative when a is preferred. The first key that
// differs decides, and its reason is the Lost Reason of the loser.
func (s *rtState) compare(y *edgeInfo, a, b *Route) cmpRes {
	// Unreachable routes are kept but not selectable: they rank below every reachable route.
	if a.Reachable != b.Reachable {
		w := a
		d := -1
		if !a.Reachable {
			w, d = b, 1
		}
		_ = w
		return cmpRes{diff: d, text: "the other route is unreachable and cannot be selected", basis: basisOf(RRtUnreachable)} // rule: rt-unreachable-code
	}
	ca, sa := s.classOf(a)
	cb, sb := s.classOf(b)
	if ca != cb {
		d := ca - cb
		win, lose, wc, lc := a, b, ca, cb
		if d > 0 {
			win, lose, wc, lc = b, a, cb, ca
		}
		res := cmpRes{diff: d}
		switch {
		case wc == clsConnected: // rule: rt-connected-flag
			res.code, res.text, res.basis = LRConnFlag, "a connected route beats a non-connected one", basisOf(RRtConnFlag) // rule: rt-connected-flag
			res.also = bases(RRtOrderOfOps)                                                                                 // rule: rt-order-of-operations
		case lc == clsLastResort && lose.owner != nil && lose.owner.static.Interface != "" && (wc == clsRemoteEdge || wc == clsNSDRemote || (wc == clsDynamic && win.Origin != "")):
			res.code, res.text, res.basis = LRSwanVsVelo, "a non-preferred static WAN route loses to a route via VeloCloud", basisOf(RRtSwanVsVelo) // rule: rt-static-wan-vs-overlay
			res.also = bases(RRtLastResort)                                                                                                         // rule: rt-static-last-resort
		case lc == clsLastResort:
			res.text, res.basis = "a static route without Preferred is a last resort", basisOf(RRtLastResort) // rule: rt-static-last-resort
			if wc == clsDynamic && win.Origin == "" {
				res.also = bases(RA70LastResort) // rule: assume-70
			}
		default:
			res.text, res.basis = fmt.Sprintf("%s ranks above %s in the order of operations", classLabel(wc, win), classLabel(lc, lose)), basisOf(RRtOrderOfOps) // rule: rt-order-of-operations
		}
		// A routed (OSPF or BGP) route of this Edge against a remote Edge connected or static route: the rows conflict.
		if (wc == clsRemoteEdge && lc == clsDynamic && lose.Origin == "") || (lc == clsRemoteEdge && wc == clsDynamic && win.Origin == "") {
			res.nm = RNMRtRoutedVsVelo                                            // rule: nm-rt-routed-vs-overlay
			res.also = append(res.also, notModelled(RNMRtRoutedVsVelo, "").Basis) // rule: nm-rt-routed-vs-overlay
		}
		return res
	}
	if ca == clsDynamic {
		return s.compareDynamic(y, a, b)
	}
	// Same class, not dynamic: static cost (assumption 71), then the tie-breakers.
	if sa != sb {
		w := "LAN"
		if sb < sa {
			w = "WAN"
		}
		_ = w
		return cmpRes{diff: sa - sb, text: "a LAN static route is preferred over a WAN static route", basis: basisOf(RRtOrderOfOps)} // rule: rt-order-of-operations
	}
	if a.cmpCost != nil && b.cmpCost != nil && *a.cmpCost != *b.cmpCost { // rule: assume-71
		return cmpRes{diff: *a.cmpCost - *b.cmpCost, text: fmt.Sprintf("lower cost (%d against %d)", min(*a.cmpCost, *b.cmpCost), max(*a.cmpCost, *b.cmpCost)), basis: basisOf(RA71StaticCost)} // rule: assume-71
	}
	return s.tieBreak(y, a, b)
}

func classLabel(c int, r *Route) string {
	switch c {
	case clsConnected:
		return "a connected route"
	case clsLocalStatic:
		return "a local static route with Preferred"
	case clsNSDLocal:
		return "a local Non SD-WAN static route"
	case clsNSDRemote:
		return "a remote Non SD-WAN static route"
	case clsRemoteEdge:
		return "a remote Edge " + strings.TrimPrefix(r.Source, "remote-") + " route"
	case clsDynamic:
		return "a dynamic route"
	}
	return "a static route without Preferred"
}

func (s *rtState) compareDynamic(y *edgeInfo, a, b *Route) cmpRes {
	k := a.Segment + "|" + a.Prefix
	// Pinned routes get a preference starting from 1.
	pa, pb := s.pins[k][a.exitID], s.pins[k][b.exitID]
	if pa != pb && (pa != 0 || pb != 0) { // rule: rt-ofc-pin
		d := 1
		if pa != 0 && (pb == 0 || pa < pb) {
			d = -1
		}
		return cmpRes{diff: d, text: "the route is pinned in Overlay Flow Control (preference starts at 1)", basis: basisOf(RRtPin)} // rule: rt-ofc-pin
	}
	ex := s.exits[k]
	if len(ex) > 0 { // rule: rt-ofc-preferred-exits
		ia, ib := idx(ex, a.exitID), idx(ex, b.exitID)
		if ia != ib {
			return cmpRes{diff: ia - ib, text: "Preferred VPN Exits for this subnet list its exit first", basis: basisOf(RRtOFCExits)} // rule: rt-ofc-preferred-exits
		}
	}
	ta, tb := typePos(a.RouteType), typePos(b.RouteType)
	if ta != tb {
		d := ta - tb
		win, lose := a.RouteType, b.RouteType
		if d > 0 {
			win, lose = lose, win
		}
		res := cmpRes{diff: d}
		switch {
		case isOSPFType(win) && isOSPFType(lose):
			res.code, res.text, res.basis = LRExtMetricType, fmt.Sprintf("OSPF type %s beats %s (order O, O IA, O E1, O E2)", win, lose), basisOf(RRtOSPFTypeOrder) // rule: rt-ospf-type-order
		case (win == TypeO || win == TypeOIA) && isBGPType(lose):
			res.code, res.text, res.basis = LRExtOSPFInter, fmt.Sprintf("an OSPF %s route beats a BGP route", win), basisOf(RRtOSPFvsBGP) // rule: rt-ospf-beats-bgp
			res.also = bases(RRtDefaultPrefs)                                                                                             // rule: rt-default-preferences
		case isBGPType(win) && (lose == TypeOE1 || lose == TypeOE2):
			res.code, res.text, res.basis = LRExtBGPRoute, fmt.Sprintf("a BGP route beats an OSPF %s route", lose), basisOf(RRtBGPvsExternal) // rule: rt-bgp-beats-external-ospf
		default: // eBGP against iBGP
			res.text, res.basis = "lower route preference: eBGP 1003 against iBGP 1004", basisOf(RRtLowerWins) // rule: rt-lower-preference-wins
			res.also = bases(RRtDefaultPrefs)                                                                  // rule: rt-default-preferences
		}
		if res.diff == 0 {
			res.diff = d
		}
		return res
	}
	// Same type: protocol attributes.
	switch {
	case isBGPType(a.RouteType):
		if la, lb := lp(a), lp(b); la != lb { // rule: rt-bgp-attributes
			d := 1
			if la > lb {
				d = -1
			}
			s.noteDefaultLP(a, b)
			return cmpRes{diff: d, code: LRBGPLocalPref, text: fmt.Sprintf("higher local preference (%d against %d)", max(la, lb), min(la, lb)), basis: basisOf(RRtBGPAttrs), also: bases(RA24BGPOrder)} // rule: rt-bgp-attributes // rule: assume-24
		}
		if la, lb := len(a.ASPath), len(b.ASPath); la != lb {
			return cmpRes{diff: la - lb, code: LRBGPASPathLen, text: fmt.Sprintf("shorter AS path (%d against %d)", min(la, lb), max(la, lb)), basis: basisOf(RRtBGPAttrs), also: bases(RA24BGPOrder)} // rule: rt-bgp-attributes // rule: assume-24
		}
		ma, mb := 0, 0
		if a.Metric != nil {
			ma = *a.Metric
		}
		if b.Metric != nil {
			mb = *b.Metric
		}
		if ma != mb {
			return cmpRes{diff: ma - mb, code: LRBGPMetric, text: fmt.Sprintf("lower metric (%d against %d)", min(ma, mb), max(ma, mb)), basis: basisOf(RRtBGPAttrs), also: bases(RA24BGPOrder)} // rule: rt-bgp-attributes // rule: assume-24
		}
	case isOSPFType(a.RouteType):
		if a.cmpCost != nil && b.cmpCost != nil && *a.cmpCost != *b.cmpCost { // rule: rt-ospf-type-order
			return cmpRes{diff: *a.cmpCost - *b.cmpCost, code: LRExtMetricVal, text: fmt.Sprintf("lower OSPF metric (%d against %d)", min(*a.cmpCost, *b.cmpCost), max(*a.cmpCost, *b.cmpCost)), basis: basisOf(RRtOSPFTypeOrder)} // rule: rt-ospf-type-order
		}
	}
	return s.tieBreak(y, a, b)
}

func (s *rtState) noteDefaultLP(a, b *Route) {
	if a.LocalPreference == nil || b.LocalPreference == nil { // rule: assume-75
		s.nm(RNMRtASPath, "How the overlay changes route attributes") // rule: nm-rt-overlay-attributes
	}
}

func idx(list []string, id string) int {
	for i, x := range list {
		if x == id {
			return i
		}
	}
	return len(list) + 1
}

// tieBreak applies Hub order, then the Logical ID, then (not modelled) route age.
func (s *rtState) tieBreak(y *edgeInfo, a, b *Route) cmpRes {
	if ha, hb := s.hubIndex(y, a.exitID), s.hubIndex(y, b.exitID); ha >= 0 && hb >= 0 && ha != hb { // rule: hub-order-tiebreak
		return cmpRes{diff: ha - hb, text: "the Hub earlier in the Hub order wins the tie", basis: basisOf(RHubOrder), also: bases(RA59HubAttrsEqual)}
	}
	ea, eb := s.c.edgeByID[a.exitID], s.c.edgeByID[b.exitID]
	if ea != nil && eb != nil && ea.Edge.LogicalID != nil && eb.Edge.LogicalID != nil && *ea.Edge.LogicalID != *eb.Edge.LogicalID { // rule: rt-logical-id-tiebreak
		s.use("logical_id_tiebreak")
		d := *ea.Edge.LogicalID - *eb.Edge.LogicalID
		if s.as.LogicalIDTieBreak == TieHigher { // rule: assume-74
			d = -d
		}
		return cmpRes{diff: d, code: LRLogicalID, text: fmt.Sprintf("the %s Logical ID of the next-hop device wins the tie", s.as.LogicalIDTieBreak), basis: basisOf(RRtLogicalID), also: bases(RA74LogicalIDDir, RRtTieBreakers)} // rule: rt-tie-breakers // rule: rt-logical-id-tiebreak // rule: assume-74
	}
	// Equal in every documented key: the stable order below keeps results deterministic, and the tie is flagged.
	s.nm(RNMRtTieAge, "Route age and the Logical ID direction")                                                                                                                                                                                                             // rule: nm-rt-logical-id-and-age
	return cmpRes{diff: strings.Compare(a.stableKey(), b.stableKey()), text: "equal in every documented key; route age decides in the product and is not modelled, so the order shown is arbitrary but stable", basis: notModelled(RNMRtTieAge, "").Basis, nm: RNMRtTieAge} // rule: nm-rt-logical-id-and-age
}

func (r *Route) stableKey() string {
	return r.Source + "|" + r.exitID + "|" + r.Via + "|" + r.RouteType
}

// selectAll orders every table, marks the selected route and gives each route its rank, reason and Lost Reason.
func (s *rtState) selectAll() {
	for _, e := range s.c.edges {
		for _, seg := range e.Profile.Segments {
			key := e.Edge.ID + "|" + seg
			byPrefix := map[string][]*Route{}
			var order []string
			for _, r := range s.tables[key] {
				p := r.Prefix
				if _, ok := byPrefix[p]; !ok {
					order = append(order, p)
				}
				byPrefix[p] = append(byPrefix[p], r)
			}
			for _, p := range order {
				c := byPrefix[p]
				sort.SliceStable(c, func(i, j int) bool { return s.compare(e, c[i], c[j]).diff < 0 })
				s.annotate(e, c)
			}
		}
	}
}

func (s *rtState) annotate(e *edgeInfo, c []*Route) {
	selected := false
	for i, r := range c {
		r.Rank = i + 1
		r.Selected = !selected && r.Reachable
		if r.Selected {
			selected = true
		}
		switch {
		case !r.Reachable:
			r.Reason = "Not selectable: " + orS(r.ReachableReason, "unreachable") + " (" + r.ReachableCode + ")"
			r.LostReasonCode = ""
			if i > 0 && c[i-1].Reachable {
				r.Reason = "Lost to " + s.label(c[i-1]) + ": it is unreachable and cannot be selected. " + r.Reason
			}
			continue
		case i == 0:
			r.Reason = "Selected: the best candidate for this prefix."
			if len(c) > 1 && c[1].Reachable {
				res := s.compare(e, r, c[1])
				r.Reason = fmt.Sprintf("Selected: it beats %s (%s).", s.label(c[1]), res.text)
				r.Also = append(r.Also, res.basis)
				r.Also = append(r.Also, res.also...)
				s.flagNM(r, res)
			}
			if len(c) == 1 {
				r.Reason = "Selected: the only candidate for this prefix."
			}
		default:
			prev := c[i-1]
			res := s.compare(e, prev, r)
			r.LostReasonCode = res.code
			if prev.Reachable {
				r.Reason = fmt.Sprintf("Lost to %s: %s.", s.label(prev), res.text)
			} else {
				r.Reason = "Lost: an earlier candidate is preferred."
				r.Selected = false
			}
			r.Also = append(r.Also, res.basis)
			r.Also = append(r.Also, res.also...)
			s.flagNM(r, res)
		}
		if r.Selected {
			r.Reason += " This is the route in the forwarding table."
		}
	}
	// The first reachable route is the selected one even if unreachable routes rank above it (they never do:
	// unreachable routes sort last), so nothing more to do here.
}

func (s *rtState) flagNM(r *Route, res cmpRes) {
	if res.nm != "" {
		nm := s.nm(res.nm, "Route comparison")
		r.NotModelled = append(r.NotModelled, nm)
	}
}

func orS(a, b string) string {
	if a == "" {
		return b
	}
	return a
}
