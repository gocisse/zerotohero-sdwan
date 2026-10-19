# Segments

Read 2026-10-19. "WebFetch" = HTML page summarised by a tool with verbatim quotes requested (Arista blocks automated browsers; not bypassed). No `reference-sources/` folder exists. The 7.0 Orchestrator Guide PDF text was read directly ("local PDF" rows cite the page).

| claim | source URL | doc version | status |
|---|---|---|---|
| Network segmentation partitions a network into distinct logical sub-networks; the guide frames it as isolating traffic by organization or data type. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| Segments are configured at Configure > Segments in the new Orchestrator UI; Add then Save Changes creates one; a segment is removed by selecting it and Delete. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| A segment that a Profile uses cannot be deleted. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| There is a Global Segment; for it the type can be Regular or Private. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| For Private segments the Orchestrator automatically overrides every business policy set to Internet Multipath and changes it to Direct. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| The segment form has a Service VLAN identifier field. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| Segments are chosen for a Profile on the Profile's Device tab: Segment menu, Change Profile Segments, then select the segments to include. Segments in use show a lock symbol and cannot be removed. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 (6.4 same wording) | VERIFIED |
| The same Change Profile Segments procedure and lock symbol are described in 6.4. (WebFetch) | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-configure-device-settings-for-profiles | 6.4 | VERIFIED |
| Settings are either segment aware (apply to multiple segments) or segment agnostic (apply only to the segment chosen in the Segment menu); the Device tab can sort by segment aware. (WebFetch) | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-configure-device-settings-for-profiles | 6.4 | VERIFIED |
| On an Edge, interface settings by default apply to all segments; an interface setting has a Segments field. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| An Edge can have up to 32 VLANs across 16 segments. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| Loopback interface IDs must be unique across all segments within an Edge. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| Default maximum segments for an Enterprise is 16; the system maximum is a property that can be raised (128 is the figure the guide gives for enabling 128 segments). (local PDF: VC-SD-WAN-7.0-Orchestrator-Guide.pdf, page 36, Table 25) | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Orchestrator-Guide.pdf | 7.0 | VERIFIED |
| Business policy rules are segment aware: a Segment drop-down on the page selects which segment's rules are shown; the default is the Global Segment [Regular]. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 (6.4: "now Segment aware") | VERIFIED |
| Firewall: Port Forwarding and 1:1 NAT rules pick the segment the LAN IP belongs to; the syslog collector is configured per segment. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-firewall-overview | 7.0 | VERIFIED |
| Troubleshooting tests (ping, traceroute, MTR, route and firewall-session views) take a Segment selector and show the segment of a session or route. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 (local PDF, e.g. pages with Ping and Traceroute tests) | VERIFIED |
| Whether general firewall rules are scoped per segment (as opposed to Port Forwarding / 1:1 NAT) was not stated on the page read. | none | 7.0 | VERIFY |
| Limits per Edge model (segments, VLANs per model) were not stated on the segments page. | none | 7.0 | VERIFY |
| Whether traffic between two segments is blocked by default, and how inter-segment traffic can be allowed, was not stated in the pages read. | none | 7.0 | VERIFY |
| Per-segment routing: the segments page gives only a use-case example; routing settings that apply per segment (OSPF, BGP, routes on the Troubleshooting tests) are segment-selectable in tests, but a procedure for per-segment routing config was not read. | none | 7.0 | VERIFY |
