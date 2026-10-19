# DMPO on-demand remediation: FEC, jitter buffer, NACK

| claim | source URL | doc version | status |
|---|---|---|---|
| On-demand remediation is used when steering cannot fix the problem (a single link, or several links degraded at once); it is on for the duration of the problem. | https://www.arista.com/assets/data/pdf/SD-WAN-Solution-Overview.pdf | solution brief (read directly) | VERIFIED |
| FEC is enabled automatically on a single link or on several links; with several links DMPO picks up to two of the best links. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| FEC in the guide is packet duplication: on one link the duplicate goes next to the original on that link; with two or more links the copies go on separate links. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | VERIFIED |
| The receiving endpoint discards duplicates and re-orders out-of-order packets before delivering to the destination. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.5) | SECONDARY |
| FEC triggers per flow, in real time, when loss crosses a threshold; it applies to real-time traffic; it is switched off when loss no longer exceeds the threshold. (The threshold value itself is not given in the text we read.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | VERIFIED |
| Cost of FEC: the duplicates use extra bandwidth. (Inferred from "packet duplication"; no percentage figure was found.) | none | 7.0 | VERIFY |
| Jitter buffer: enabled for real-time applications when jitter exceeds 7 ms (voice) or 5 ms (video); the receiver buffers up to 10 packets or 200 ms; disabled if the flow is not constant. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| Cost of the jitter buffer: it holds packets at the receiver, so it can add delay up to the buffer limit. (Our inference from the 200 ms limit; the guide has no "cost" sentence. Do not use as a quiz answer.) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 (WebFetch) | SECONDARY |
| NACK: for TCP applications such as file transfer the receiving endpoint asks the sending endpoint to retransmit a missing packet, so the end application does not see the loss; enabled when loss exceeds a per-application-type threshold. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 text (WebFetch) | VERIFIED |
| NACK keeps the TCP window open and throughput high in lossy conditions (WP2017 wording). | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.5) | SECONDARY |
| Premium Service on = Gateway applies FEC to impaired traffic; off = no Gateway error correction (monitoring and steering continue). | https://www.arista.com/en/operator-guide-vc-7-0/sd-wan-7-0-manage-customers | 7.0 (WebFetch) | VERIFIED |
| Default pairing per service class (WP2017): Real-Time gets FEC and jitter buffer; Transactional and Bulk get NACK. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 (p.12) | SECONDARY |
| Numeric loss thresholds that switch FEC and NACK on per application type were image-only. | none | 7.0 | VERIFY |
| Per-release: jitter buffer limits (7 ms, 5 ms, 10 packets or 200 ms), FEC wording and NACK wording are the same in 6.4 and 7.0 overview text. | https://www.arista.com/en/admin-guide-vc-6-4/sd-wan-6-4-overview | 6.4 vs 7.0 | VERIFIED |
