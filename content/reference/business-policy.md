# Business policy

| claim | source URL | doc version | status |
|---|---|---|---|
| The business policy lets the administrator control QoS, steering and services for application traffic. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Profile path: Configure > Profiles > the Profile > Business Policy tab; Edge path: Configure > Edges > the Edge > Business Policy tab. The Profiles and Edges lists have "Biz. Pol" / "Business Policy" columns that link there. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| A rule is created in a Create Business Policy Rule dialog with a Match tab and an Action tab. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Match, Source: Any, Object Group, or Define (VLAN, Interface, IP address with CIDR / mask / wildcard, ports, operating system). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Match, Destination: Any, Object Group, or Define (Internet, Edge, Non SD-WAN Destination via Gateway or Edge, domain name, protocol, ports). Domain name matches the entire name or a portion of it. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 (6.4 domain wording same) | VERIFIED |
| Protocols that can be matched: GRE, ICMP, TCP, UDP. (WebFetch) | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-configuring-business-policy | 6.4 = 7.0 | VERIFIED |
| Match, Application: Any or Define, which selects an application and can also match a DSCP value on incoming traffic. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Action fields: Priority (High, Normal, Low); Network Service (Direct, Multi-Path, Internet Backhaul); Link Steering; Service Class (Real-time, Transactional, Bulk; for custom applications only); Enable Rate Limit; Enable NAT; Inner and Outer Packet DSCP tags. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 (6.4 lists Priority, Network Service, NAT, Service Class) | VERIFIED |
| Internet Backhaul applies only when the Destination is Internet. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Rate limit applies per flow in the inbound and outbound directions. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Rules are listed in order of highest precedence, and traffic is matched to the rule with the highest precedence. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 = 6.4 | VERIFIED |
| Whether evaluation is first-match top-down, how ties are resolved, and how Profile and Edge rules interleave beyond "the Edge rule supersedes" are not stated. | none | 7.0 | VERIFY |
| When no rule matches a default action applies; its definition is not given in text. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFY |
| Smart Defaults is a ready-made business policy supporting over 2500 applications and maps applications to traffic classes with default scheduler weights. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Business policy rules are segment aware (Segment drop-down). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Profile rules are pushed to all associated Edges; the Edge cannot edit them, it can create a superseding identical rule. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 = 6.4 | VERIFIED |
| Link steering (Auto, Transport Group, Interface, WAN Link; Mandatory / Preferred / Available) is covered in dmpo-steering.md; it is one action field of a rule. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Business policy changes (rules with source Any, destination Internet, rate limit toggle) are classified as Both Profile & Edge Override, both segment types, no Edge service restart (Appendix Table 5, IPv4). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 | VERIFIED |
| The meaning of "Edge2Edge" and an SLA match field were not found on the page. | none | 7.0 | VERIFY |
