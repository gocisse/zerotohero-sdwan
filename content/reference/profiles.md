# Profiles

| claim | source URL | doc version | status |
|---|---|---|---|
| Administrators use Profiles to define a template configuration and apply it to multiple Edges simultaneously. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| Profiles provide a composite of the configurations created in Segments and Network Services. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| A default profile, the Quick Start Profile, is available when Orchestrator is installed. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| Profiles live at Configure > Profiles; New Profile then name, description and Create opens the Device tab. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| A profile is duplicated with Duplicate on the Profiles page, or Shortcuts > Duplicate Profile from the profile page (Copy Profile dialog). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| A profile can be deleted (More > Delete) but not while Edges are associated with it. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| The Profiles list has a Used By column showing the number of Edges associated with the Profile. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-profiles | 7.0 | VERIFIED |
| Profile pages include an Overview tab and a Device tab; the Device tab is where segments are assigned and Profile settings are configured. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| Profile Device-tab categories include Connectivity (VLANs, Interfaces), Routing and NAT (for example OSPF areas), Telemetry (syslog), VPN Services and Edge Services. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 | VERIFIED |
| Interface settings of a Profile are applied automatically to the Edges associated with it, and can be overridden for a specific Edge. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-profiles | 7.0 (6.4 same) | VERIFIED |
| Profiles also carry Business Policy and Firewall tabs (Configure > Profiles > profile > Business Policy; Configure > Profiles > Firewall). (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-business-policy | 7.0 | VERIFIED |
| When an Edge is provisioned (Configure > Edges > Add Edge) a Profile is selected from a drop-down menu. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-provision-a-new-edge | 7.0 | VERIFIED |
| The recommended order for configuring (Network, Network Services, Profile, Edge) was seen in a web-search summary only, not on a page read. | none | 7.0 | VERIFY |
| What exactly happens to Edges when a Profile changes (immediate push, restart): the Appendix says the Orchestrator pushes configuration changes (event MGD_CONF_APPLIED) and some changes restart the Edge service; the Profile page itself says nothing. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 (WebFetch) | VERIFIED |
| The Appendix has a section on Edge configuration changes that trigger an Edge service restart (Edged restart, not a hardware reboot, causing a brief traffic interruption; in an HA pair the restart triggers a failover). Each row says whether a change restarts the service. (WebFetch) | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-appendix | 7.0 | VERIFIED |
| Changing an Edge's Profile: only the BGP filter case was found (Profile-level filters on neighbours are removed on switch); the general effect on overrides is not stated. | https://www.arista.com/en/admin-guide-vc-7-0/sd-wan-7-0-configure-device-settings-for-edges | 7.0 | VERIFY |
| Whether the Quick Start Profile can be edited or deleted, and its default contents, were not stated. | none | 7.0 | VERIFY |
