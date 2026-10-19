# DMPO monitoring: probes, measurements, intervals, link states

Sources: see `dmpo.md` for how each was read. "WebFetch" rows were summarised by a tool with verbatim quotes requested.

| claim | source URL | doc version | status |
|---|---|---|---|
| DMPO measures loss, latency and jitter per tunnel in one direction at a time, so uplink and downlink are judged independently. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| While user traffic flows, DMPO adds a sequence number and a timestamp to the tunnel header; the receiving endpoint uses them to spot lost and out-of-order packets and to compute jitter and latency. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| The two endpoints exchange the measured metrics every 100 ms. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| With no user traffic an active probe is sent every 100 ms; after five minutes with no high-priority user traffic the probe interval becomes 500 ms. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| Passive plus active: DMPO uses both passive (in-band, on user traffic) and active (probes) monitoring. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.3) | SECONDARY |
| Packet loss is computed over a one-minute window, using counts the endpoints exchange every second. (Summarised via WebFetch; wording not read directly.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | VERIFIED |
| A brownout or outage is detected within 300 to 500 ms; on loss of communication the action is immediate (within 300 ms in the summarised text). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| Automated Bandwidth Discovery: after the Edge detects a WAN link it sets up tunnels to Gateways and runs a short burst bandwidth test, in both directions, against the closest Gateway, measuring the received rate at each end. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | VERIFIED |
| Because the Gateway sits at an Internet point of presence it can also discover the real public IP of a link behind NAT. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.3) | SECONDARY |
| For a hub or head-end Edge the WAN bandwidth is statically configured; branches testing toward a hub Edge use a similar test as on public links. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.3) | SECONDARY |
| The 7.0 Appendix lists a LINK_DEAD event, raised when no packets are received on a link for seven seconds. (Summarised via WebFetch.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 (WebFetch) | VERIFIED |
| Links can be reported as USABLE or UNUSABLE for steering purposes. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 (WebFetch) | VERIFIED |
| In the WP2017 default-behaviour table, a link that misses three consecutive heartbeats is marked unusable and flows go to the next best link. Not found in the 7.0 text we could read, so teach it only as legacy. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.13) | SECONDARY |
| Exact probe packet size and the precise loss, latency and jitter measurement algorithms were not found. | none | 7.0 | VERIFY |
| Per-release: monitoring intervals (100 ms, 500 ms after five minutes) are the same in 6.4 and 7.0 overview text. Link-event names were read for 7.0 only. | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-overview | 6.4 vs 7.0 | VERIFIED |
