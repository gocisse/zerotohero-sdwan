# DMPO load balancing, bandwidth aggregation, service and traffic classes

| claim | source URL | doc version | status |
|---|---|---|---|
| For applications that benefit from more bandwidth, such as file transfer, DMPO can spread the packets of one flow across all available links (per-packet load balancing) and resequence them at the receiver. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.5) | SECONDARY |
| Arista's Overview describes per-flow vs per-packet behaviour only through the steering and remediation text; a current sentence on bandwidth aggregation conditions was not captured by the summarising tool. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | VERIFY |
| Example: two 50 Mbps links give 100 Mbps for one flow, with QoS applied at aggregate and per-link level. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.5) | SECONDARY |
| Traffic sent Direct (bypassing the DMPO tunnel) can be load-balanced per flow. Not confirmed for 7.0. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.8) | SECONDARY |
| Real-Time traffic defaults to per-packet steering; Transactional and Bulk default to per-packet load balancing; traffic sent Direct is load-balanced per flow. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.12) | SECONDARY |
| A traffic class is a Priority (High, Normal, Low) combined with a Service Class (Real-Time, Transactional, Bulk): nine classes. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.6) | SECONDARY |
| Service classes Real-Time, Transactional and Bulk exist in current Arista documentation. Only the WP2017 text was read; the summarised 7.0 page did not confirm it. | none | 7.0 | VERIFY |
| Scheduler default weights per traffic class (the 3x3 table) are image-only in the white paper. | none | WP2017 | VERIFY |
| Whether current releases still load-balance Direct traffic per flow, and the exact bandwidth aggregation conditions, are not established. | none | 7.0 | VERIFY |
