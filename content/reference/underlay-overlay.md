# Underlay and overlay

| claim | source URL | doc version | status |
|---|---|---|---|
| The Edge uses whatever WAN links are present (private line, cable, DSL, LTE or 5G, satellite) and the SD-WAN overlay runs over them as a virtual overlay across multiple links. | https://www.arista.com/assets/data/pdf/SD-WAN-Solution-Overview.pdf | solution brief (undated) | VERIFIED |
| The overlay is built from DMPO tunnels: Edge to Gateway, and Edge to Edge. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Cloud VPN is configured per Segment and offers three tunnel types: Branch to Hub VPN, Branch to Branch VPN, and Edge to Non SD-WAN via Gateway. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| Branch-to-Branch VPN has a "Transit & Dynamic" setting; traffic between branches can pass through Cloud Gateways or through Hubs. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| Dynamic Branch to Branch: traffic first goes through the Gateway or Hub; if the initiating Edge determines a secure multipath tunnel is possible it creates a direct tunnel; the direct tunnel is torn down after 180 seconds of traffic silence; the Orchestrator enables it for all Edges by default. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| The Orchestrator has a Configure > Overlay Flow Control view for the routing table of the overlay. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| "Underlay" is not a term the guides read use for the physical WAN links; the course uses "underlay" for the transport networks (Internet, MPLS, LTE) and "overlay" for the VeloCloud tunnels on top. | none | n/a | VERIFY |
