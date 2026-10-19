# VCO configuration UI map (target release 7.0, new Orchestrator UI)

| claim | source URL | doc version | status |
|---|---|---|---|
| Segments: Configure > Segments (Enterprise portal, New Orchestrator UI). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-segments | 7.0 | VERIFIED |
| Profiles: Configure > Profiles; the Profiles page has Duplicate and More > Delete; a Used By column. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| Profile editor: Overview tab and Device tab; Business Policy and Firewall tabs also exist on a Profile. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Edges: Configure > Edges; Add Edge provisions one; opening an Edge shows a Device tab (configuration) and, per the business-policy and firewall pages, Business Policy and Firewall tabs. An Overview page exists but its contents were not read. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-provision-a-new-edge | 7.0 | VERIFIED |
| Edge Device tab sections include Connectivity (Interfaces, VLAN, ARP timeouts, Loopback Interfaces), Routing and NAT (BFD, OSPF). Path example: Device tab > Connectivity > VLAN > Edit > Override. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| Override: a checkbox beside an inherited setting. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| Edge Business Policy tab: sections "Rules From Profile" and "Edge Overrides"; segment drop-down at the top (default Global Segment [Regular]). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Edge Firewall tab: Override checkbox; Additional Settings > Inbound ACLs; Syslog forwarding button. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-firewall-overview | 7.0 | VERIFIED |
| Monitor > Firewall Logs shows Edge firewall logs. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-firewall-overview | 7.0 | VERIFIED |
| Diagnostics: tests (ping, traceroute, route views, firewall sessions) take a Segment selector. (local PDF: VC-SD-WAN-7.0-Troubleshooting-Guide.pdf) | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 | VERIFIED |
| Enterprise-level Enhanced Security: Configure > Enhanced Security > Security Services. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-firewall-overview | 7.0 | VERIFIED |
| Overlay Flow Control and Network Services exist as configuration areas (Appendix tables 12 and 13); their menu paths were not read and are step 9 scope. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 | VERIFIED |
| The classic UI labels (Configure > Edges tabs in the classic UI) for the same screens were not read. | none | 6.4 | VERIFY |
| How the Edge editor visually flags an override, and whether an "Overview" tab shows the effective configuration, were not read. | none | 7.0 | VERIFY |
| The Applications / application-map screen path (HTTP 403 on fetch). | none | 7.0 | VERIFY |
