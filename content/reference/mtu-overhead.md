# MTU and tunnel overhead

| claim | source URL | doc version | status |
|---|---|---|---|
| VCMP adds 31 bytes of overhead to user packets (VCMP header 23 bytes + VCMP data header 8 bytes). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Internet Multipath traffic overhead totals 59 bytes: IP header 20 + UDP header 8 + VCMP header 23 + VCMP data header 8. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| VPN (IPsec) traffic overhead totals 97 to 112 bytes: IP 20, UDP 8, IPsec sequence number 4, IPsec SPI 4, VCMP header 23, VCMP data header 8, initialization vector 16, padding 0-15, padding length 1, next header 1, authentication data 12. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| The Effective Packet MTU is computed by subtracting this overhead, depending on whether the destination is VPN or Internet Multipath. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Derived arithmetic (not printed in the guide): on a 1500-byte path MTU, Internet Multipath traffic leaves 1500 - 59 = 1441 bytes, and VPN traffic leaves 1500 - (97 to 112) = 1388 to 1403 bytes. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Path MTU discovery: the Edge first tries RFC 1191, then falls back to RFC 4821; for public Internet links discovery runs toward the Gateways and all tunnels use the minimum MTU found; for private WAN links it runs toward the other Edges and each tunnel gets its own result. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| RFC 4821 probing measures MTU only down to 1300 bytes; if a link's MTU is below 1300 the MTU must be configured manually. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| VeloCloud SD-WAN does not support jumbo frames (as of Release 5.0). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| An IPv6 underlay tunnel adds 20 bytes (larger IPv6 header), so the effective path MTU is reduced by 20 bytes. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Path MTU discovery starts from a default link MTU of 1500 bytes, and the maximum supported IP MTU for packets sent across the overlay without fragmentation is 1500. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
| When a packet is too large the Edge automatically performs IP fragmentation as per RFC 791. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFIED |
