# Applications, QoS and traffic classes

| claim | source URL | doc version | status |
|---|---|---|---|
| DMPO identifies traffic using Layer 2 through Layer 7 attributes and can steer applications from the very first packet by caching earlier classification results. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Each application has a Service Class (Real Time, Transactional, Bulk) that determines how DMPO handles its traffic, and a Business Priority (High, Normal, Low) that users can change for any application. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| Service Class cannot be changed for default applications but can be set for custom applications. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 = 6.4 rule text | VERIFIED |
| Smart Defaults maps over 2500 applications to traffic classes with a default scheduler weight per class. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-overview | 7.0 | VERIFIED |
| A traffic class is a Priority combined with a Service Class: a 3x3 matrix of nine classes, with scheduler weights and optional policers per class (Overlay QoS CoS Mapping). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Firewall rules that match an application depend on the DPI engine. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-firewall-overview | 7.0 | VERIFIED |
| Users can define custom applications and a user-defined application map used as match criteria (a business policy change that does not restart the Edge service). (WebFetch of Appendix; page-level detail from a search summary only) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 | SECONDARY |
| DPI needs the first 5-10 packets of a flow to identify the application, and learned classifications are cached by destination IP and port. (web-search summary only; page not read) | none | 7.0 | VERIFY |
| The Configure > Applications / application-map page could not be fetched (HTTP 403), so menu names for it are not confirmed. | none | 7.0 | VERIFY |
| The inner and outer DSCP tag fields exist in the rule action; what the WAN carrier does with DSCP is outside the documentation read. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Rate limit in a rule applies per flow in both directions. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| Relationship to Section 3: Service Class is what DMPO uses to choose a remediation (the default pairing is from the legacy 2017 white paper) and link steering is another action in the same rule. | https://cdn.allbound.com/velocloud-ab/2019/03/DMPO-White-Paper.pdf | WP2017 | SECONDARY |
| Default scheduler weights per traffic class are an image only. | none | 7.0 | VERIFY |
