package obsreport

import (
	"fmt"
	"strings"
)

// DebugItem is one command or option of the Edge or Gateway command line that appears in a source read for step 20. The
// report lists every one so the course owner knows exactly what to capture on a real Edge. Nothing here is an observation.
type DebugItem struct {
	Command  string // as the source prints it
	Device   string // Edge or Gateway command line, or "named only"
	Found    string // where it was found
	Claim    string // what the source says it does
	Capture  string // the exact thing to capture on a real Edge
	ReadOnly string // "yes", "no (state-changing: do not run)" or "unknown"
}

// DebugItems is the list. TestDebugItemsAreInTheFactSheet fails when a command here is not in content/reference/troubleshooting/ts-edge-cli.md.
var DebugItems = []DebugItem{
	{"debug.py --pki", "Edge or Gateway", "Orchestrator Guide 7.0, page 142", "shows PKI settings, the trusted CA list and the certificate",
		"the first 30 lines of the output and the Edge release; do not paste private keys", "yes"},
	{"debug.py --ike_sa", "Edge or Gateway", "Orchestrator Guide 7.0, page 142", "shows the details of current security associations",
		"the first 30 lines of the output with the Edge release (the guide shows three associations for the Orchestrator and two Gateways)", "yes"},
	{"debug.py --ike_spd", "Edge or Gateway", "Orchestrator Guide 7.0, pages 142 and 143", "shows the security protocols and encryption methods used per peer",
		"the first 30 lines of the output and the Edge release", "yes"},
	{"debug.py --routes", "named only", "Troubleshooting Guide 7.0, page 88", "its output carries the route flags n-nonVeloCloud and D- DCE (syntax and output not documented)",
		"the usage text line for --routes, and the first 20 lines of its output beside the Route Table Dump of the same Edge", "unknown"},
	{"sudo tail -f /var/log/mgd.log", "Edge", "Orchestrator Guide 7.0, page 141", "shows a certificate mismatch between the Edge and the Orchestrator",
		"the last 50 lines of /var/log/mgd.log (use tail -n 50, not follow) and the Edge release", "yes"},
	{"openssl x509 -text -noout -in cert.pem", "Edge or Gateway", "Orchestrator Guide 7.0, page 143", "reads a certificate file",
		"the certificate fields (subject, issuer, validity dates) of the certificate in /etc/vc_public, without any private key", "yes"},
	{"activate.py", "Edge", "Orchestrator Guide 7.0, page 125", "activates the device with an Orchestrator address and an activation key",
		"nothing: do not run it on a working Edge", "no (state-changing: do not run)"},
	{"debug.py --icmp_monitor", "Gateway", "Gateway Monitoring Guide 7.0, page 20", "prints the ICMP probe state, UP or DOWN, as JSON",
		"only if you have a Gateway: the output with the release; not an Edge command", "yes"},
	{"debug.py --bgp_view_summary", "Gateway", "Gateway Monitoring Guide 7.0, page 21", "used to count established BGP sessions",
		"only if you have a Gateway: the output with the release", "yes"},
	{"debug.py --handoff", "Gateway", "Gateway Monitoring Guide 7.0, page 24", "shows the handoff queue (with -v; a table without it)",
		"only if you have a Gateway: both forms of the output", "yes"},
	{"debug.py --list_edges", "Gateway", "Gateway Monitoring Guide 7.0, page 27", "counts connected Edges",
		"only if you have a Gateway: the output", "yes"},
	{"debug.py --vrf", "Gateway", "Gateway Monitoring Guide 7.0, page 29", "used to count BGP VRFs",
		"only if you have a Gateway: the output", "yes"},
	{"debug.py --stale_nat_dump", "Gateway", "Gateway Monitoring Guide 7.0, page 32", "shows stale NAT entries; collect it with a diagnostic bundle for support",
		"only if you have a Gateway: the first 20 lines", "yes"},
	{"debug.py --path", "Gateway", "Gateway Monitoring Guide 7.0, page 34", "shows path details such as the latency thresholds (with -v)",
		"only if you have a Gateway: the output", "yes"},
	{"debug.py --dpdk_ports_dump", "Gateway", "Gateway Monitoring Guide 7.0, page 17", "used inside a script in the guide",
		"only if you have a Gateway: the output", "yes"},
	{"/opt/vc/bin/getpolicy.py managementPlane", "Gateway", "Gateway Monitoring Guide 7.0, page 8", "reads the cached Gateway configuration",
		"only if you have a Gateway: the first 20 lines", "yes"},
}

// DebugQuestions are the open questions about the Edge command line that no source read answers.
var DebugQuestions = []struct{ Question, Capture string }{
	{"How do you get a shell on an Edge (console, SSH, a support login) and as which user?", "how you log in, the prompt, and the Edge release"},
	{"What is the full option list of debug.py on an Edge?", "the first lines of its usage text (run it with no option or with the help option it prints), with the Edge release"},
	{"Where does debug.py live on an Edge?", "the output of the command that locates it, and the folder listing"},
	{"Is there an Edge option for flows, paths, ARP or ND tables, VPN state, bandwidth or link statistics?", "whether the usage text lists one, and the first lines of its output next to the matching Remote Diagnostics test"},
	{"Which log files exist on an Edge besides /var/log/mgd.log?", "the folder listing of /var/log"},
	{"Do Remote Diagnostics tests and debug.py share an implementation?", "one test's output next to the matching debug.py option's output, same minute"},
}

func debugCaptureMarkdown() string {
	var b strings.Builder
	b.WriteString("## debug.py and Edge command line: what I need you to run\n\n")
	b.WriteString("Every command or option of the Edge or Gateway command line that appears in a source read for step 20 (`content/reference/troubleshooting/ts-edge-cli.md`), one row each. Only the Edge rows can be captured on an Edge; the Gateway rows are listed so you know they are **not** documented for an Edge. Run only read-only items, and write the result as an observation file (`docs/HOW-TO-ADD-AN-OBSERVATION.md`). Nothing here is an observation.\n\n")
	b.WriteString("| command | device | where found | what the source says | capture on a real device | read-only |\n|---|---|---|---|---|---|\n")
	for _, d := range DebugItems {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n", d.Command, d.Device, d.Found, d.Claim, d.Capture, d.ReadOnly)
	}
	b.WriteString("\nSearch results that could not be read as pages and are therefore not used: a Broadcom knowledge-base article on Partner Gateway troubleshooting commands (HTTP 404) and the Remote Diagnostics WebSocket API reference (HTTP 503).\n\n")
	b.WriteString("### Open questions with no documented answer\n\n| question | what to capture |\n|---|---|\n")
	for _, q := range DebugQuestions {
		fmt.Fprintf(&b, "| %s | %s |\n", q.Question, q.Capture)
	}
	b.WriteString("\n")
	return b.String()
}
