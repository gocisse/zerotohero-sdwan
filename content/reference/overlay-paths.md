# How a flow's path is chosen

| claim | source URL | doc version | status |
|---|---|---|---|
| Branch-to-branch traffic can go via a Gateway (Cloud Gateways option) or via a Hub (Hubs for VPN option); with Dynamic Branch to Branch a direct tunnel is added after the first packets. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| 6.4: with Hubs for VPN the data plane through the Gateway is deactivated for Branch to Branch VPN and traffic first transits a Hub in the order specified. | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-site-configurations-overview | 6.4 | VERIFIED |
| Hub choice between primary and secondary: routes sort by BGP attributes first, then Hub order; if the primary Hub goes down the Spoke moves local flows to the secondary Hub. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| The Gateway test (remote diagnostic) can change the path of cloud (Internet) traffic: use business policy, use the Gateway path (Multi-Path) or go directly on the WAN interface. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 (local PDF, section 5.61 Gateway) | VERIFIED |
| Business policy network service (Direct, Multi-Path, Internet Backhaul) decides how matching traffic is carried; Link Steering and DMPO choose the link or tunnel inside the overlay (Section 3). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| The packets of a flow can take any tunnel, IPv4 or IPv6, based on the real-time health of the tunnel. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| The firewall log records where VPN traffic went: via a Hub or via a backhaul hub. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| The Edge route table's reason codes include PR_UNREACHABLE (for overlay routes, the remote Gateway or Edge peer is unreachable) and the tie-breakers LR_NETADDR and LR_LOGICAL_ID. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 (local PDF, printed pages 86-87) | VERIFIED |
| The documented precedence between route types (connected, static, overlay, NSD, BGP, OSPF) was not found in the pages read; it belongs to routing (step 10). [needs observation] | none | 7.0 | VERIFY |
| Whether a flow prefers a direct dynamic tunnel over the Hub or Gateway path once the tunnel exists, and how a flow is moved: not stated. [needs observation] | none | 7.0 | VERIFY |
| What happens to in-flight flows when a direct tunnel is torn down after 180 seconds: not stated. | none | 7.0 | VERIFY |
| Return traffic from a destination outside the SD-WAN (the Internet or a Non SD-WAN site), and whether the return path through the overlay is the reverse of the forward path, are not stated. [needs observation] | none | 7.0 | VERIFY |
