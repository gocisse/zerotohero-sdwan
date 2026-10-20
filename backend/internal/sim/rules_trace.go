package sim

// Rules added in step 14 (the flow tracer). Same convention as rules.go: code cites "// rule: <id>"; rules_test.go
// checks the evidence.
const (
	RBizHighestPrec  = "biz-highest-precedence"
	RFwActions       = "fw-actions"
	RFwPassThrough   = "fw-pass-through"
	RFwSymmetric     = "fw-stateful-symmetric"
	RRemOnDemand     = "remed-on-demand"
	RRemFEC          = "remed-fec"
	RRemJitter       = "remed-jitter-buffer"
	RRemNACK         = "remed-nack"
	RRemPremium      = "remed-premium"
	RMTUFragment     = "mtu-fragmentation"
	RNMBizDefault    = "nm-biz-default"
	RNMTraceReturn   = "nm-trace-return"
	RNMRemedThresh   = "nm-remed-thresholds"
	RA85FwEvaluation = "assume-85"
	RA86FwDefault    = "assume-86"
	RA87UnspecHost   = "assume-87"
	RA88Asymmetry    = "assume-88"
	RA89RemedClass   = "assume-89"
	RA90RuleFields   = "assume-90"
	RA91StageOrder   = "assume-91"
)

var traceRules = []Rule{
	{ID: RBizHighestPrec, Status: StatusVerified, Sheet: "business-policy", Claim: "Rules are listed in order of highest precedence, and traffic is matched to the rule with the highest precedence", Text: "Business rules are listed in order of precedence and traffic is matched to the rule with the highest precedence."},
	{ID: RFwActions, Status: StatusVerified, Sheet: "firewall", Claim: "Actions: Allow, Drop (silent), Reject (drop and send a reset), Skip (go to the next rule)", Text: "Firewall actions: Allow, Drop (silent), Reject (drop and send a reset), Skip (go to the next rule)."},
	{ID: RFwPassThrough, Status: StatusVerified, Sheet: "firewall", Claim: "Edge firewalls apply to traffic passing through the Edge, not to self-originating or self-destined traffic.", Text: "The Edge firewall applies to traffic passing through the Edge, not to the Edge's own traffic."},
	{ID: RFwSymmetric, Status: StatusVerified, Sheet: "firewall", Claim: "A stateful firewall enforces symmetric routing", Text: "A stateful firewall enforces symmetric routing; asymmetric routing is not supported on an Edge with Stateful Firewall."},
	{ID: RRemOnDemand, Status: StatusVerified, Sheet: "dmpo-remediation", Claim: "On-demand remediation is used when steering cannot fix the problem", Text: "On-demand remediation is used when steering cannot fix the problem (a single link, or several links degraded at once)."},
	{ID: RRemFEC, Status: StatusVerified, Sheet: "dmpo-remediation", Claim: "FEC triggers per flow, in real time, when loss crosses a threshold", Text: "FEC triggers per flow when loss crosses a threshold and applies to real-time traffic."},
	{ID: RRemJitter, Status: StatusVerified, Sheet: "dmpo-remediation", Claim: "Jitter buffer: enabled for real-time applications when jitter exceeds 7 ms (voice) or 5 ms (video)", Text: "The jitter buffer is enabled for real-time applications when jitter exceeds 7 ms (voice) or 5 ms (video)."},
	{ID: RRemNACK, Status: StatusVerified, Sheet: "dmpo-remediation", Claim: "NACK: for TCP applications such as file transfer", Text: "NACK retransmission applies to TCP applications such as file transfer."},
	{ID: RRemPremium, Status: StatusVerified, Sheet: "dmpo-remediation", Claim: "Premium Service on = Gateway applies FEC to impaired traffic", Text: "With Premium Service on the Gateway applies FEC to impaired traffic."},
	{ID: RMTUFragment, Status: StatusVerified, Sheet: "mtu-overhead", Claim: "When a packet is too large the Edge automatically performs IP fragmentation as per RFC 791.", Text: "When a packet is too large the Edge fragments it (IP fragmentation, RFC 791)."},
	{ID: RNMBizDefault, Status: StatusNotModelled, Sheet: "business-policy", Claim: "When no rule matches a default action applies; its definition is not given in text.", Text: "not modelled: the default action when no business rule matches. The trace continues as Multi-Path only because assumption 2 says so."},
	{ID: RNMTraceReturn, Status: StatusNotModelled, Sheet: "overlay-paths", Claim: "Return traffic from a destination outside the SD-WAN", Text: "not modelled: return traffic from a destination outside the SD-WAN, and whether the return path is the reverse of the forward path."},
	{ID: RNMRemedThresh, Status: StatusNotModelled, Sheet: "dmpo-remediation", Claim: "Numeric loss thresholds that switch FEC and NACK on per application type were image-only.", Text: "not modelled: the loss thresholds that switch FEC and NACK on, and whether remediation is active right now."},
	{ID: RA85FwEvaluation, Status: StatusAssumption, Assumption: 85, Text: "Firewall rules are evaluated first match wins in list order (Edge override rules before Profile rules; Skip moves to the next rule); the evaluation order is not documented, so last match is a selectable alternative."},
	{ID: RA86FwDefault, Status: StatusAssumption, Assumption: 86, Text: "When no firewall rule decides, the flow is allowed; the default action is not documented and is a selectable assumption."},
	{ID: RA87UnspecHost, Status: StatusAssumption, Assumption: 87, Text: "A flow with no source IP and no source interface is treated as LAN traffic passing through the Edge, so policy and firewall apply."},
	{ID: RA88Asymmetry, Status: StatusAssumption, Assumption: 88, Text: "With Stateful Firewall on, a flow counts as asymmetric when the forward and reverse path decisions choose different kinds of option."},
	{ID: RA89RemedClass, Status: StatusAssumption, Assumption: 89, Text: "Remediation candidates are classified from the flow: TCP is a NACK candidate; UDP real-time traffic (traffic class real_time or priority high) is an FEC and jitter buffer candidate."},
	{ID: RA90RuleFields, Status: StatusAssumption, Assumption: 90, Text: "A business or firewall rule matches the flow's own fields: addresses and prefixes by containment, ports as a number, list or range, 'internet' for a destination outside every prefix of the topology; a rule that needs a field the flow does not carry does not match."},
	{ID: RA91StageOrder, Status: StatusAssumption, Assumption: 91, Text: "The trace presents segment, policy, firewall, route, path, remediation, MTU and return path in that order; the real order of the policy, firewall and routing decisions is not documented."},
}
