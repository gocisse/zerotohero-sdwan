# Release target

Which Arista VeloCloud SD-WAN release this course is written against, and the evidence. Checked 2026-10-19.

| claim | source URL | doc version | status |
|---|---|---|---|
| Arista's documentation library lists a 7.0 documentation set (Administration, Design, Operator, Orchestrator, Troubleshooting, Gateway Monitoring and other guides) and product bulletins for 7.0.0, 6.4.2, 6.4.1 and 6.4.0. | https://www.arista.com/en/support/product-documentation | library page, read 2026-10-19 | VERIFIED |
| The 7.0 Administration Guide has a "What's New in Version 7.0.0" table (e.g. BGP filter-list match options, ENAv2 on AWS, Link Insights on-prem, LLDP on Edge). | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-whats-new | 7.0 | VERIFIED |
| Orchestrator (VCO) builds in the 7.0.x train exist: Arista Security Advisory 0183, dated 2026-09-22, lists "7.0.0.2 and below" as affected, next to the 5.2, 6.1 and 6.4 trains. So 7.0.x is a shipped train with maintenance builds. | https://www.arista.com/en/support/advisories-notices/security-advisory/24765-security-advisory-0183 | advisory rev 1.1 | VERIFIED |
| Release 6.4.0 was released 2025-05-02 with builds R6400-20250429-GA (Gateway, Edge) and R6400-20250430-GA (Orchestrator) and was offered as a Long-Term Support (LTS) release candidate. | https://www.arista.com/assets/data/pdf/release-notes/VC-SD-WAN-6.4-Release-Notes.pdf | 6.4.0 release notes | VERIFIED |
| Arista's Edge 7x0 datasheet (read directly) says Edges must be on software code R7.0.0 to achieve the listed performance, and footnotes ENAv2 support as available in 7.0.0F, 6.1.4 and 6.4.2. | https://www.arista.com/assets/data/pdf/Datasheets/VeloCloud-SD-WAN-Edge-7x0-Series.pdf | datasheet, read directly | VERIFIED |
| Arista Security Advisory 0144 (dated 2026-07-27) lists Orchestrator 7.0.x releases prior to 7.0.0.1 as affected, and advisory 0183 (2026-09-22) lists 7.0.0.2 and below, so 7.0.0, 7.0.0.1 and 7.0.0.2 Orchestrator builds are in the field. | https://www.arista.com/en/support/advisories-notices/security-advisory/24364-security-advisory-0144 | advisory, summarised via WebFetch | VERIFIED |
| The 7.0.0 release notes (GA date, build strings, LTS or STS designation) could not be read: the guides point to them and the product bulletin is listed, but Arista's site blocks automated browsers and a guessed PDF address was rejected. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-whats-new | 7.0, summarised via WebFetch (verbatim quotes requested) | VERIFY |
| Arista's "Recommended Releases" knowledge-base article (which names the currently recommended LTS or STS build) could not be read (the page returned only a loading stub). | https://arista.my.site.com/AristaCommunity/s/article/VeloCloud-SD-WAN-Software-Versions-Recommended-Releases | n/a | VERIFY |
| A search-result summary of the recommended-releases article said Arista uses an LTS / STS / EOGS methodology and listed 6.4.2 and later as LTS. The article itself was not read. | https://arista.my.site.com/AristaCommunity/s/article/VeloCloud-SD-WAN-Software-Versions-Recommended-Releases | n/a | VERIFY |

## Decision

The course targets the **7.0 documentation set (release train 7.0.x)** because it is the newest documentation set Arista publishes and a 7.0.x Orchestrator train is demonstrably in the field. Its "GA" and "LTS" labels are **not** confirmed (VERIFY). Facts used in Section 1 were chosen to be stable: where the 6.4 documentation was also readable the same statements apply, and a lesson never depends on a feature that is new in 7.0.
