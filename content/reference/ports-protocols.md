# Ports and protocols (per flow)

Gathered 2026-10-19. "Read directly" means the text was extracted from a PDF by us. "Summarised via WebFetch" means an Arista HTML page was read through a summarising tool (Arista's website blocks automated browsers, so raw HTML was not obtainable); verbatim quotes were requested and the claim is worded to match them.

| claim | source URL | doc version | status |
|---|---|---|---|
| Edge to Orchestrator: the firewall must allow outbound TCP/443 from the (virtual) Edge for communication with the Orchestrator. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-virtual-edge-deployment | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| Edge to Gateway: the firewall must allow outbound UDP/2426 (VCMP) from the Edge to the Internet. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-virtual-edge-deployment | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| VCMP uses the IANA-registered port UDP 2426 for tunnel management and information exchange between two DMPO endpoints (Edge-Gateway, Edge-Edge). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| VCMP is encrypted with transport-mode IPsec and NAT-T is forced on with the special NAT-T port 2426; the guide says VeloCloud's behaviour is the same whether or not NAT is present (NAT-T is always activated). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| UDP 500 and UDP 4500 are for IKEv2 negotiation and IPsec NAT transparency between DMPO endpoints; IP protocol 50 (ESP) is for IPsec when no NAT exists between them. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| Gateway to Orchestrator: a Gateway behind a firewall or NAT needs outbound TCP/443 for communication with the Orchestrator. | https://www.arista.com/en/partner-guide-vc-7-0/sd-wan-7-0-install-partner-gateway | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| Partner Gateway inbound: the firewall must allow inbound traffic from the Internet to UDP/2426 (VCMP), UDP/4500 and UDP/500, and IP/50 (ESP) when NAT is not used; with NAT those ports must be translated to an externally reachable IP address, and 1:1 NAT and port translation are both supported. | https://www.arista.com/en/partner-guide-vc-7-0/sd-wan-7-0-install-partner-gateway | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| The same Partner Gateway requirements appear in the 6.4 Partner Guide. | https://www.arista.com/en/partner-guide-vc-6-4/sd-wan-6-4-install-partner-gateway | 6.4, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| Edge to Edge behind NAT: with unconfigured NAT devices on the WAN side, incoming VCMP tunnels on UDP/2426 from other Edges are blocked, so the direct tunnel will not likely come up; UDP hole punching is a workaround that is not applicable to all NAT types. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| An on-premises Orchestrator needs the upstream firewall to allow inbound HTTP (TCP/80) and HTTPS (TCP/443); in the on-premises design the GUI uses HTTPS and the Orchestrator makes HTTPS management connections to Edges over the underlay and the overlay. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Orchestrator-Guide.pdf | 7.0 PDF, read directly | VERIFIED |
| The Orchestrator-Edge diagnostic and heartbeat channel is a persistent WebSocket over a single TCP connection (the port is not stated). | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 PDF, read directly | VERIFIED |
| Every documented management flow read (Edge to Orchestrator, Gateway to Orchestrator, browser to Orchestrator) is HTTPS or TCP/443; the guides do not claim that this is the only management port. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Orchestrator-Guide.pdf | 7.0 PDF, read directly | VERIFIED |
| An Edge acting as a Hub needs UDP/2426 inbound, and DNS (UDP/53) and NTP (UDP/123) are required outbound. These appear only in search snippets of Broadcom's ports article (knowledge.broadcom.com 312353), which could not be fetched. | https://knowledge.broadcom.com/external/article/312353/ | n/a | VERIFY |
| The default and maximum IP MTU across the overlay without fragmentation is 1500 (see mtu-overhead.md). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |

## Per-flow summary (for lesson 6)

| Flow | Port / protocol | Direction | Status | Notes |
|---|---|---|---|---|
| Edge to Orchestrator | TCP 443 | Outbound from the Edge | VERIFIED | stated for the virtual Edge firewall |
| Edge to Gateway | UDP 2426 (VCMP) | Outbound from the Edge | VERIFIED | NAT-T is always on with port 2426 |
| Edge to Edge (direct tunnel) | UDP 2426 | Outbound; incoming 2426 must also be reachable behind NAT | VERIFIED | needed for the direct tunnel to come up |
| Gateway to Orchestrator | TCP 443 | Outbound from the Gateway | VERIFIED | |
| Internet to a Partner Gateway | UDP 2426, UDP 4500, UDP 500, plus ESP when no NAT | Inbound | VERIFIED | with NAT, translate the ports (1:1 or port translation) |
| Browser to an on-prem Orchestrator | TCP 443 (and TCP 80) | Inbound | VERIFIED | hosted Orchestrator not covered |
| Edge as a Hub, inbound | UDP 2426 | Inbound | VERIFY | seen only in an unreadable Broadcom article |
| DNS and NTP from the Edge | UDP 53, UDP 123 | Outbound | VERIFY | seen only in an unreadable Broadcom article |
