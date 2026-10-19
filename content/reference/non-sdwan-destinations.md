# Non SD-WAN destinations (NSD)

| claim | source URL | doc version | status |
|---|---|---|---|
| NSD via Gateway: a Gateway builds an IPsec tunnel to an external destination such as a cloud provider, firewall or data center. Configured at Configure > Network Services > Non SD-WAN Destinations via Gateway. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| NSD via Gateway supported types: AWS VPN Gateway, Check Point, Cisco ASA, Cisco ISR, Generic IKEv2 (route-based), Microsoft Azure Virtual Hub, Palo Alto, SonicWALL, Zscaler, Generic IKEv1 (route-based), Generic Firewall (policy-based). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| NSD via Gateway is defined exclusively at the Profile level and cannot be overridden at the Edge level. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| Active/Active allows up to 4 tunnels for most NSD types and Active/Hot-Standby up to 2; Cisco ASA and Generic Firewall allow 1; Azure Virtual Hub up to 2. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| NSD via Edge: an Edge builds an IPsec tunnel to an external destination; types listed are Generic IKEv1 (route-based), Generic IKEv2 (route-based) and Microsoft Azure Virtual WAN. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-network-services | 7.0 | VERIFIED |
| NSD via Edge is added under Configure > Network Services > Non SD-WAN Destinations via Edges (New), then enabled in the Profile Device tab VPN Services; one such service per Segment. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| For NSD via Gateway the Profile Device tab has Cloud VPN on and the "Enable Edge to Non SD-WAN via Gateway" checkbox under "Edge to Non SD-WAN Sites". | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| An Edge's Device tab VPN Services has "Non SD-WAN Destination via Edge: enable to establish tunnel between a branch and Non SD-WAN destination via Edge" with an Add button. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFIED |
| BGP can be configured for Non SD-WAN Destination neighbors in an Edge. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| Diagnostics: MTR can choose its path: Direct, Multipath, NVS via Edge or NVS via Gateway if configured. | https://www.arista.com/assets/data/pdf/qsg/VC-SD-WAN-7.0-Troubleshooting-Guide.pdf | 7.0 (local PDF, printed page 30) | VERIFIED |
| When to choose Gateway versus Edge (trade-offs: where the tunnel terminates, scale, who needs the router): not stated beyond the descriptions above. [needs observation] | none | 7.0 | VERIFY |
| Authentication method (PSK or certificate), primary and secondary destination addresses for NSD via Edge: seen only in a web-search summary. [needs observation] | none | 7.0 | VERIFY |
| Whether the Gateway tunnels from the nearest or a chosen Gateway for every NSD type: not stated except for the Zscaler section's geo-IP selection (not read in detail). | none | 7.0 | VERIFY |
