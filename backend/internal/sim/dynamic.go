package sim

import (
	"fmt"
	"time"
)

// TeardownAfter is the documented silence after which a dynamic tunnel is torn down.
const TeardownAfter = 180 * time.Second // rule: dynamic-teardown-180

// trafficFor finds the traffic evidence for an unordered Edge pair. The first entry's A is the initiating Edge.
func (c *overlayCtx) trafficFor(a, b string) *TrafficSpec {
	for i := range c.cond.Traffic {
		tr := &c.cond.Traffic[i]
		if (tr.A == a && tr.B == b) || (tr.A == b && tr.B == a) {
			return tr
		}
	}
	return nil
}

// buildDecision is the explicit, configurable stub for "the initiating Edge determines that a secure overlay
// multi-path tunnel is possible". The real conditions are not documented (rule nm-secure-tunnel-decision).
func (c *overlayCtx) buildDecision(initiator, responder *edgeInfo) (ok bool, code, why string) {
	switch c.cond.DynamicBuild {
	case BuildAlways: // rule: assume-65
		return true, CodeOK, "the build decision was forced to always (a teaching switch)"
	case BuildNever:
		return false, CodeBuildNever, "the build decision was forced to never (a teaching switch)"
	}
	if responder.Edge.NATBlocksIncomingVCMP { // rule: dynamic-nat
		return false, CodeNATBlocks, fmt.Sprintf("%s sits behind a NAT that blocks incoming UDP 2426, so the direct tunnel will likely not come up", responder.Edge.Name) // rule: assume-16
	}
	return true, CodeOK, "no NAT blocks the direct tunnel" // rule: assume-16
}

// dynamicTunnels adds one dynamic branch-to-branch tunnel entry for every Edge pair, in whatever state the
// topology, the conditions and the traffic history put it.
func (c *overlayCtx) dynamicTunnels(ts *TunnelSet, now time.Time) {
	for i, a := range c.edges {
		for _, b := range c.edges[i+1:] {
			tn := c.dynamicTunnel(ts, a, b, now)
			ts.Tunnels = append(ts.Tunnels, tn)
		}
	}
}

func (c *overlayCtx) dynamicTunnel(ts *TunnelSet, a, b *edgeInfo, now time.Time) Tunnel {
	tr := c.trafficFor(a.Edge.ID, b.Edge.ID)
	init, resp := a, b
	if tr != nil && tr.A == b.Edge.ID {
		init, resp = b, a
	}
	tn := Tunnel{ID: fmt.Sprintf("dynamic:%s:%s", a.Edge.ID, b.Edge.ID), Type: TunDynamic, Mode: "dynamic", A: edgeEnd(a), B: edgeEnd(b), Paths: []Path{}, // rule: dynamic-first-packet
		Basis: basisOf(RDynFirstPacket), Also: bases(RDynDefault, RPathsMode, RA9DynamicAfterFlow, RA54BothDynamic)} // rule: assume-9
	notEst := func(code, reason string, extra ...string) Tunnel {
		tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, code, reason
		tn.Also = append(tn.Also, bases(extra...)...)
		return tn
	}
	switch {
	case !a.CVPN.Enabled || !b.CVPN.Enabled:
		return notEst(CodeCloudVPNOff, "Cloud VPN is not enabled on both Edges, so no branch-to-branch tunnel can be built", RA42CloudVPNOff)
	case a.CVPN.B2B.Enabled != b.CVPN.B2B.Enabled: // rule: nm-b2b-one-site
		tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMB2BOneSite, "Branch to Branch enabled at only one of two sites"))
		return notEst(CodeOneSiteOnly, "Branch to Branch VPN is enabled at only one of the two sites; what happens then is not documented")
	case !a.CVPN.B2B.Enabled: // rule: b2b-options
		return notEst(CodeB2BOff, "Branch to Branch VPN is not enabled, so there is no standing path and no direct tunnel", RB2BOptions)
	case !a.CVPN.DynamicOn || !b.CVPN.DynamicOn: // rule: assume-54
		return notEst(CodeDynamicOff, "Dynamic Branch to Branch is off for at least one of the two Edges")
	}
	if tr == nil {
		return notEst(CodeNoTraffic, "no traffic has flowed between these Edges: a direct tunnel is built on demand after the first packets", RA9DynamicAfterFlow)
	}
	silent := time.Duration(tr.LastPacketAgoS * float64(time.Second))
	if silent >= TeardownAfter { // rule: assume-52
		tn.State, tn.ReasonCode = StTornDown, CodeTornDown
		tn.Reason = fmt.Sprintf("no traffic for %.0f s (at least 180 s): the initiating Edge tore the direct tunnel down", silent.Seconds())
		tn.Also = append(tn.Also, bases(RDynTeardown, RA52Boundary, RA53Restart)...)                  // rule: assume-53
		tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMInflight, "In-flight flows at teardown")) // rule: nm-inflight-teardown
		tn.Initiator = init.Edge.ID
		return tn
	}
	ok, code, why := c.buildDecision(init, resp)
	tn.Initiator = init.Edge.ID
	tn.Also = append(tn.Also, bases(RDynTeardown, RA51NoBuildDelay, RA16BuildNAT)...)
	tn.NotModelled = append(tn.NotModelled, ts.addNM(RNMSecureTunnel, "The 'secure tunnel possible' decision")) // rule: nm-secure-tunnel-decision
	if !ok {
		tn.State, tn.ReasonCode, tn.Reason = StNotEstablished, code, why
		return tn
	}
	tn.Paths = c.edgePaths(a, b)
	capPaths(&tn)
	tn.State, tn.ReasonCode, tn.Reason = aggregate(tn.Paths)
	if tn.State == StUp {
		tn.Reason = fmt.Sprintf("built by %s after the first packets; silent for %.0f s of the 180 s allowed", init.Edge.Name, silent.Seconds())
	}
	return tn
}
