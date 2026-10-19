# DMPO: overview fact sheet (step 7 rewrite)

Hub sheet for DMPO. Detail lives in `dmpo-monitoring.md`, `dmpo-steering.md`, `dmpo-remediation.md`, `dmpo-load-balancing.md`
and `dmpo-vco-ui.md`. Sources and how they were read (2026-10-19):

- **Read directly (PDF text extracted by us):** 2017 VeloCloud DMPO white paper ("WP2017", legacy, vendor-authored, pre-Arista), VMware DMPO
  technology brief (mirror host, SECONDARY), Arista Cloud Gateway Advantages white paper (30 June 2025), 7.0 Troubleshooting Guide,
  7.0 Gateway Monitoring Guide, 6.4 Troubleshooting Guide, 6.4 Design Guide, Edge 7x0 datasheet, solution brief, 6.4 release notes.
- **Summarised via WebFetch (verbatim quotes requested):** the 7.0 and 6.4 Overview pages, 7.0 Configure Business Policy, 7.0 Monitor Enterprise,
  7.0 Appendix, 7.0 Operator Guide "Manage Customers", 5.2 "Enterprise-level UI changes". The Arista site serves a CAPTCHA to automated
  browsers (not circumvented).
- **Not readable:** the 7.0 and 6.4 Administration Guide, Operator Guide and Partner Guide PDFs (over the 10 MB fetch limit); Broadcom ports article.

Per-release convention: the "doc version" cell says which release text the row comes from. "7.0 = 6.4 text" means the Overview pages of both
releases carry the same wording. WP2017 rows are legacy and are never used as a quiz correct answer.

| claim | source URL | doc version | status |
|---|---|---|---|
| DMPO continuously measures loss, latency and jitter of every packet, unidirectionally, on every path between DMPO endpoints. (Summarised via WebFetch, verbatim quotes requested.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text | VERIFIED |
| DMPO is used between the components that forward data traffic: Edge to Gateway for cloud destinations, and Edge to Edge for branch-to-branch and branch-to-hub. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (read directly, p.3) | SECONDARY |
| Arista's overview says Gateways implement DMPO and an Edge can optionally connect to Gateways for cloud services. (Summarised via WebFetch.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Four functions are named in the overview: continuous monitoring, dynamic application steering, on-demand remediation, and overlay QoS. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text | VERIFIED |
| DMPO detects an SLA failure or outage within 300 to 500 ms and steers the flow to a healthy link. (Summarised via WebFetch.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text | VERIFIED |
| A single flow can be moved between tunnels mid-session without the flow noticing (WP2017 wording: "no impact to the flow"). | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.4) | SECONDARY |
| The Arista guides use "outage" for a link that fails completely and "brownout" for a link that is up but cannot meet the SLA. The word "blackout" appears only in the legacy 2017 white paper, where it means a completely failed link. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text | VERIFIED |
| WP2017 says "blackout" (link completely down) and "brownout"; the course uses "outage" and mentions "blackout" only as older/common industry wording. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.4) | SECONDARY |
| After a brownout or outage clears, DMPO waits one minute before moving traffic back to the preferred link, if the business policy names one. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text | VERIFIED |
| On-demand remediation (error correction) is enabled for the duration of WAN link issues, typically when steering is not possible (a single link, or several links degraded at once); the brief says it applies to network-sensitive priority applications. | https://www.arista.com/assets/data/pdf/SD-WAN-Solution-Overview.pdf | solution brief (read directly) | VERIFIED |
| Premium Service is on by default for a customer and means the On-Demand Remediation feature: with it, the Gateway applies FEC to impaired traffic that cannot be steered; without it, continuous monitoring, dynamic steering and secure transmission still apply but no Gateway error correction. (Summarised via WebFetch.) | https://www.arista.com/en/operator-guide-vc-7-0/sd-wan-7-0-manage-customers | 7.0 | VERIFIED |
| Arista runs cloud Gateways in more than 30 regions with more than 3000 Gateways, and assigns each Edge a primary and a secondary Gateway by geolocation. (Read directly; figures are as of the 30 June 2025 white paper.) | https://www.arista.com/assets/data/pdf/VeloCloud-SD-WAN-Cloud-Gateway-Advantages.pdf | white paper, 30 Jun 2025 | VERIFIED |
| The VMware DMPO technology brief lists the same four capabilities (monitoring, steering, remediation, QoS) and the same FEC, jitter buffer and NACK descriptions. | https://www.perfect-packet.com/wp-content/uploads/2024/07/vmware-dynamic-multi-path-optimization.pdf | VMware brief (read directly, mirror) | SECONDARY |
| Per-application SLA threshold values (the exact loss, latency and jitter limits per application class) exist in the Administration Guide only as an image (Figure 26 area); the text of the table could not be read. | none | 7.0 | VERIFY |
| Per-release difference check: the overview paragraphs on FEC, jitter buffer, NACK, 300 to 500 ms detection and one-minute hold time are identical in 6.4 and 7.0 as summarised; no DMPO behaviour difference between 6.4 and 7.0 was found in the pages we could read. Absence of a difference in unreadable pages is not proven. | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-overview | 6.4 vs 7.0 | VERIFIED |
