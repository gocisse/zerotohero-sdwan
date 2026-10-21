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
| `edge-models.md`, `site-topologies.md`, `bronze-silver-gold.md`, `edge-ha.md`, `vco-edge-models-ui.md` | step 17: Edge models and sizing, branch site topologies, the Bronze/Silver/Gold terms, Edge HA, where the model, interfaces and HA appear in the VCO |
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

## Step 12: simulator tunnels and paths (no new reading)

No new pages were read. The overlay engine cites existing rows of `cloud-vpn`, `hubs-and-branches`, `tunnel-types`, `non-sdwan-destinations`, `overlay-paths`, `gateway-details`, `mtu-overhead`, `dmpo-steering`, `dmpo-monitoring` and `vco-overlay-ui` (a test checks every cited row exists and is VERIFIED or OBSERVED, and every not-modelled marker cites a VERIFY row). New VERIFY rows written to record what the engine must not guess: whether a Spoke moves local flows back to the primary Hub and how long it takes to treat a Hub as down; a Hub Edge behind a NAT; the effective MTU of tunnels to Non SD-WAN destinations; Preferred steering on a private link and how DMPO picks among equally healthy paths; the loss, latency and jitter values that make a link degraded; whether an Edge's Cloud VPN override replaces a whole sub-section. The OSPF override conflict is recorded in `edge-overrides` and `ospf` (decision 83).

### Open VERIFY items that need a real-VCO observation (step 12)

1. Defaults: Cloud VPN on or off; whether an Edge override can enable a tunnel type its Profile disables; whether an override replaces a sub-section.
2. The conditions of "a secure overlay multi-path tunnel is possible"; Branch to Branch enabled at only one of two sites.
3. What happens to in-flight flows at teardown; whether a flow prefers an existing direct tunnel over the Hub path.
4. Hub failover: how long a Spoke takes to treat a Hub as down; whether local flows return to the primary Hub.
5. A Hub Edge behind a NAT; Edge behaviour with the Orchestrator unreachable.
6. Preferred steering on a private link; the numbers behind a degraded link; load balancing.
7. The MTU of tunnels to Non SD-WAN destinations.
8. Whether the Edge overrides OSPF settings (the two 7.0 pages disagree).

### Step 12 claim audit (self-review)

Every Section 2, 3 and 5 lesson was read in full against the engine. Lesson claims the engine cannot support (the engine has no routes, firewall logs, VPN Test, Flows tab or measurements): route reflection by the Gateway (step 13), "...-via-Hub" firewall log names and VPN Test (step 14), per-link measurements on the Links tab. Engine behaviours the lessons do not state, all shown as assumptions: Gateway derivation by region (44), Hub order deciding between Hubs while routing attributes are not modelled (59, lesson 5.3 says tie-breaker after attributes), hubs for VPN used for 7.0 as in 6.4 (61), the 180 s boundary (52), no hold time or remediation in a single decision (67, 68), VPN overhead for site-to-site flows (66). Mismatch fixed: lesson 4 of Section 4 and lesson 3 of Section 6 on OSPF overrides.

## Step 13: simulator routing and reachability (no new reading)

No new pages were read. The routing engine cites existing rows of `routing-overview`, `static-routes`, `ospf`, `bgp`, `route-redistribution-and-precedence`, `overlay-flow-control`, `overlay-paths` and `vco-routing-ui`; a test checks every cited row exists and is VERIFIED or OBSERVED and every not-modelled marker cites a VERIFY row. Five VERIFY rows were added to record what the engine must not guess: whether a route learned at this Edge beats a remote Edge connected or static route (order of operations against LR_ROUTED_VS_VELO), how the transit metric enters selection, the Logical ID tie-break direction and route age, what a passive OSPF interface does, and the local preference default and attribute handling across the overlay.

### Open VERIFY items that need a real-VCO observation (step 13)

1. The defaults of Overlay Prefix, Overlay Prefixes, Connected Routes, OSPF into BGP and BGP into OSPF, and of the static and connected Global Advertise Flags.
2. The filter default action and evaluation order; the BGP local preference default.
3. Whether a routed route beats a remote Edge route; the transit metric; the Logical ID direction; route age.
4. What a passive OSPF interface does; connected and static routes into OSPF and BGP.
5. How routes cross segments; whether static routes and BGP are per segment or per Profile.

### Step 13 claim audit (self-review)

Every Section 6 lesson was read in full against the engine. Lesson sentences changed: lesson 2 and the glossary entry for the ICMP probe said a failed probe shows SLA_STATE_DOWN (the guide only lists the code; now hedged, and assumption 72 in the simulator); lesson 3 said a passive interface advertises the connected subnet and forms no neighbors (the guides read do not say; now hedged, assumption 73). Gaps the engine has against lesson statements, none contradicted: static routes redistributed into BGP or OSPF on the owning Edge, the Networks field and Uplink flag of BGP, OSPF route filters, BGP community, AS-path regular expressions and As-Path-Exclude, the internal and uplink Global Advertise Flags, a flag taking effect only for routes learned afterwards (the engine shows the steady state), BGP per segment with a Profile default (authored per Edge), and the Gateway as the route reflector (the engine derives remote routes from advertisement and Cloud VPN settings, so a Gateway outage makes routes unreachable instead of absent).

## Step 14: flow tracer and labs (no new reading)

No new pages were read. The tracer and the labs cite existing rows of `business-policy`, `firewall`, `dmpo-remediation`, `mtu-overhead`, `overlay-paths` and `routing-overview`; a test checks every cited row exists and is VERIFIED or OBSERVED and every not-modelled marker cites a VERIFY row. Two VERIFY rows were added: what happens to learned routes when the reflecting Gateway is unreachable (and which Gateway reflects), and return traffic from a destination outside the SD-WAN.

### Open VERIFY items that need a real-VCO observation (step 14)

1. The business-policy default when no rule matches; business rule evaluation order beyond "highest precedence".
2. Firewall rule evaluation order, the default action and whether the firewall decides before or after business policy and routing.
3. Whether remediation is active for a flow at a given moment, and the loss thresholds.
4. Return traffic from outside the SD-WAN; the effective MTU of Non SD-WAN tunnels.
5. How the Gateway reflects routes when it is unreachable, and between primary and secondary Gateways.

### Step 14 claim audit (self-review of the labs)

Every lab text was read against the lessons and fact sheets; see the step report for the list. Labs may not rest on any VERIFY row: the validator rejects an answer that depends on a not-modelled result.

## Step 17: Edge models and branch topologies verification pass

New sheets: `edge-models`, `site-topologies`, `bronze-silver-gold`, `edge-ha`, `vco-edge-models-ui`. No `reference-sources/` folder exists, so nothing was read from a local PDF in that folder; the PDFs below were downloaded by us and their text extracted.

Reading: **read directly (3 PDFs, text extracted by us)**: the Arista Edge 7x0 datasheet (12 pages, "Oct 16th 2026", 04-0063-07), the Edge 4100/5100 datasheet (9 pages, 04-0062-06) and the VMware SD-WAN 3.4 Administration Guide (483 pages, Broadcom legacy archive, 2020; pages 14 to 18, 29 to 33, 266 to 267, 332 to 334). **Through WebFetch with verbatim quotes requested (12 reads on 8 pages)**: 7.0 and 6.4 "Configure High Availability on an Edge", 7.0 "Configure Device Settings for Edges" (three windows, 0 to 200000 of 321366 characters), 7.0 Overview (three reads, 0 to 153454 characters), 7.0 and 5.2 Site Configurations, the Edge 1RU quick start front panel. **Failed**: the 7.0 "Configure VRRP Settings" page (HTTP 403, not circumvented); the old VMware HTML pages redirect to Broadcom's home page. **Search summaries used only to find pages**: 8, plus retailer listings (CDW) for the Edge 840, recorded as VERIFY.

| item | status now | evidence |
|---|---|---|
| Are Bronze, Silver and Gold vendor terms? | **Yes, documented vendor terms** | 3.4 guide read directly (pages 29 to 33) and 7.0 Overview (summarised). The 7.0 Site Configurations page does not use them. See `bronze-silver-gold.md`. |
| Course outline: a large-branch model with eight interfaces | **Confirmed for the 720 and 740, corrected as a descriptor** | Both list 6 x 2.5G RJ-45 and 2 x SFP+; the 4100 and 5100, which Arista positions for large branches, list 18 and 16. The outline names no model. |
| Segment and flow limits | **Published per model** | 128 segments (720, 740, 4100, 5100, 840 and the 620 to 3810 group), 32 (710); concurrent flows 225K to 7.5 million. |
| VNF support per model | **Partly documented** | A firewall-VNF throughput table for eight older models (summarised); no per-model matrix for the 7x0, 4100 or 5100. VERIFY. |
| HSRP and VRRP for Edge deployments | **Documented in three places** | Required on the L3 switch pair in HA Deployment Type 2; named on the routers of Silver variant 2; an Edge-run "VRRP with 3rd party router" option (3.4 guidelines read directly; the 7.0 guidelines page was unreadable). |

### Open VERIFY items for topologies (step 17)

1. VNF support on the current Edges (710, 720, 740, 4100, 5100): the 7.0 Edge device settings page (step 18) lists only the 520v, 840, 620, 640 and 680.
2. Interfaces and limits of earlier Edge models (840 and others) from an official datasheet.
3. What the Edge does about HSRP or VRRP between the routers in Silver variant 2, whether the variant uses an HA pair, and what happens when a router, CE or switch fails.
4. Whether the 3.4 Edge-run VRRP guidelines hold in 6.4 and 7.0 (the 7.0 page returned 403).
5. Where the Gold virtual IP is configured, and where the Edge sits relative to the CE and the L3 switches.
6. Whether the older guide's two Silver lists (Branch Site Topologies and Site Configurations) name the same design; whether the 6.4 Overview has the Bronze, Silver and Gold section.
7. Where the VCO shows the Edge model, the interface list, and the HA pair's Standby; whether the Orchestrator warns about an HA pair of different models. These need a real-VCO observation: the lessons list them as `[observation candidate]`.

## Section 7 claim audit (self-review, step 17)

Sentences making a technical claim without a VERIFIED or OBSERVED row, per lesson. All are tagged in the text as (design practice) or general networking knowledge, or sit in a table row with the status "design practice" or "not documented". None is a quiz correct answer. After the audit three sentences were rewritten (the Bronze firewall wording, the Silver 1 "documented" lead-in, an inferred Gold interface claim).

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Edge models | 3 | (a) compare throughput with real traffic and keep headroom; (b) a Hub is sized by tunnels; (c) the six-step shortlist method. All tagged design practice. |
| 2 Overview | 2 | (a) the designer's five questions; (b) the "ask which variant" advice in the Predict. |
| 3 Bronze | 5 | (a) the LAN's gateway is "naturally" the Edge; (b) low equipment count; (c) one link means nothing to fail over to; (d) a single Edge is a single point of failure; (e) when Bronze fits. |
| 4 Silver 1 | 6 | (a) the two traffic flows (illustration); (b) the L3 switch is the users' gateway; (c) the CE and the Edge are two exits; (d) duplicate advertisement; (e) an L3 switch failure; (f) an Edge failure. |
| 5 Silver 2 | 3 | (a) an HSRP or VRRP router failing is handled by its standby router (general protocol behaviour); (b) HSRP or VRRP "runs on the switches, not the Edge" is an inference from the documented requirement on the L3 switch pair; (c) the placement of an HA pair in the variant (the lesson says the text does not show one). |
| 6 Gold | 4 | (a) failure of one active/active switch; (b) a routing session drop (general); (c) a Gold branch is not a Hub (Arista positions models, it does not define the branch); (d) sharing across switches (stated as not known). |
| 7 Choosing | 6 | the framework, the table's "Add" column, and the three scenarios' choices. The settings lists carry (documented) or are marked as not confirmed. |
| **Total** | **29** | |

Consistency check across Sections 1 to 7 and the glossary: "HA" appears in Section 5 (a Hub cluster provides Active/Active high availability) and in Section 7 (an Edge pair is Active/Standby); these are different mechanisms and the glossary now says so. "Gateway" means the VeloCloud Gateway in Sections 1 to 6 and Section 7 also uses "default gateway"; the two glossary terms say they are different. A Hub is "an Edge designated in the Profile, typically in a data center" in Section 5 and Section 7 keeps that wording. "Transport", "link" and "WAN overlay" are used as in Sections 2 and 5. No contradiction was found.

## Step 18: network services settings verification pass

New folder `config/` with ten sheets (the loader accepts it and applies the claims-table rule; sheet names keep their own file name, for example `config-dns`). No `reference-sources/` folder exists, so no local PDF was used for these; the one local PDF row reused is the 7.0 Troubleshooting Guide (MTR paths).

Reading: **through WebFetch with verbatim quotes requested (19 reads on 5 pages)**: 7.0 Network Services (three windows, 0 to 269017), 6.4 Configuring Network Services (three windows, 0 to 266968), 7.0 Profile device settings (three windows, 0 to 270251), 7.0 Edge device settings (windows 200000 to 321366), 7.0 dynamic routing page (100000 onward). **Failed (HTTP 403, not circumvented)**: the 6.4 `configure-network-services` address (the `configuring-network-services` address worked). The tool marks some lines as its own paraphrase; rows say so. Nothing was read directly from a PDF this step.

### Third-party claims from the course outline

| claim | result | source and evidence |
|---|---|---|
| RADIUS: server address, ports 1812 and 1813, shared secret, assignment under the Profile authentication settings | **Partly corrected** | Server Address, Shared Secret and Profile assignment (Device tab > Edge Services > Authentication) confirmed; Authentication Port default 1812 confirmed; 1813 is not named, the Accounting Port is optional with no default (7.0 and 6.4) |
| DNS: public versus private, a "DNS mask" or fastest-response behaviour, IPv4 and IPv6 options | **Confirmed except "DNS mask"** | DNS Type Private or Public; fastest response (dnsmasq asks all servers); IPv4 Server, optional IPv6 Server; no "DNS mask" in the pages read |
| Prefix delegation tags: a pair of WAN and LAN tags, DHCP prefix delegation, /64 | **Partly corrected** | One tag shared by a WAN interface (unique per WAN) and its LAN interfaces, not a pair; DHCPv6 prefix delegation; LAN prefix length auto-populates as 64 |
| NetFlow: collectors and filters with match criteria and actions, applied per Profile | **Confirmed, with corrections** | collectors (name, IP, port), filters (match source IP, destination IP or application; Allow or Deny); applied per Profile and per segment, with an Edge Override and Source Interface |
| Clusters: one logical unit, promotion to Hub, auto-rebalance managed by the Orchestrator | **Partly corrected** | a logical cluster seen as one Hub (confirmed); designated as a Hub in a Profile with Edit Hubs (no "promotion" step); Auto ReBalance is a per-cluster checkbox, off by default, moving Spokes when a Hub's score is above 70 (the Gateway moves them) |
| Private network names: tunnels only between interfaces with the same name | **Unconfirmed** | one Name field, assigned to private WAN overlays; only Conditional Backhaul says Branch and Hub use the same name |
| API credentials: tenant ID, client ID and secret for Azure and Zscaler | **Confirmed for Azure, corrected for Zscaler** | Zscaler uses Subscription Name, Zscaler Cloud, Partner Admin Username and Password, API Key and Domain |
| NSD via Gateway: AWS VPN gateway tunnels, site subnets, BGP; via Edge: IPsec IKEv2 router-based, subnets | **Confirmed** | AWS is one of eleven Gateway types; BGP over IPsec is eBGP only; the Edge variant lists IKEv1 and IKEv2 route-based and Azure Virtual WAN, with Site Subnets |
| VNF: firewall VNF (Fortinet), manager IP, credentials, image version, keys; documented per Edge model | **Confirmed** | Fortinet, Check Point and Palo Alto types with management address, credentials or keys, image version; the 7.0 Edge page lists the supported models (520v, 840, 620, 640, 680) |

### Fields documented versus unreadable, per service

| sheet | field rows (a row can name several fields) | rows still VERIFY (unreadable or unstated) |
|---|---|---|
| config-nsd-gateway | 44 | 3: per-type tables beyond AWS and IKEv2 (Check Point, Cisco ASA, Cisco ISR, Palo Alto, SonicWall, IKEv1, Generic Firewall), default Tunnel Mode and Enable Tunnel(s), the Cisco ASA table inconsistency |
| config-nsd-edge | 20 | 2: status or events, how Edge Add Tunnel fields relate to the service fields |
| config-api-credentials | 6 | 2: status or role information |
| config-clusters-hubs | 6 | 1: Cluster Score location, Spoke down time, interconnect menu path |
| config-netflow | 8 | 2: status or events, Allow All and Deny interaction |
| config-dns | 5 | 2: status, limits on servers |
| config-private-network-names | 1 | 3: the general tunnel rule, the assignment screen, status |
| config-prefix-delegation | 3 | 1: status or events |
| config-authentication | 7 | 1: status or events |
| config-vnf | 9 | 1: current-hardware support, licences, deploy time |

### Open VERIFY items for network services (step 18)

1. Field tables of the NSD via Gateway types other than AWS VPN Gateway and Generic IKEv2, and the Cisco ASA tunnel-mode inconsistency.
2. The screen where a Private Network Name is assigned and what it does to tunnels.
3. Status screens or events for NSD via Edge, NetFlow, DNS, RADIUS, TACACS, prefix delegation and Private Network Names.
4. Where the Cluster Score is shown, and the Profile menu path that turns on Hub or Cluster Interconnect.
5. VNF support on the 710, 720, 740, 4100 and 5100 and VNF licences for Check Point and Fortinet.
6. 6.4 Profile and Edge menu steps (only the 6.4 Network Services page was read).

## Section 8 claim audit (self-review, step 18)

Sentences making a technical claim without a VERIFIED row, per lesson; all are labelled "our method" or sit in a table row marked as not documented; none is a quiz correct answer. Every field name in a lesson table exists in a VERIFIED row (automated).

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Where | 2 | create, assign, verify is our method; the verification order |
| 2 NSD via Gateway | 1 | "route-based" and "policy-based" restate the documented type labels |
| 3 NSD via Edge | 1 | IKE and IPsec described in general terms (glossary marks IKE as general knowledge) |
| 4 Clusters | 0 | |
| 5 Credentials and NetFlow | 0 | the Predict distractor "2055" is a wrong option, not a claim |
| 6 Name and address | 1 | the Private Network Name effect on tunnels is stated as not documented |
| 7 Auth and VNF | 1 | the deploy order create, deploy, insert is read from the prerequisites of the Edge page |
| 8 Review | 4 | the reading order, the check order, "an Override ignores the Profile value" (Section 4), "a saved service does not prove traffic uses it" |
| **Total** | **10** | |

Consistency across Sections 1 to 8 and the glossary: "network service" has two meanings (the Network Services area and the business policy action), now explained in Section 8 lesson 1 and the glossary term; "Non SD-WAN Destination via Gateway" (service), "Edge to Non SD-WAN via Gateway" (Profile checkbox) and "NSD via Gateway" (shorthand) are used as documented; Hub, cluster and Cluster Score agree with Sections 5 and 7; the Section 5 services lesson said Edge-level assignment was documented for TACACS only, which is now reworded to point to Section 8; "tunnel" means the VCMP tunnel Edge to Gateway or the IPsec tunnel to a destination, with different diagram styles. No contradiction remains.

## Step 19: segments, Profiles and policy rules settings verification pass

New sheets in `config/`: `config-segments`, `config-profiles`, `config-business-policy-rules`, `config-firewall-rules`, `config-application-match` (same claims-table rule; field rows start `Field "<name>" (applied: ...)`). No `reference-sources/` folder exists; no PDF was read this step except the earlier local Orchestrator Guide row (segment limits, page 36) and the two datasheets read in step 17.

Reading, all through WebFetch with verbatim quotes requested (14 reads on 9 pages): 7.0 Configure Segments (whole article); 7.0 Configure Profiles (whole article); 7.0 Profile device settings (three windows, 0 to 270251, the whole page); 7.0 Configure Business Policy (whole article by the summary's account); 7.0 Firewall Overview (0 to 100000 of 137822; the last 37822 characters, from the Enhanced Firewall Services section on, were **not read**); 7.0 Configure Custom Applications (whole article); 6.4 Configuring Business Policy and 6.4 Firewall (0 to 100000 of 137657). **Failed**: 6.4 `configure-segments` (HTTP 404), 6.4 `configure-network-services` earlier, and the 7.0 application maps page (HTTP 403, not circumvented). The 6.4 Profile pages and the 6.4 custom applications page were not read.

### Third-party claims from the course outline

| claim | result | source and evidence |
|---|---|---|
| Segments: corp, guest, IoT and OT as examples | **Unconfirmed** | the segments page names guest traffic, voice media and PCI traffic; corp, IoT and OT are not named |
| A default Global segment | **Confirmed** | the Global Segment exists; its type is Regular or Private |
| "Create a test segment and map a VLAN" | **Confirmed as steps** | Configure > Segments > Add > Save Changes, then the VLAN dialog's Segment field; "test" is only a name |
| Apply a segment to the Quick Start and a test Profile | **Confirmed as a procedure** | Segment > Change Profile Segments for any Profile; the Quick Start contents are not documented |
| Profiles: create, duplicate, modify | **Confirmed** | New Profile, Duplicate or Copy Profile, Modify or the Overview tab EDIT |
| Wi-Fi, authentication, DNS and VLANs inside a Profile | **Confirmed** | Wi-Fi Radio and WLAN SSIDs, Edge Services > Authentication, Routing and NAT > DNS, Connectivity > VLAN |
| Routing, NAT, VPN and firewall settings customised per Profile | **Confirmed** | Routing and NAT (including LAN-Side NAT Rules), VPN Services, Profile Firewall tab and Common Criteria Firewall |
| Business policy rules with IPv4 match, domain names, protocols and an application rule | **Confirmed** | CIDR, mask or wildcard address match; Domain Name matches all or part of a name; Protocol GRE, ICMP, TCP, UDP; Application Define |
| Base and firewall policies assignable to Edges via Profiles | **Confirmed, "base policy" not a term** | Profile rules are pushed to Edges for both business policy and firewall |
| "QoS traffic classes" | **Confirmed as priority plus service class** | the Action tab has Priority and Service Class, not a field called traffic class |
| A "Netflix.com application rule" | **Partly confirmed** | a rule can match an application or a destination domain name; whether Netflix is in the application list is not stated |

### Fields documented versus unreadable, per sheet

| sheet | field rows (a row can name several fields) | rows still VERIFY |
|---|---|---|
| config-segments | 10 | 5: edit and rename, delete error text, Service VLAN format, segment-to-segment traffic, 6.4 segments page |
| config-profiles | 28 | 3: the OSPF override conflict, Profile change propagation, Copy Profile fields and Quick Start contents |
| config-business-policy-rules | 16 | 2: rule order beyond precedence and the no-match action, DSCP values and defaults |
| config-firewall-rules | 13 | 2: evaluation order and the no-match action, the unread page tail |
| config-application-match | 6 | 1: the Application Maps and Applications screens and domain fields of a custom application |

### Open VERIFY items for configuration walkthroughs (step 19)

1. Segment edit and rename, the Service VLAN format, the delete error text, any in-use column, inter-segment traffic.
2. The Copy Profile dialog fields, the Quick Start Profile contents, and what a Profile change does to Edges (push timing, restarts).
3. The OSPF override conflict between the Profile page and the Edge page (unresolved).
4. Rule evaluation order, the no-match action and the firewall versus business policy order, for both rule types.
5. The Application Maps and Applications screens (HTTP 403), and domain-name fields inside a custom application.
6. The unread tail of the 7.0 firewall page, and the 6.4 Profile and segments pages.

## Section 9 claim audit (self-review, step 19)

Sentences making a technical claim without a VERIFIED row, per lesson; each is labelled "our method", illustrative or marked not documented; none is a quiz correct answer. Every field name in a lesson table exists in a VERIFIED row (automated, now over 15 lessons).

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Segments | 1 | the comparison method (switch the Segment menu and read the same sections) is labelled our method |
| 2 Profiles | 1 | the illustrative diagram step where a Profile change reaches an Edge without an override; the overriding Edge's behaviour is stated as not documented |
| 3 Device | 1 | "Other Connectivity and Telemetry" mixes sections; the Override table says "not stated" where the pages are silent |
| 4 Business policy | 2 | the worked example (field values are illustrative, fields documented); "Auto is the default steering" restates the documented default |
| 5 Firewall | 2 | the worked example; the diagram's single verdict is illustrative and ends with the no-match action as not documented |
| 6 Application matching | 1 | when to use a domain name instead of an application is stated as not decided by the documentation |
| 7 Comparing | 6 | the seven-step walk and its table order (our method, labelled); the simulator link description |
| **Total** | **14** | |

Consistency across Sections 1 to 9 and the glossary: "override" means the Edge Override checkbox for Device settings and a superseding identical rule for business policy and firewall rules, as in Section 4; "business policy" and "firewall" keep their separate roles and the unknown order is stated in both; "application" means a DPI-identified application (default or custom) and is separate from a destination domain name; "Edge-level" settings are the Override checkboxes, Source Interface and Edge-only items (loopbacks, LAN interfaces and SSID of a VLAN); the Section 5 and Section 8 statements about segments and Cloud VPN agree. The OSPF override conflict (Profile page versus Edge page) is reported in Sections 4 and 9 and remains unresolved, not hidden. No contradiction was found.

## Step 20: troubleshooting verification pass

Six sheets in `reference/troubleshooting/`. There is no `reference-sources/` folder, so the PDFs were downloaded by the author from the URLs in the rows and read as text; "local PDF page N" is the PDF page index (the printed number is about 4 lower).

| source | how it was read | rows resting on it |
|---|---|---|
| 7.0 Troubleshooting Guide (104 pages, DOC-08770-01) | local PDF, read in full as text | 55 |
| 7.0 Orchestrator Guide (144 pages) | local PDF, searched and the relevant chapters read (1.4 DR, 1.6 diagnostics, 1.7 operations, certificates and PKI pages 141 to 143) | 21 |
| 7.0 Gateway Monitoring Guide (47 pages, DOC-08768-01) | local PDF, read in full as text | 12 |
| 7.0 Design Guide (29 pages) | local PDF, searched for debug.py and the CLI (no hit) | 0 |
| 7.0 Administration Guide appendix (events) | summarising tool (WebFetch, verbatim quotes requested); the first 100000 of about 127000 characters read, the rest not | 13 |
| 6.4 Administration Guide appendix (events) and 6.4 Remote Diagnostic Tests on Edges | summarising tool; the events page the same way (first 100000 of about 129000 characters), the tests page in both halves (titles and two limit sentences) | 4 |
| Broadcom knowledge-base article on Partner Gateway commands; Remote Diagnostics WebSocket API reference | search-tool summaries only; both pages unreadable (HTTP 404 and 503) | 2 (SECONDARY) |
| (no source found) | rows marked VERIFY | 19 |

Counts: 106 VERIFIED, 2 SECONDARY, 19 VERIFY, 0 OBSERVED (127 rows). Counted directly: 4 PDFs (88 rows). Through a summarising tool: 3 page reads (17 rows). Arista's pages returned HTTP 406 to a plain fetch (not circumvented).

### Verification of the engineer-tool claims

| claim | result |
|---|---|
| a script called debug.py on the Edge | **confirmed in part**: named in the Orchestrator Guide (pages 141 to 143) for the "Edge or Gateway command line", and referenced as "the debug.py--routes output" in the Troubleshooting Guide (page 88); the Gateway Monitoring Guide uses it at /opt/vc/bin/debug.py on the Gateway; its location on an Edge is unconfirmed |
| it can dump flows, paths, routes, ARP/NDP, VPN state, bandwidth and link statistics | **unconfirmed for an Edge**: only `--pki`, `--ike_sa`, `--ike_spd` (documented) and `--routes` (referred to) are named; the Gateway options (`--path`, `--handoff`, `--vrf`, `--list_edges`, `--icmp_monitor`, `--bgp_view_summary`, `--stale_nat_dump`, `--dpdk_ports_dump`) are not Edge options; the equivalent read-only reads exist as Remote Diagnostics tests (List Active Flows, List Paths, Route Table Dump, ARP Table Dump, IPv6 ND Table Dump, VPN Test, WAN Link Bandwidth Test, Interface Status) |
| it is run from the Edge shell | **unconfirmed**: the documents say "Edge or Gateway command line interface" but not how to reach the Edge's shell |
| Edge-side log files | **confirmed in part**: only `/var/log/mgd.log` is named for the Edge (certificate mismatch case); the Gateway's vc_procmon, mgd, gwd and natd logs are listed in the Gateway guide; there is no Edge log list |
| an Edge diagnostic bundle | **confirmed**: Diagnostics tab > Diagnostic Bundles > Request Diagnostic Bundle (Operators), "collection of all the configuration and logs from a specific Edge", with Core Limit; and Request PCAP Bundle (Operators, Standard Admins, Customer Support) |
| the Remote Diagnostics tests List Paths, VPN Test, MTR, route and flow dumps | **confirmed** in 7.0 (PDF pages 79, 91, 33, 84, 36); **corrected for 6.4**: the 6.4 test list has no "MY Traceroute" (MTR), only "Traceroute"; the other tests exist in 6.4 |
| packet capture | **confirmed**: the PCAP bundle above (filters IP1, IP2, IP1:Port1, IP2:Port2, Protocol; the HA standby interface from 5.2.0) |

### Open VERIFY items (step 20)

1. How to get a shell on an Edge, as which user, and where debug.py lives on an Edge; its full option list; the output formats of every option (observation candidates; see the capture list in `docs/OBSERVATION-PRIORITIES.md`).
2. Edge log files other than `/var/log/mgd.log`, and the Orchestrator's service and log list.
3. Hub, NTP and license events (the unread tail of the events appendix), event retention, the Events screen's filters.
4. Whether a branch raises an event when a Hub tunnel is lost (`VPN_TUNNEL_DOWN` is documented only for the tunnel between the Edge service and a VPN gateway).
5. Documented causes and fixes for link flapping, activation failures beyond a wrong link, NAT or firewall blocks of tunnels, license problems and DMPO symptoms.
6. The control that sends a new activation from an Edge's entry in Configure > Edges, and where the activation state is shown.
7. Whether a single "connected Edges" health figure exists in the VCO.
8. Whether Remote Diagnostics tests and debug.py share an implementation.

## Section 10 claim audit (self-review, step 20)

Sentences that make a technical claim without a VERIFIED or OBSERVED row, per lesson. Each one is labelled "our method", marked illustrative or hedged ("to confirm in your VCO"); none is a quiz correct answer except the questions whose prompt says "according to this course's method". Every name in a lesson's Settings reference table is in a VERIFIED row (automated: `TestSection10TablesExistInVerifiedSheetRows`), and the debug.py runbook lists an option as documented only with such a row (`TestDebugReferenceRunbookListsOnlyVerifiedOptions`).

| lesson | count | sentences and resolution |
|---|---|---|
| 1 Method | 2 | the nine steps and the scope-to-layer hint (our method, labelled) |
| 2 Layers | 4 | the eight-layer model, its order, the rules of thumb, "a missing route can look like a tunnel problem" (labelled as our model and reasoning) |
| 3 Tools | 1 | the order we suggest (our method) |
| 4 Edge CLI | 1 | the safe-capture habit (our method); the lesson says plainly what is not documented |
| 5 Gateway and Orchestrator | 2 | "many Edges at once: shared component first" (our method); restarting services is named as state-changing and out of scope |
| 6 Events | 2 | the timeline method (ours); time order does not prove cause (reasoning) |
| 7 Scenarios 1 | 4 | hypothesis trees (ours); the "no event at all" reading (marked not a documented event); the activation control location (hedged) |
| 8 Scenarios 2 | 4 | hypothesis trees (ours); the NAT cause rests on the documented unconfigured-NAT effect and a simulator assumption; fix locations hedged "for example" |
| 9 Scenarios 3 | 5 | hypothesis trees (ours); the MTU shape "small works, large stalls" labelled our reading and tested with sizes; remediation noted as not readable per flow; the interface MTU location hedged |
| 10 Escalation | 2 | the two templates and "when to escalate" (ours, labelled) |
| **Total** | **27** | |

Every command, option or output that appears in a lesson or runbook, with its source: `sudo tail -f /var/log/mgd.log`, `/etc/vc_private`, `/etc/vc_public` (Orchestrator Guide page 141); `--pki`, `--ike_sa`, `--ike_spd` (Orchestrator Guide pages 142 and 143); `--routes` (Troubleshooting Guide page 88); the Gateway options in the debug.py runbook (Gateway Monitoring Guide pages 17 to 34, labelled Gateway only); the Remote Diagnostics test names and fields (Troubleshooting Guide); event names (events appendix, summarising tool). No output format of debug.py is shown. The `text` blocks in the scenario lessons and the support-case template are labelled "schematic (illustrative, not real output)" and use only field names the documents print (Route Table Dump Table 8, List Paths Table 7, the events table columns); no command output is invented.

Consistency across Sections 1 to 10 and the glossary: the Monitor > Edges tab names, the Remote Diagnostics test names (List Paths, VPN Test, Route Table Dump, List Active Flows, MY Traceroute/MTR) and the event names are spelled as in Sections 5 and 6 and in the glossary ("Remote Diagnostics", "List Paths", "MTR (My Traceroute)", "Event"); the Diagnostics tab is the entry point everywhere. One difference is deliberate: MTR is documented from 7.0 in the pages read and is not in the 6.4 list, which the lesson and the glossary entry say. No contradiction was found.

