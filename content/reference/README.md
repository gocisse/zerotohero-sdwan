# Reference fact sheets

Author-only research notes that back the course lessons. Learners never see these files. The backend does not treat this folder as
course content (no orphan error), but it **does** validate every file here except this README.

## Format

Each fact sheet is Markdown with one or more claims tables:

```
| claim | source URL | doc version | status |
|---|---|---|---|
| One checkable statement, in our own words. | https://exact-page-url | 7.0 | VERIFIED |
```

- **VERIFIED**: read in an official source (Arista, or Broadcom/VMware documentation or filings). The URL is the page that was read.
- **SECONDARY**: found only in non-official sources (news, blogs). A URL is still required.
- **VERIFY**: could not be confirmed (page unreachable, behind a login, only seen in a search snippet, or not found). The URL may be `none`.
- Rules enforced by `make validate-content`: every file except this README needs a claims table with the exact header above; each row has four
  cells; status is one of the three words; VERIFIED and SECONDARY rows must have an `http(s)` URL.
- Never invent a source. If a page could not be read, say so and use VERIFY. A claim written in a lesson must not be stronger than its row.
- Claims are paraphrased. Do not paste vendor text into lessons.

## How sources were read

- **Read directly**: PDFs downloaded and converted to text by us (7.0 Orchestrator, Troubleshooting and Gateway Monitoring Guides; the Edge 7x0 datasheet; the 6.4 release notes; the solution brief). Rows say "read directly".
- **Summarised via WebFetch**: Arista's HTML guide pages. Arista's website serves a CAPTCHA to automated browsers (not circumvented), so raw HTML was not obtainable; the text came through a summarising tool with verbatim quotes requested. Rows say "summarised via WebFetch". Step-5 rows from HTML pages were gathered the same way and carry the same caveat.
- Never invent a source: where a page could not be read, the row is VERIFY.

Facts were gathered on **2026-10-19** against the **7.0** documentation set on arista.com (see `release-target.md`).

## Fact sheets

| file | covers |
|---|---|
| `release-target.md` | which release the course targets and the evidence |
| `edge-details.md` | what the Edge does, interfaces, WAN overlay, local services, activation, inheritance |
| `orchestrator-details.md` | what the Orchestrator runs and stores, DR, certificates, portals, what breaks if it is unreachable |
| `gateway-details.md` | Gateway roles, Partner Gateway, software structure, what it caches |
| `tunnel-types.md` | Branch to Hub, Branch to Branch, dynamic tunnels, NAT effects, non-SD-WAN tunnels (high level) |
| `ports-protocols.md` | per-flow port and protocol table |
| `wan-background.md` | MPLS, broadband, backhauling, cloud and SaaS drivers |
| `sdwan-concepts.md` | what SD-WAN is, policy, zero-touch, activation |
| `components.md` | Edge, Orchestrator, Gateway; what talks to what |
| `underlay-overlay.md` | transport networks, overlay tunnel types, dynamic branch-to-branch |
| `vcmp-tunnels.md` | VCMP, ports and protocols, firewall rules |
| `mtu-overhead.md` | VCMP overhead, effective MTU, path MTU discovery |
| `dmpo.md` | monitoring, steering modes, on-demand remediation |
| `branding-history.md` | VeloCloud, VMware, Broadcom, Arista |
| `vco-ui-map.md` | where concepts live in the Orchestrator UI |
| `routing-overview.md`, `static-routes.md`, `ospf.md`, `bgp.md`, `route-redistribution-and-precedence.md`, `overlay-flow-control.md`, `vco-routing-ui.md` | step 10: routing (VCRP, route table, static, OSPF, BGP, redistribution and preference, Overlay Flow Control, VCO screens) |

## Checks of the numbers from the course outline

_Written in step 5. Items 2, 4 and 5 below have since been updated: see "Step 6: resolving the open items"._

| outline statement | result | what the sources say |
|---|---|---|
| VCMP default port UDP 2426 | **confirmed** | The 7.0 Administration Guide: VCMP uses the IANA-registered UDP port 2426, for tunnel management and information exchange between two DMPO endpoints. Arista's virtual-edge page lists outbound UDP/2426 (VCMP). |
| UDP 500 / UDP 4500 | **confirmed, but narrower than the outline** | 7.0 overview: IKEv2 negotiation and IPsec NAT transparency between DMPO endpoints (Edge-Gateway, Edge-Edge). Nothing read ties them to Edge-to-Orchestrator traffic. |
| ESP (IP protocol 50) | **confirmed, with a condition** | 7.0 overview: used for IPsec "when no NAT exists" between the two DMPO endpoints. The same guide also says VCMP is encrypted with transport-mode IPsec and NAT-T is forced on with a special port, 2426, so the exact wire format in each case is **unconfirmed** (VERIFY). |
| Inbound 2426 needed for hubs, dynamic branch-to-branch and NAT | **unconfirmed** | Only seen in search snippets of a Broadcom knowledge-base article that returned 404 to the fetch tool. Dynamic branch-to-branch itself (direct tunnel on demand, torn down after 180 s of silence) is confirmed. |
| Management traffic on TCP 443 | **confirmed narrowly** | Arista's 7.0 virtual-edge page: the firewall must allow outbound TCP/443 for Edge-to-Orchestrator communication. Whether all management traffic uses 443 was not read (VERIFY). |
| VCMP overhead about 60-70 bytes | **corrected** | VCMP itself adds 31 bytes (23-byte header + 8-byte data header). The total encapsulation is 59 bytes for Internet Multipath traffic (IP 20 + UDP 8 + VCMP 31) and 97-112 bytes for VPN (IPsec) traffic. "60-70" matches neither exactly. |
| Effective MTU 1441 | **corrected** | 1441 is 1500 - 59: valid only for Internet Multipath traffic on a 1500-byte path MTU, and the guide does not print that number (it says the overhead is subtracted). VPN traffic leaves 1388-1403 bytes. Jumbo frames are unsupported; discovery uses RFC 1191 then RFC 4821 (1300 byte floor). |
| DMPO: FEC, packet duplication, jitter buffer and when each applies | **confirmed, with a terminology correction** | In the 7.0 guide FEC *is* packet duplication (adjacent copy on one link, copies on separate links when several exist; up to two best links), triggered per flow when loss crosses a threshold, for real-time traffic. Jitter buffer: real-time apps when jitter exceeds 7 ms (voice) or 5 ms (video), up to 10 packets or 200 ms, off if the flow is not constant. TCP apps get NACK retransmission. The guide says "outage" and "brownout"; "blackout" never appears. Per-application SLA threshold tables were not read (VERIFY). |
| Link steering: auto, transport group, interface, WAN link | **confirmed** | 7.0 Business Policy page: the rule's Link Steering modes are Auto (default), Transport Group, Interface, WAN Link. |
| Mandatory / preferred / available semantics | **confirmed, but a different axis** | They are not extra "modes": per the Business Policy page they are choices for Interface and WAN Link steering, and the overview page describes them (Mandatory pins and never steers away, even in an outage; Preferred stays while the SLA holds and steers to other public links; Available stays while the link is up). The overview also lists "Auto, Preferred, Mandatory and Available" as the four options, so the guides frame it two ways. A 6.4 snippet saying Mandatory drops traffic when the link is down is unconfirmed. |

## Step 6: resolving the open items

| # | item | status now | evidence and how it was read |
|---|---|---|---|
| 1 | 7.0.0 GA / LTS | **Still VERIFY** (more evidence that 7.0.0 shipped) | Edge 7x0 datasheet (read directly): Edges must be on "R7.0.0" for the listed performance. Advisory 0144 (2026-07-27, summarised) lists VCO 7.0.x prior to 7.0.0.1 as affected; advisory 0183 (2026-09-22) lists 7.0.0.2 and below. Tried and failed: 7.0.0 release notes (a guessed PDF address returned 400; Arista's site blocks automated browsers), the Recommended Releases article (loading stub), the product-bulletin listing (no addresses visible). No page read labels 7.0.0 GA, LTS or STS. |
| 2 | Inbound UDP 2426 | **Partly resolved** | VERIFIED (7.0 Edge settings, summarised): behind NAT the direct VCMP tunnel on UDP/2426 will not likely come up unless incoming 2426 from other Edges is allowed; hole punching is a limited workaround. VERIFIED (Partner Guides, summarised): Partner Gateway needs inbound 2426, 4500, 500. Still VERIFY: an Edge acting as a Hub, and the NAT recommendation (only in snippets of Broadcom KB 312353, which the fetch tool could not open; the old VMware address redirects to Broadcom's home page). |
| 3 | Partner Gateway ports | **Resolved** | 7.0 and 6.4 Partner Guide "Install Partner Gateway" pages (summarised, verbatim quotes requested): inbound UDP/2426, UDP/4500, UDP/500; ESP when NAT is not used; outbound TCP/443 to the Orchestrator; ports translated if NAT is used (1:1 and port translation). |
| 4 | UDP 2426 / 500,4500 / ESP per NAT case | **Partly resolved** | 7.0 overview (summarised): VeloCloud behaves the same with or without NAT because NAT-T is always activated, with special NAT-T port 2426; ports 500/4500 for IKEv2 and NAT transparency; ESP for IPsec when no NAT. How ESP is used given always-on NAT-T was not spelled out, so that part stays VERIFY. |
| 5 | Is all management traffic TCP 443? | **Resolved for the documented flows** | Edge to Orchestrator and Gateway to Orchestrator are outbound TCP/443 rules (7.0 Virtual Edge and Partner Guide pages, summarised); browser and on-prem Edge management are HTTPS (Orchestrator Guide, read directly); the Orchestrator-Edge channel is a persistent WebSocket over TCP (Troubleshooting Guide, read directly). No page claims 443 is the only management port, and the WebSocket port is not stated. |
| 6 | Default and maximum overlay MTU | **Resolved** | 7.0 overview (summarised, verbatim): path MTU discovery starts from a default of 1500; the maximum supported IP MTU across the overlay without fragmentation is 1500; the Edge fragments per RFC 791; no jumbo frames. |
| 9 | UI labels and menu paths for Section 2 | **Mostly resolved** | "Diagnostics" tab confirmed by the 7.0 Troubleshooting Guide (read directly); Configure > Segments, Configure > Network Services, Service Settings > Edge Management, Configure > Edges > More > Assign Software Image, Insights > Link Performance, Operator portal Gateway Management > Gateways (summarised or read directly as noted in `vco-ui-map`). Still VERIFY: whether the live UI also shows "Test & Troubleshoot", paths for user management and licensing pages, the new vs classic UI. |
| (new) | Planes wording | **Resolved** | The 7.0 Orchestrator Guide (read directly) calls the Orchestrator the management plane, the Controller (Gateway) the control and routing plane, and the Hub Cluster the data plane. The three-planes framing is Arista language. |

## Step 7: DMPO verification pass

New sheets: `dmpo.md` (rewritten), `dmpo-monitoring.md`, `dmpo-steering.md`, `dmpo-remediation.md`, `dmpo-load-balancing.md`, `dmpo-vco-ui.md`.
Each sheet records the release each row comes from. Finding: in the pages we could read, DMPO behaviour text is the same in 6.4 and 7.0 (overview wording on FEC, jitter buffer, NACK,
detection time, hold time and Mandatory/Preferred/Available). The 2017 legacy white paper differs in vocabulary ("blackout", "VCMP adds 59 bytes" where Arista now splits VCMP 31 + IP 20 + UDP 8) and is used
only for SECONDARY rows, never for a quiz answer.

| # | item | status now | evidence and how it was read |
|---|---|---|---|
| 1 | 7.0.0 GA / LTS | **Still VERIFY** | One more attempt: no local PDF labels 7.0.0 GA or LTS (checked the Orchestrator, Troubleshooting, Gateway Monitoring and Edge datasheet text). No further effort spent, as instructed. |
| 5 | Per-application SLA tables | **Partly resolved** | Customizable QoE latency range (1 to 1000 ms, Edge 5.2.0 or later) and the Real-Time loss default of 0.3% for Error Correct Before Steering were read (summarised). The per-class loss and jitter threshold tables are images in the Administration Guide; the text is not extractable, and the Administration Guide PDF is over the 10 MB limit. Stays VERIFY. |
| 6 | Mandatory wording; Gateway FEC under Premium Service | **Resolved** | 7.0 business-policy Table 6 (summarised) says Mandatory drops the packet if the link or group is inactive; the overview says it never steers away: same behaviour from two sides. Premium Service: 7.0 Operator Guide "Manage Customers" (summarised): on by default; Gateway applies FEC to impaired traffic; off means no Gateway error correction. The 6.4 policy table was not read, so the "6.4 says drops" snippet is not confirmed for 6.4. |
| 7 | UI labels | **Mostly resolved** | "Diagnostics" is the current name (5.2 UI-changes page, summarised; 7.0 Troubleshooting Guide, read directly). The new UI is described as the common management system with "Enable Classic Orchestrator UI" as a customer capability. Which UI is the default and how to switch were not stated: VERIFY. |
| 8 | Edge without the Orchestrator | **Still VERIFY** | Searched the readable Troubleshooting, Orchestrator, Gateway Monitoring text and the 7.0 overview; no statement of forwarding behaviour. |
| 12 | Gateway placement | **Resolved** | Arista Cloud Gateway Advantages white paper, 30 June 2025 (read directly): more than 30 regions, more than 3000 Gateways, primary and secondary Gateway assignment by geolocation. Dynamic-tunnel trigger conditions remain VERIFY. |

New VERIFY items from step 7: bandwidth aggregation conditions in current releases and whether Direct traffic is still load-balanced per flow; scheduler default weights (image); numeric loss thresholds for FEC and NACK; whether the VCO shows that a remediation mechanism is active.

## Open VERIFY items (updated)

1. **7.0.0 GA / LTS status** and build strings.
2. **Edge acting as a Hub: inbound UDP 2426**, and the "open 2426 instead of hole punching" recommendation (Broadcom KB unreadable).
3. **How ESP coexists with the always-on NAT-T port 2426** (which cases use ESP on the wire).
4. **DNS (UDP 53) and NTP (UDP 123)** requirements for an Edge (only in snippets).
5. **Per-application SLA threshold tables** for DMPO (image only; the latency customisation range and the 0.3% Real-Time loss default were read).
6. ~~6.4 Mandatory wording; Premium Service~~ **resolved in step 7** (see the step 7 table); the 6.4 policy table itself was not read.
7. **UI labels**: which UI (new or classic) is the default and how to switch; paths for user management and licensing. ("Diagnostics" confirmed.)
8. **Edge behaviour when the Orchestrator is unreachable** (forwarding and last configuration), and whether an Edge has a local web UI.
9. **Terminology**: the "VCE" and "VCG" acronyms (VCO appears in an advisory; on-prem docs say Controller / VECC), "underlay", "transport independence", "DIA".
10. **History details**: exact Broadcom-VMware close date (title only), founding year, product names over time.
11. **General WAN claims** with no source read: carrier-provisioned MPLS with contractual service levels, deliberate backhauling.
12. ~~Gateway placement~~ **resolved in step 7**; exact dynamic-tunnel trigger conditions remain VERIFY.
15. **Overlay (step 9)**: see the step 9 list of items needing a real-VCO observation.
16. **Routing (step 10)**: see the step 10 list of items needing a real-VCO observation.
14. **Configuration model** (step 8): see the step 8 table (override marking and reverting, overridable-settings list, rule evaluation order, firewall versus business policy order, segment-to-segment traffic, Profile-switch effects, Quick Start contents, Applications screen, DPI timing).
13. **DMPO details** (step 7): bandwidth aggregation conditions and per-flow load balancing of Direct traffic in current releases; scheduler default weights; numeric FEC and NACK loss thresholds; whether the VCO shows an active remediation mechanism.

## Section 1 claim audit (self-review, 2026-10-19)

Every lesson was re-read against the fact sheets. Sentences that make a technical claim **without** a VERIFIED row, and what was done:

| where | sentence or idea | resolution |
|---|---|---|
| Lesson 1 | "MPLS circuits are carrier-provisioned with contractual service levels"; the meaning of "DIA" | Not stated as fact in the lesson body; named in a visible **warning** Callout ("Source note") and recorded as VERIFY in `wan-background` |
| Lesson 1 | Some organizations backhaul on purpose (central inspection) | Hedged ("some organizations"), shown as a **tip** Callout, recorded as VERIFY in `wan-background` |
| Lesson 1, diagram 1 | The data center holds the only Internet exit; all branch Internet traffic funnels through it; the exit link must be sized for everyone | These are properties of the *simplified diagram* and are introduced as such ("In this simplified design"); backhauling itself is VERIFIED |
| Lesson 1 | "For a long time the WAN had one dominant job" | Paraphrase of the brief's "originally designed for applications in private, on-premises data centers" (VERIFIED) |
| Lesson 2 | Table column "What hurts today" (agility, branch rollout) | Rewritten to wording the brief supports (architectural complexity, long lead times); "What SD-WAN promises" is labelled the vendor's view |
| Lesson 2 | "Underlay" and "transport independence" | Flagged as course vocabulary in a **tip** Callout; VERIFY in `underlay-overlay` and `sdwan-concepts` |
| Lesson 3 | VCE / VCG abbreviations | **Warning** Callout; VERIFY in `components` |
| Lesson 3 | Management / control / data planes | **Gotcha** Callout says it is a teaching device and not product wording; VERIFY in `components` |
| Lesson 3 | "Links are the underlay and paths are the overlay" | Softened to "roughly speaking" (the guide describes the Links and Paths tabs but does not map them to underlay and overlay) |
| Lesson 3 | Inbound firewall rules for hubs and NAT | Not stated as fact; the Callout says they are not fully confirmed (VERIFY in `vcmp-tunnels`) |
| Lesson 4 | Day-level dates for the VMware and Broadcom steps; older product names | **Warning** Callout; the table has a "How sure are we?" column; SECONDARY / VERIFY rows in `branding-history` |
| Lesson 4 | Whether 7.0.0 is GA or LTS | Stated openly as not confirmed (VERIFY in `release-target`) |
| Lesson 5 | Menu paths for Segments, Network Services, Link Insights, user management, licensing; "Diagnostics" vs "Test & Troubleshoot"; the new vs classic UI | **Warning** Callouts; VERIFY in `vco-ui-map`; none of these appears in a quiz answer |
| Glossary | MPLS definition ("carrier-run private service"), VCE / VCG abbreviations, the three plane terms | Marked as course vocabulary or hedged in the entry text |
| Quiz | all 15 questions | Correct answers use only VERIFIED rows (checked row by row); no quiz answer depends on a VERIFY or SECONDARY row |


## Section 2 claim audit (self-review, step 6)

Every Section 2 lesson, diagram, glossary entry and quiz answer was re-read against the fact sheets. Sentences that make a technical claim **without** a VERIFIED row, and what was done:

| where | sentence or idea | resolution |
|---|---|---|
| Lesson 1 | "Nearly every configuration question about an Edge is either LAN-side or WAN-side" | Teaching generalisation, not a documented claim; presented in a **tip** Callout as a way to think, not as product behaviour |
| Lesson 1 | Whether an Edge has a local web UI | Not claimed; **warning** Callout says it was not confirmed (VERIFY in `edge-details`) |
| Lesson 1 | What an Edge does when the Orchestrator is unreachable | Not claimed; **gotcha** Callout says the pages do not state it (VERIFY in `edge-details`) |
| Lesson 1 and 6 | An Edge that stops sending heartbeats "shows as offline" in a specific screen | Reworded to "is marked OFFLINE by the Orchestrator" (the documented behaviour); screen location left open |
| Lesson 1 diagram | After activation the Edge downloads configuration and software | Narration now says this is described for the email (pull) method |
| Lesson 2 | Edge forwarding while the Orchestrator is unreachable | Shown as "Not stated" in the table and in a **warning** Callout; no quiz answer depends on it |
| Lesson 2 | Where the DR "Replication" view is for a hosted Orchestrator | Hedged ("we read this path in the on-premises guide; the hosted service may differ") |
| Lesson 3 | A full mesh of tunnels grows with every pair of sites | Marked as simple arithmetic in parentheses; not attributed to Arista |
| Lesson 3 | Exact placement and number of cloud Gateways | **Warning** Callout; VERIFY in `gateway-details` |
| Lesson 3 | An Edge's Paths tab "includes its Gateways" | Softened to "the peers can be other Edges and Gateways" |
| Lesson 4 | "Underlay" is course vocabulary; underlay "owned by carriers and the site's network team" | **Tip** Callout labels the word as ours; the ownership row is general industry knowledge (no row), kept in a neutral table cell |
| Lesson 4 | Underlay problems are a far more common cause than SD-WAN faults | Removed: reworded to "a place to check before assuming an SD-WAN fault" |
| Lesson 5 | An Edge keeps its Gateway tunnels permanently; a direct tunnel "shows up" in the Paths tab | Softened ("built by each Edge"; "should appear among the peers") |
| Lesson 5 | "The Orchestrator is not in the data path" | Removed (the exact phrase was not found); replaced by "described as the management plane" |
| Lesson 5 | When no direct tunnel can be built, traffic keeps using the Gateway or Hub path | Hedged in a **gotcha** Callout that says the cases are not fully walked through; VERIFY in `tunnel-types` |
| Lesson 6 | Hub inbound UDP 2426, DNS (UDP 53) and NTP (UDP 123) | Shown in the table as "Check your release docs" and in a **warning** Callout; VERIFY in `ports-protocols`; no quiz answer uses them |
| Lesson 6 | How ESP and the always-on NAT traversal port 2426 coexist | **Gotcha** Callout says it is not spelled out; VERIFY in `ports-protocols` |
| Lesson 7 | "Internet Multipath traffic is generally unencrypted" | Removed: this came only from a search summary of the page, never read directly |
| Lesson 7 | Padding varies "depending on the packet" | Reworded to "the padding field varies from 0 to 15 bytes" |
| Lesson 7 | MSS = MTU minus 40 for plain IPv4 and TCP; symptoms of MTU problems; a large-packet ping test | General networking knowledge, labelled as such in the lesson and the glossary; the ping test is hedged ("check whether your release lets you choose the packet size"); no quiz answer uses these |
| Lesson 7 | Where the chosen MTU is shown in the Orchestrator | **Warning** Callout: not confirmed |
| Glossary | NAT, MSS, fragmentation (general definitions) | Marked as general knowledge where the text says so; no quiz answer depends on them |
| Quiz | all 18 questions | Correct answers use only VERIFIED rows; the two MTU answers are arithmetic on VERIFIED overhead figures; one distractor explanation that quoted "not in the data path" was reworded |


## Step 8: configuration-model verification pass

New sheets: `segments`, `profiles`, `edge-overrides`, `business-policy`, `firewall`, `applications-qos`, `vco-config-ui`. No `reference-sources/` folder exists, so nothing was read from a local PDF except the already-extracted 7.0 Orchestrator Guide (page 36: segment limits), Troubleshooting Guide (Segment selectors) and Design Guide (EFS licence). Everything else came from Arista HTML pages through WebFetch with verbatim quotes requested (12 page fetches: 7.0 segments, profiles, Edge device settings (three offsets), profile device settings, business policy, firewall overview, overview, Appendix (two offsets), Provision a New Edge; 6.4 business policy, firewall overview, profile device settings). Two web-search summaries were used only as hints and are VERIFY or SECONDARY.

Per-release: the 6.4 and 7.0 pages read agree on Change Profile Segments, the lock symbol, "Edge override rules take priority", "Profile rules cannot be updated at the Edge", Interface and WAN Link steering at Edge level only, and the firewall actions. The Appendix tables of restarts list releases up to 6.0.x in the sample read.

| item | status | notes |
|---|---|---|
| How the VCO marks an overridden setting; how to revert | VERIFY | Not described on any Edge device-settings page read (about 300 of 320 thousand characters of the Edge page were read in three windows). |
| Complete list of overridable settings | VERIFY | Only per-setting statements. |
| Rule evaluation order, ties, default action (business policy and firewall) | VERIFY | Pages say "highest precedence" only. |
| Firewall versus business policy order | VERIFY | Not stated; diagram is labelled illustrative. |
| Segment-to-segment traffic; general firewall rules per segment; per-Edge-model segment limits | VERIFY | Not stated. |
| What changes when an Edge switches Profile (general effect on overrides) | VERIFY | Only VLAN, interface and BGP-filter side effects documented. |
| Quick Start Profile contents | VERIFY | Existence is VERIFIED; contents are not. |
| Applications screen path, application map, DPI packet count | VERIFY | The Applications page returned HTTP 403; DPI timing is only in a search summary. |
| Whether later Profile edits reach an Edge that overrides that setting | VERIFY | Search summary only. |
| Overlay Flow Control, Network Services | not examined | Step 9 scope. |

## Step 9: overlay connectivity and network services verification pass

New sheets: `cloud-vpn`, `hubs-and-branches`, `non-sdwan-destinations`, `network-services-other`, `overlay-paths`, `vco-overlay-ui`. New folder `observations/` with the status **OBSERVED** (see its README; the validator accepts it only with the source "user observation YYYY-MM-DD"). The folder starts with a template only. In the sheets, VERIFY rows that a real VCO could settle end with "[needs observation]".

Reading: no `reference-sources/` folder exists. Read directly: the 7.0 Troubleshooting Guide (List Paths, VPN Test, MTR, Gateway test). Through WebFetch with verbatim quotes requested (12 calls on 5 pages): 7.0 Profile device settings (three windows), 7.0 Network Services (four windows), 7.0 Edge device settings (two windows), 7.0 overview (one window), 6.4 Site Configurations Overview. Three web-search summaries were used only to find pages and are not cited as sources.

Per release: 6.4 and 7.0 agree on Cloud VPN being enabled under the Profile's VPN Services and on the Cloud Gateways and Hubs for VPN choices. 6.4 additionally says Hubs for VPN deactivates the Gateway data plane for Branch to Branch VPN and orders hubs by list position; the 7.0 pages read describe Hub order as a route tie-breaker and the dynamic first packet as "via the Cloud Gateway (or the Hub)". The 6.4 page read does not state the 180-second teardown; the 7.0 Profile page does.

| item | status now | evidence |
|---|---|---|
| Dynamic-tunnel trigger conditions | **Partly resolved** | 7.0: first packet via the Gateway or Hub; the initiating Edge builds the direct tunnel when it determines a secure overlay multi-path tunnel is possible; 180 s of silence tears it down. Further conditions not found: needs observation. |
| Edge-as-Hub inbound UDP 2426 | **Still VERIFY** | Searched the 7.0 overview, Edge and Profile pages: no statement; the only source stays a Broadcom KB we cannot open. Needs observation (packet capture or Hub firewall counters in a lab). |
| ESP with NAT-T on UDP 2426 | **Still VERIFY** | Only "VeloCloud forces NAT-T to be true with a special NAT-T port of 2426" and the ESP overhead description were found; nothing on when ESP appears on the wire. Needs observation (capture). |
| Edge behaviour when a Hub is unreachable | **Resolved (narrowly)** | 7.0 Network Services page: the Spoke re-points local flows to the secondary Hub; peer flows are not moved back when the primary returns. |
| Edge behaviour when the Orchestrator is unreachable | **Still VERIFY** | Searched all readable pages and local guides again: no statement. Needs observation (disconnect an Edge from the Orchestrator in a lab and watch forwarding). |

### Open VERIFY items that need a real-VCO observation

1. Edge-as-Hub inbound ports (UDP 2426 and others).
2. ESP versus NAT-T on the wire.
3. Edge forwarding with the Orchestrator unreachable.
4. Further dynamic-tunnel build conditions; whether a flow prefers a direct tunnel over the Hub path; what happens to in-flight flows at teardown.
5. Cloud VPN default state and what an Edge override can enable.
6. Tunnel-event names in Monitor > Events; where the Cluster Score is shown.
7. Step 8 items: override marking and reverting, rule evaluation order, firewall versus business policy order, Applications screen path.

## Section 5 claim audit (self-review, step 9)

Sentences making a technical claim without a VERIFIED or OBSERVED row, per lesson. All 19 quiz correct answers map to VERIFIED rows. Diagram `ov-why-hub` is a suggested walk (not a documented procedure) and says so.

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Glance | 3 | (a) "They are not mutually exclusive": inference, hedged by "can". (b) The five-question framework: our method. (c) "Without Cloud VPN there are no branch-to-branch or branch-to-hub tunnels": verified only for branch-to-hub in 6.4. |
| 2 Cloud VPN | 2 | (a) Diagram note that no direct tunnel is needed for the Gateway path: inference. (b) The "who does what" split: general, consistent with Section 2. |
| 3 Hubs | 3 | (a) "a branch always has a ready path": mild inference from "permanent". (b) "branches do not need tunnels to each other": inference. (c) The UDP 2426 expectation: in a warning Callout as unconfirmed. "Everything else in the Profile becomes a spoke" was removed. |
| 4 Dynamic | 2 | (a) "It is simply absent while it does not exist": inference. (b) The expected screen sequence over time: stated as derived from documented behaviour, not a recording. |
| 5 NSD | 2 | (a) "What the branch needs: its own WAN": inference. (b) The via-Edge "Edge-level" wording was corrected to Network Services plus Profile enablement. |
| 6 Services | 2 | (a) Problem statements for DNS and others paraphrase one-line purposes. (b) Edge-level assignment is documented for TACACS only; stated so. |
| 7 Diagnosis | 2 | (a) The order of checks: our method. (b) "If a Dynamic entry exists, the direct path exists": inference from Mode. |
| **Total** | **16** | none appears as a quiz correct answer |

## Section 4 claim audit (self-review, step 8)

Sentences making a technical claim without a VERIFIED row, per lesson, and what was done. All 19 quiz correct answers map to VERIFIED rows. The diagram `cfg-fw-vs-bp` is titled illustrative because the order is unknown; `cfg-bp-classify` rules are invented and labelled.

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Model | 2 | (a) "Segments and network services are inputs, not a layer that can override": inference from "composite", in an info Callout. (b) The four-question method: our own procedure, labelled a method. |
| 2 Segments | 2 | (a) Examples of corp, guest and IoT segments: labelled examples. (b) "The Edge treats each segment's traffic by that segment's own configuration" (diagram): inference from per-segment settings. The isolation caveat is in a warning Callout. |
| 3 Profiles | 3 | (a) "No drift between Edges" benefit: inference. (b) "Treat a Profile switch as a change to test": advice. (c) Diagram step that an overriding Edge keeps its own value: inference from "can be overridden for a specific Edge"; recorded as simulator assumption 5. |
| 4 Overrides | 3 | (a) When to override and when not: our advice, now stated as such (the earlier "Arista material says" wording was removed). (b) The drift concept: our term. (c) "The Edge no longer follows the Profile for that setting": inference. |
| 5 Business policy | 2 | (a) "Smart Defaults means traffic is treated sensibly before you write a rule": mild inference. (b) Diagram rules: invented, labelled illustrative. |
| 6 Firewall | 3 | (a) "Permission, not treatment" framing: the page does not compare the two (VERIFY row). (b) "Both exist at both levels so they get confused": opinion. (c) The last diagram step implies a sequence: the narration says the order is unknown. |
| 7 Apps and QoS | 4 | (a) "That is what lets a rule match an application instead of ports": inference. (b) What scheduler weights and policers do: inferred from their names. (c) "QoS treats traffic on a link; steering chooses the link; QoS adds no bandwidth": general. (d) "Priority and weights decide who wins when busy": inference. |
| 8 Diagnosis | 3 | (a) The step order: our procedure. (b) "Direct bypasses the DMPO tunnel": legacy wording, now hedged. (c) "A brownout may be why MPLS was chosen" under Auto: general reading of Section 3. |
| **Total** | **22** | none appears as a quiz correct answer |

## Section 3 claim audit (self-review, step 7)

Every Section 3 lesson, diagram, glossary entry and quiz answer was re-read against `dmpo*.md`. Below, each sentence making a technical claim **without** a VERIFIED row, per lesson (count in brackets), and what was done.
Quiz: all 19 correct answers map to VERIFIED rows (checked row by row); no correct answer uses a SECONDARY or VERIFY row or legacy white-paper text.

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Overview | 4 | (a) "A traditional router picks a path from a routing table and mostly ignores current quality" and (b) "Internet fast but wandering, MPLS steadier but dearer": teaching generalisations, used as framing only, not quiz answers. (c) "does not make a bad link good", "cannot create bandwidth", "follows the policy": inference from the documented mechanisms, stated as consequences. (d) Per-packet steering and load balancing: labelled as older VeloCloud material in the lesson and glossary. |
| 2 Monitoring | 3 | (a) "That is why DMPO can move traffic to an idle link; it already has numbers": inference from the 100 ms probes. (b) "A link can look fine in one direction and poor in the other": inference from one-way measurements. (c) The worked table: labelled "illustrative numbers"; the interpretations (steady MPLS, noisy Internet) are exercise reading, not claims. "No extra packets needed" was removed. |
| 3 Steering | 3 | (a) Transport group "e.g. Public Wired": the Links view uses those names; that the rule editor uses the same names was not confirmed, so the wording is "for example". (b) "Choose Mandatory only for traffic you would rather lose": design advice, labelled as such. (c) Preferred on MPLS with failover to the Internet: the documented text is about moving to another public link; a **warning** Callout says the private-link case should be tested. "Use it when moving is disruptive" was removed. |
| 4 Outages and brownouts | 4 | (a) "A brownout is gradual; a link that drops one packet in a few dozen looks alive to an interface counter": general; the routing-protocol sentence was removed. (b) "The hold stops a flapping link dragging a call back and forth": purpose inferred from a one-minute hold. (c) "If traffic does not come back, it can be the hold": hedged. (d) "Monitor > Events records link events": the menu and the event were read in different pages; flagged as a pointer. |
| 5 Remediation | 4 | (a) FEC cost is extra bandwidth: inference from "packet duplication"; no percentage (VERIFY in the sheet). (b) Jitter-buffer cost is a bounded delay: labelled "our reading". (c) "Textbook FEC means extra coded data": general knowledge in a **warning** Callout. (d) NACK "protects the end hosts from seeing the loss": SECONDARY wording; "TCP does not slow down" removed. Worked examples labelled illustrative. |
| 6 Together | 3 | (a) Scenario 2 aggregation and "about 100 Mbps": legacy 2017 claim, in a **warning** Callout. (b) "Remediation applies to this flow the same way": inference. (c) The "which mechanism is acting" table: our own heuristic, now stated as such in the lesson. |
| 7 Reading in the VCO | 3 | (a) "Choppy audio is usually loss or jitter, sometimes latency": general. (b) The "reading the answer" table: heuristic, the **warning** Callout says the VCO does not confirm a mechanism. (c) Step order is a suggested method, not Arista's procedure. |
| **Total** | **24** | none appears in a quiz correct answer |


## Step 10: routing verification pass

New sheets: `routing-overview`, `static-routes`, `ospf`, `bgp`, `route-redistribution-and-precedence` (with the third-party claim table), `overlay-flow-control`, `vco-routing-ui`. No `reference-sources/` folder exists.

Reading: **local PDF read directly** (extracted by us from the 7.0 Troubleshooting Guide, printed pages 4, 36-47, 58-65, 80-85: Remote Diagnostics routing tests and the Route Table Dump reason codes). **Through WebFetch with verbatim quotes requested** (the tool caps quotes near 125 characters, so some rows are its paraphrase and say so): 7.0 and 6.4 "Configure Dynamic Routing with OSPF or BGP" (nine windows, 0 to 100000 of 156444 and 146788 characters, plus 100000 onward once), 7.0 Overview (four reads, 0 to 100000 of 153454), 7.0 Edge device settings (three windows, 0 to 300000 of 321367), 7.0 Profile device settings (three windows, 0 to 270251), 7.0 Monitor Enterprise (two windows). **Failed**: the dedicated 7.0 and 6.4 Overlay Flow Control pages and a 7.0 Monitor Routing page returned HTTP 403 (not circumvented); a guessed print address returned 404. Counts: 1 local PDF; 20 WebFetch reads on 6 pages; 3 failed fetches; 4 search-result summaries used only to find pages.

Part A re-reads (7.0 Profile page): **"By default, the Orchestrator configures it for all Edges"** is quoted for Dynamic Branch to Branch VPN, so the default statement is VERIFIED and kept, with its source named; the Cloud VPN default itself is still not stated. **"The Hub is an Edge deployed in Data Centers for branches to access Data Center resources"**; no eligibility rule is listed, so lessons say a Hub is a designated Edge, typically in a data center.

| item | status now | evidence |
|---|---|---|
| Administrative distance | **Not used by Arista** | Pages use an order of operations and numeric route preference (lower wins). |
| OE1 and OE2 preference numbers (printed 1001006 and 1001007) | VERIFY | Looks inconsistent with the other values in the same table; lessons say "much larger". |
| Default states: Overlay Prefix, Overlay Prefixes, Connected Routes, OSPF and BGP into each other | VERIFY [needs observation] | Switches are described; defaults are not stated. |
| Default action and evaluation order of BGP filters | VERIFY [needs observation] | Not stated. |
| Setting that suppresses summarized and supernet routes | VERIFY [needs observation] | Name and default not captured. |
| Whether static routes and routing tables are per segment, and cross-segment routes | VERIFY [needs observation] | Static Route table has no Segment field; diagnostics are per segment. |
| OFC page details (flag defaults beyond two, Eligible VPN Exits, access roles) | VERIFY [needs observation] | Page unreadable (HTTP 403). |
| Monitor > Routing tab columns and Gateway Route Table | VERIFY [needs observation] | Tab names read, columns not. |
| OSPF interface settings (timers, cost, MTU, priority), limits | VERIFY [needs observation] | Not in the text received. |
| Dynamic B2B and Cloud VPN defaults | Dynamic B2B VERIFIED (all Edges by default); Cloud VPN default VERIFY | 7.0 Profile page re-read. |

### Open VERIFY items that need a real-VCO observation (step 10)

1. Defaults of the five redistribution switches (Overlay Prefix, Overlay Prefixes, Connected Routes, OSPF into BGP, BGP into OSPF).
2. BGP filter default action for unmatched routes and rule evaluation order.
3. OE1, OE2 and uplink preference values in the Route Table Dump.
4. Whether static routes and routes are per segment, and how routes cross segments.
5. Name and default of the summarized and supernet route suppression setting.
6. Overlay Flow Control: remaining flag defaults, meaning of Eligible VPN Exits, effect of a pinned route on other Edges.
7. Monitor > Routing columns and the Gateway Route Table.

## Section 6 claim audit (self-review, step 10)

Sentences making a technical claim without a VERIFIED or OBSERVED row, per lesson. All 20 quiz correct answers map to VERIFIED rows. Diagrams with prefixes or numbers are titled illustrative; `rt-diag-flow` is our method and says so.

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Big picture | 3 | (a) The three-stage model is our teaching frame built from verified steps. (b) "Each stage is a separate switch": inference from the redistribution rows. (c) A remote Edge "if you enable it" redistributes to its LAN: inferred from the overlay-to-protocol check boxes. |
| 2 Static | 5 | (a) "Typical use: behind a LAN router with no dynamic protocol": our example. (b) "Preferred everywhere can pull traffic away from the overlay": inference. (c) Two static routes as primary and backup (tip Callout): inference from the metrics sentence. (d) "Remote sites see one exit": inference. (e) The Predict scenario (Preferred without Advertise) follows from the two rows. |
| 3 OSPF | 3 | (a) A passive interface advertises its subnet but forms no neighbors: general OSPF meaning. (b) Summarization definition and the /23 example: general networking. (c) Edge-to-LAN advertised content paraphrases the Profile settings. |
| 4 BGP | 4 | (a) "A network statement names a prefix; redistribution names a source": general framing. (b) Check Max-hop and Local IP for multi-hop eBGP: general. (c) The neighbor-down order: our method. (d) "Check the AS numbers match": general. |
| 5 Redistribution | 3 | (a) The "five boxes" framing. (b) Loop theory and "tag routes" advice: in a warning Callout as general practice. (c) "OE1 and OE2 lose to BGP": supported by the reason code, the numbers are VERIFY and not shown. |
| 6 Filters | 4 | (a) What a route map is (general). (b) Standard BGP attribute order: stated as an assumption. (c) The worked example's step 1 consequence and its premises: illustrative, with a warning Callout. (d) That local preference reaches other Edges is explicitly not claimed. |
| 7 OFC | 3 | (a) "When to use it": advice. (b) OFC does not override longest prefix, connected or local static: inference from the order of operations. (c) The record-before-edit habit: advice (tip Callout). |
| 8 Diagnosis | 2 | (a) The order of checks: our method, in a warning Callout. (b) Which setting to change at each stage: pointer to the documented screens. |
| **Total** | **27** | none is a quiz correct answer |

`docs/SIMULATOR-MODEL.md` additions: three statements without VERIFIED rows (OE1/OE2 order, Gateway reflection scoping, example topology values), all in assumptions 21 and 32 or labelled illustrative.
