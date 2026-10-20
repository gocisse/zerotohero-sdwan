package sim

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// RuleCtx is what a business rule can be matched against. Fields the flow does not carry are left empty and a rule
// that needs them does not match (assumption 90).
type RuleCtx struct {
	Segment     string
	Application string
	// DestKind is edge, nsd, internet or ip (an address inside the topology); DestIP is set when known.
	DestKind string
	DestIP   netip.Addr
	SrcIP    netip.Addr
	Protocol string
	Port     int
	DSCP     string
}

// RuleVerdict is one rule's evaluation for a flow: matched or not, and why.
type RuleVerdict struct {
	RuleID  string  `json:"rule_id"`
	Name    string  `json:"name,omitempty"`
	Owner   string  `json:"owner"`
	Matched bool    `json:"matched"`
	Chosen  bool    `json:"chosen"`
	Reason  string  `json:"reason"`
	Basis   Basis   `json:"basis"`
	Also    []Basis `json:"also,omitempty"`
}

// RuleEval is the outcome of evaluating a segment's business rules for one flow.
type RuleEval struct {
	Matched  *BusinessRule
	Owner    string
	Effect   *EffectiveRule
	Verdicts []RuleVerdict
}

func portMatches(spec string, port int) (bool, bool) {
	ok := false
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if lo, hi, found := strings.Cut(part, "-"); found {
			a, e1 := strconv.Atoi(strings.TrimSpace(lo))
			b, e2 := strconv.Atoi(strings.TrimSpace(hi))
			if e1 != nil || e2 != nil {
				return false, false
			}
			ok = ok || (port >= a && port <= b)
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return false, false
		}
		ok = ok || port == n
	}
	return ok, true
}

func prefixOrAddr(s string) (netip.Prefix, bool) {
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked(), true
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return netip.PrefixFrom(a, a.BitLen()), true
	}
	return netip.Prefix{}, false
}

// ruleMatch says whether a business rule's match accepts the flow, with the first field that fails. rule: assume-90
func ruleMatch(m RuleMatch, c RuleCtx) (bool, string) { // rule: assume-90
	if m.Application != "" {
		switch {
		case c.Application == "":
			return false, "the rule matches application " + m.Application + " and the flow carries no application"
		case m.Application != c.Application:
			return false, fmt.Sprintf("application %s does not match %s", c.Application, m.Application)
		}
	}
	if m.Source != "" && m.Source != "any" {
		p, ok := prefixOrAddr(m.Source)
		switch {
		case !ok:
			return false, "the source match " + m.Source + " is not an address or prefix"
		case !c.SrcIP.IsValid():
			return false, "the rule matches source " + m.Source + " and the flow carries no source IP"
		case !p.Contains(c.SrcIP):
			return false, fmt.Sprintf("source %s is outside %s", c.SrcIP, p)
		}
	}
	switch m.Destination {
	case "", "any":
	case "edge":
		if c.DestKind != "edge" && c.DestKind != "ip" {
			return false, "the rule matches destinations inside the SD-WAN and this one is " + c.DestKind
		}
	case "nsd":
		if c.DestKind != "nsd" {
			return false, "the rule matches Non SD-WAN destinations and this one is " + c.DestKind
		}
	case DestInternet:
		if c.DestKind != DestInternet {
			return false, "the rule matches Internet destinations and this one is " + c.DestKind
		}
	default:
		p, ok := prefixOrAddr(m.Destination)
		switch {
		case !ok:
			return false, "the destination match " + m.Destination + " is not a keyword, address or prefix"
		case !c.DestIP.IsValid():
			return false, "the rule matches destination " + m.Destination + " and the flow carries no destination IP"
		case !p.Contains(c.DestIP):
			return false, fmt.Sprintf("destination %s is outside %s", c.DestIP, p)
		}
	}
	if m.Protocol != "" && !strings.EqualFold(m.Protocol, c.Protocol) {
		if c.Protocol == "" {
			return false, "the rule matches protocol " + m.Protocol + " and the flow carries none"
		}
		return false, fmt.Sprintf("protocol %s does not match %s", c.Protocol, m.Protocol)
	}
	if m.Port != "" {
		if c.Port == 0 {
			return false, "the rule matches port " + m.Port + " and the flow carries none"
		}
		ok, valid := portMatches(m.Port, c.Port)
		switch {
		case !valid:
			return false, "the port match " + m.Port + " is not a port, list or range"
		case !ok:
			return false, fmt.Sprintf("port %d does not match %s", c.Port, m.Port)
		}
	}
	if m.DSCP != "" && m.DSCP != c.DSCP {
		return false, fmt.Sprintf("DSCP %q does not match %s", c.DSCP, m.DSCP)
	}
	return true, "every field of the rule matches"
}

// evaluateRules evaluates every business rule of the flow's segment, in precedence order, and picks the first that
// matches (assumptions 1 and 3). Superseded Profile rules are listed but never match. edgeFirst puts Edge rules
// above Profile rules (assumption 3); false reverses the two groups.
func evaluateRules(ec *Effective, c RuleCtx, edgeFirst bool) RuleEval {
	var ev RuleEval
	for _, sp := range ec.BusinessPolicy {
		if sp.Segment.ID != c.Segment {
			continue
		}
		var cands []EffectiveRule
		profile := []EffectiveRule{}
		for _, r := range sp.FromProfile {
			profile = append(profile, r)
		}
		if edgeFirst { // rule: assume-3
			cands = append(append(cands, sp.EdgeOverrides...), profile...)
		} else {
			cands = append(append(cands, profile...), sp.EdgeOverrides...)
		}
		for i := range cands {
			r := cands[i]
			v := RuleVerdict{RuleID: r.Rule.ID, Name: r.Rule.Name, Owner: r.Owner.Name, Basis: basisOf(RBizHighestPrec), Also: bases(RA1FirstMatch, RA90RuleFields)} // rule: biz-highest-precedence // rule: assume-1 // rule: assume-90
			switch {
			case r.SupersededBy != "":
				v.Reason = "superseded by the Edge rule " + r.SupersededBy + " with the same match"
				v.Basis = basisOf(RBizSupersede)
			case ev.Matched != nil:
				v.Reason = "not evaluated: " + ev.Matched.ID + " matched first"
			default:
				ok, why := ruleMatch(r.Rule.Match, c)
				v.Matched, v.Reason = ok, why
				if ok {
					rule := r.Rule
					ev.Matched, ev.Owner = &rule, r.Owner.Name
					eff := cands[i]
					ev.Effect = &eff
					v.Chosen = true
				}
			}
			ev.Verdicts = append(ev.Verdicts, v)
		}
	}
	if ev.Verdicts == nil {
		ev.Verdicts = []RuleVerdict{}
	}
	return ev
}
