# Simulator data model (design only, no code yet)

Design for the later Go simulator and mock VCO. Behaviour is taken only from rows VERIFIED in `content/reference/` (`segments`, `profiles`, `edge-overrides`, `business-policy`, `firewall`, `applications-qos`, `dmpo-*`, `tunnel-types`, `mtu-overhead`, `vco-config-ui`). Anything else is an **assumption** and is listed in section 7. Terms match `content/glossary.yaml`.

## 1. Entities and relationships

```
Customer 1--* Site 1--* Edge *--1 Profile
Customer 1--* Segment            Profile *--* Segment (included)
Customer 1--* Profile            Profile 1--* BusinessRule, FirewallRule, DeviceSetting
Edge 1--* Interface 1--0..1 WanLink
Edge 1--* EdgeOverride           (device setting | edge business rule | edge firewall rule)
Customer 1--* Gateway (assigned: primary + secondary per Edge)
```

| entity | key fields (authored) | notes |
|---|---|---|
| Customer | id, name, `max_segments` (default 16), `premium_service` (default true) | defaults are VERIFIED (segments, dmpo sheets). Stateful Firewall moved to a Profile/Edge setting in step 11 (assumption 39, section 16) |
| Site | id, name, region | grouping only; region drives Gateway assignment |
| Edge | id, name, site, model, `profile`, `role` (branch or hub) | exactly one Profile (VERIFIED: chosen at provisioning) |
| Interface | id, edge, kind (`wan` / `lan` / `routed` / `loopback`), `vlan`?, `segments` (default: all), addressing | loopback is Edge-only (VERIFIED); WAN interfaces attach to one WanLink |
| WanLink | id, interface, `type` (`public` / `private`), `medium` (wired, wireless), bandwidth up/down | transport groups derive from type and medium: Public Wired, Public Wireless, Private Wired (VERIFIED names) |
| Segment | id, name, `type` (`regular` / `private`), service_vlan? | the Global Segment always exists (VERIFIED) |
| Profile | id, name, `segments`, device settings, business rules, firewall rules | composite of segments and network services (VERIFIED); Quick Start exists |
| EdgeOverride | edge, `kind`, `setting` or rule | see section 2 |
| BusinessRule | id, segment, match, action, owner (`profile` / `edge:<id>`) | list order = precedence (VERIFIED wording) |
| FirewallRule | id, match, action, owner | actions Allow, Drop, Reject, Skip |
| Gateway | id, region, pool | assignment by geolocation, primary and secondary (VERIFIED white paper) |
| Application | id, name, service_class, business_priority, custom? | default service class is not editable (VERIFIED) |
| CloudVpn (per Profile, per segment) | enabled, `branch_to_hub`, `branch_to_branch` (`cloud_gateways` / `hubs_for_vpn`), `dynamic_b2b` (all or profile), `edge_to_nsd_via_gateway` | VERIFIED names and options (cloud-vpn sheet) |
| HubDesignation (in a Profile) | ordered list of hubs (Edge or Cluster), `backhaul` flag | order is a tie-breaker (VERIFIED) |
| Cluster | id, members (Edges), `auto_rebalance` (default false) | members same model; other Edges see one Hub (VERIFIED) |
| NonSdwanDestination | id, `via` (`gateway` / `edge`), type, tunnels (active-active or hot-standby), segment | via gateway: Profile level only (VERIFIED) |
| NetworkService | kind (netflow, dns, private_network_name, prefix_delegation_tag, authentication, tacacs, vnf, api_credential), params | one list, not computed (network-services-other sheet) |

## 2. Effective configuration

Computed per Edge, per segment where a setting is segment aware. Rules below are VERIFIED unless in section 7.

1. **Device settings**: for each setting `s`, effective = Edge override value if the Edge has an override for `s`, else the Profile value. Override is per setting. Exceptions encoded as data: Profile VLAN name and ID are never overridable; loopback interfaces exist only on the Edge.
2. **Business rules**: the Edge's rule list for a segment is its Edge rules plus the inherited Profile rules ("Rules From Profile", "Edge Overrides"). Profile rules cannot be edited on the Edge. An Edge rule with the **same match** as a Profile rule supersedes it. The list is shown in order of precedence. Rules are per segment.
3. **Firewall rules**: Edge override rules take priority over the inherited Profile rules; Profile rules cannot be edited at the Edge. Stateful Firewall is a customer/Profile setting (default off); Reject and Skip are invalid unless it is on.
4. **Profile switch**: inherited VLANs of the old Profile are removed; an interface tied to a removed VLAN reverts to the new Profile; Profile BGP filters on neighbours are dropped (VERIFIED side effects only).
5. **Private segment**: business rules whose network service is Internet Multipath are treated as Direct.
6. **Link steering scope**: Interface and WAN Link steering are valid only on Edge-level rules; Transport Group on both.

Output: an `EffectiveConfig` document per Edge listing each setting with `value` and `source` (`profile`, `edge-override`, `edge-only`, `default`). The `source` field is what the mock VCO uses to answer "where is this coming from?".

## 3. Topology file (`content/topologies/*.yaml`)

Authored: customer, sites, edges, interfaces, links, segments, profiles, overrides, rules. **Computed** by the simulator: effective configuration, tunnels (Edge to Gateway, Edge to hub, dynamic branch to branch, to non SD-WAN destinations), paths, link metrics over time, flow routes. Worked example (3 sites: HQ hub, Branch A, Branch B):

```yaml
version: 1
customer:
  id: acme
  name: Acme Ltd
  max_segments: 16
  premium_service: true
  stateful_firewall: false

segments:
  - { id: global, name: Global Segment, type: regular }
  - { id: guest, name: Guest, type: private }

gateways:
  - { id: gw-eu, region: eu-west }

profiles:
  - id: branch
    name: Branch Profile
    segments: [global, guest]
    device:
      arp_timeout_s: 120          # example setting
    business_rules:
      - id: voice-mpls
        segment: global
        match: { application: voice }
        action: { priority: high, network_service: multipath, steering: { mode: transport_group, group: private_wired, strictness: preferred } }
      - id: stream-direct
        segment: global
        match: { application: streaming }
        action: { priority: low, network_service: direct }
    firewall_rules:
      - { id: allow-all-lan, match: { source: lan }, action: allow }

sites:
  - id: hq
    region: eu-west
    edges:
      - id: hq-edge
        role: hub
        profile: branch
        interfaces:
          - { id: ge1, kind: wan, link: hq-inet }
          - { id: ge2, kind: lan, vlan: 10, segments: [global] }
  - id: branch-a
    region: eu-west
    edges:
      - id: a-edge
        role: branch
        profile: branch
        interfaces:
          - { id: ge1, kind: wan, link: a-inet }
          - { id: ge2, kind: wan, link: a-mpls }
          - { id: ge3, kind: lan, vlan: 10, segments: [global] }
          - { id: ge4, kind: lan, vlan: 20, segments: [guest] }
        overrides:
          device: { arp_timeout_s: 300 }
  - id: branch-b
    region: eu-west
    edges:
      - id: b-edge
        role: branch
        profile: branch
        interfaces:
          - { id: ge1, kind: wan, link: b-inet }
          - { id: ge3, kind: lan, vlan: 10, segments: [global] }

links:
  - { id: hq-inet, type: public, medium: wired, up_mbps: 500, down_mbps: 500 }
  - { id: a-inet,  type: public, medium: wired, up_mbps: 100, down_mbps: 100 }
  - { id: a-mpls,  type: private, medium: wired, up_mbps: 50, down_mbps: 50 }
  - { id: b-inet,  type: public, medium: wired, up_mbps: 100, down_mbps: 100 }
```

## 4. What each later phase computes

| phase | computes | supported by | blocked or limited (VERIFY) |
|---|---|---|---|
| 1. Effective config | layer merge, `source` per setting | profiles, edge-overrides, segments | how overrides are shown or reverted; full list of overridable settings; behaviour on later Profile edits for overridden settings |
| 2. Tunnel formation (implemented in step 12, section 17) | static tunnels (Edge to Gateway, branch to Hub), dynamic branch to branch (on demand, torn down after 180 s silence), NSD tunnels; see sections 8 and 17 | cloud-vpn, hubs-and-branches, non-sdwan-destinations, tunnel-types, vcmp-tunnels, gateway-details | further build conditions beyond "secure overlay multi-path tunnel possible"; Edge-as-Hub inbound ports; NAT cases; Edge behaviour when the Orchestrator is unreachable |
| 3. Flow path selection (implemented in step 12, section 17) | which peer path (Gateway, Hub, direct, NSD) and which WAN link a flow takes; see sections 8 and 17 | overlay-paths, business-policy, dmpo-steering, tunnel-types | rule evaluation order; default action; Edge vs Profile rule interleaving; direct-tunnel preference over Hub/Gateway path; route-type precedence (step 10) |
| 4. Policy matching | application, source, destination, DSCP, port, domain matching; firewall verdict | business-policy, firewall, applications-qos | DPI first-packets behaviour; firewall-versus-business-policy order; segment scoping of firewall rules |
| 5. Link steering and DMPO | Auto, Mandatory, Preferred, Available; outage and brownout; hold time; FEC, jitter buffer, NACK | dmpo-monitoring, dmpo-steering, dmpo-remediation | SLA threshold values; FEC and NACK numeric thresholds; aggregation conditions; Private-link Preferred cases |
| 6. Routing | per-segment routes, route sources, redistribution, selection, reachability; see sections 10 and 11 | routing-overview, static-routes, ospf, bgp, route-redistribution-and-precedence, overlay-flow-control | default states of several redistribution switches, filter evaluation order, cross-segment routing, tie behaviour (section 12) |
| 7. MTU | effective MTU per tunnel | mtu-overhead | none significant |

Behaviours we cannot yet model because they are VERIFY (step 9 additions in section 8): Edge behaviour with an unreachable Orchestrator; per-application SLA values; segment-to-segment traffic; firewall and business policy ordering; DPI timing; scheduler default weights; the Quick Start Profile contents.

## 5. Topology validation rules (all errors at once, `file: message`, same style as the content validator)

- Unique ids per kind; every reference resolves (edge to site and profile, interface to link, rule to segment, override to a setting that exists, gateway region).
- Exactly one Profile per Edge; Profile segments exist; no more segments than `max_segments`; an interface's segments are included in the Edge's Profile.
- An Edge has at most 32 VLANs across 16 segments; VLAN ids unique per Edge; loopback ids unique across segments within an Edge.
- Overrides: not on Profile VLAN name or ID; loopbacks must be Edge-level; an Edge rule must not claim `owner: profile`.
- Link steering `interface` or `wan_link` only on Edge-owned rules; `transport_group` allowed anywhere; strictness only with interface, wan_link (and transport group as authored).
- Firewall `reject` and `skip` require `stateful_firewall: true`; protocols limited to gre, icmp, tcp, udp.
- Network service `internet_backhaul` only with destination Internet; service class set only on custom applications.
- Every WAN interface has a link; every Edge has at least one WAN link; private links need a private-capable peer (hub) to be useful (warning, not error).
- Cloud VPN: `branch_to_hub` needs at least one hub that exists and is an Edge or cluster; `hubs_for_vpn` needs `branch_to_hub` hubs; `branch_to_branch.via` is `cloud_gateways` or `hubs_for_vpn`; segments listed exist in the Profile.
- Cluster members share a model; an Edge is not both a spoke of and a member of the same cluster (warning).
- `non_sdwan_destinations` with `via: gateway` may be referenced only from Profiles, never from Edge overrides; `via: edge` allows at most one per segment; tunnel counts respect the per-type limits.
- No more than 16 tunnels between an Edge and another Edge or Gateway.
- Unknown YAML keys are errors.

## 6. Mock VCO screens to build first

| mock screen | data it needs | real screen (vco-config-ui) |
|---|---|---|
| Configure > Segments | segments, type, in-use lock | Configure > Segments |
| Configure > Profiles | profiles, Used By, segments | Configure > Profiles |
| Configure > Edges (list) | edges, profile, business policy and firewall links | Configure > Edges |
| Edge > Device | effective settings with Override checkbox and `source` | Configure > Edges > Device |
| Edge > Business Policy | segment drop-down, Rules From Profile, Edge Overrides | Edge Business Policy tab |
| Edge > Firewall | rules, Override, inbound ACLs | Edge Firewall tab |
| Monitor > Edges > Links / Paths | link metrics, tunnels | Monitor > Edges tabs |
| Diagnostics (List Paths, flow trace) | computed path of a flow | Diagnostics |

Each screen is read-only at first and fed by the effective-config and path computations; the "Override" marker display is an **assumption** because the real UI's marking was not read.

## 7. Assumptions (not VERIFIED, must be revisited)

1. Business rules are evaluated top-down, first match wins; precedence ties do not occur.
2. When no business rule matches, Smart Defaults classification applies (default action undefined in docs).
3. An Edge rule sits above Profile rules in precedence when both match.
4. Whether a firewall decision happens before or after business-policy classification: the simulator will apply firewall first, flagged as an assumption.
5. A change to a Profile setting reaches an overriding Edge only for settings it did not override.
6. Segments are fully isolated: no traffic crosses segments unless a rule says so.
7. General firewall rules apply to all segments.
8. Overridden settings are marked with a badge; reverting removes the override and restores the Profile value.
9. Dynamic branch-to-branch tunnels appear after the first flow between two branches (trigger conditions unspecified).
10. Link transport group is derived from `type` and `medium` as in the Links tab groups.
11. Quick Start Profile has no special behaviour beyond being a default Profile.
12. An interface's segments must be among the segments its Edge's Profile includes (validation rule in section 5).
13. A private link is only useful with a private-capable peer such as a hub (validation warning in section 5).

## 8. Overlay tunnels and path selection (step 9)

Sources: `cloud-vpn`, `hubs-and-branches`, `non-sdwan-destinations`, `overlay-paths`, `network-services-other` (all 7.0, WebFetch and local Troubleshooting PDF). Rules marked (V) are VERIFIED; anything else is in section 9.

### Topology additions (YAML, authored)

```yaml
profiles:
  - id: branch
    cloud_vpn:
      segments: [global]            # Cloud VPN is configured per segment (V)
      branch_to_hub: { enabled: true, hubs: [hq-edge, hub-cluster-1], backhaul_hubs: [] }   # list order = Hub order (V)
      branch_to_branch: { enabled: true, via: hubs_for_vpn }   # or cloud_gateways (V)
      dynamic_b2b: true                                       # all Edges or per Profile (V)
      edge_to_nsd_via_gateway: [azure-vhub]
clusters:
  - { id: hub-cluster-1, members: [hq-edge, hq-edge-2], auto_rebalance: false }
non_sdwan_destinations:
  - { id: azure-vhub, via: gateway, type: azure_virtual_hub, mode: active_hot_standby }   # Profile level only (V)
```

### Tunnel formation rules (computed)

1. **Edge to Gateway (static):** every Edge has tunnels to its assigned Gateways (primary and secondary, by region). Mode `static`. (V: Gateway assignment by geolocation; Cloud Gateways option tunnels to the closest Gateway.)
2. **Branch to Hub (static):** if Cloud VPN and `branch_to_hub` are on for a segment, each Edge in the Profile gets a permanent tunnel to every designated Hub or cluster. (V)
3. **Cluster membership:** other Edges see a cluster as one Hub; the Gateway picks a member as logical ID modulo the number of Hubs; Auto Rebalance default off. (V)
4. **Standing branch-to-branch path:** `cloud_gateways` means via the closest Gateway; `hubs_for_vpn` means via a Hub in order (6.4: the Gateway data plane is deactivated for Branch to Branch VPN in that case). (V)
5. **Dynamic tunnel:** if `dynamic_b2b` is on, the first packets use the standing path; the initiating Edge builds a direct tunnel if it decides a secure overlay multi-path tunnel is possible; it is torn down after 180 s without traffic. Mode `dynamic`. (V)
6. **NSD via Gateway:** a Gateway-built IPsec tunnel set per destination (up to 4 active-active for most types, 2 hot-standby, 1 for Cisco ASA and Generic Firewall); Profile-level only. (V)
7. **NSD via Edge:** an Edge-built IPsec tunnel, one service per segment. (V)
8. **Tunnel cap:** at most 16 tunnels between an Edge and another Edge or Gateway. (V)

### Path selection (computed)

1. Classify the flow with the effective business policy (section 2): network service Direct, Multi-Path or Internet Backhaul. (V)
2. Multi-Path flow to another site: use a direct dynamic tunnel if one exists (assumption 14); otherwise the standing path (Gateway or Hub per rule 4).
3. Between several Hubs: sort by routing attributes, then Hub order. (V) If the primary Hub is down, a Spoke re-points local flows to the secondary; peer flows stay until flushed. (V)
4. Choose the WAN link and tunnel inside the path with DMPO steering and remediation (Section 3 sheets). (V)
5. Destination is an NSD: use the NSD tunnel (via Gateway or via Edge) as the egress. (V for existence; selection precedence is assumption 15)

### Mock screens added

| mock screen | data | real screen |
|---|---|---|
| Profile > Device > VPN Services | cloud_vpn settings, Override on the Edge | Configure > Profiles > Device > VPN Services |
| Configure > Network Services | NSD, clusters and hubs, network services list | Configure > Network Services |
| Remote Diagnostics > List Paths | tunnels with WAN link, local and remote IP, state, mode, uptime | Diagnostics > Remote Diagnostics > List Paths |
| Remote Diagnostics > VPN Test | Pass or Fail and latency per peer | VPN Test |
| Monitor > Edges > Paths | peers and paths | Paths tab |

### Modelable now versus blocked on VERIFY

- **Modelable now:** static tunnel formation, Hub order tie-break, cluster assignment, dynamic tunnel lifecycle (build decision as a configurable stub, teardown at 180 s), NSD tunnel counts, the Hub-failover re-point, List Paths and VPN Test outputs.
- **Blocked:** the exact "secure overlay tunnel possible" decision, Edge-as-Hub inbound ports and NAT effects on direct tunnels beyond the documented UDP 2426 behaviour, Edge behaviour when the Orchestrator is unreachable, and preference between a direct tunnel and the Hub path. (Route-type precedence is now modelled in section 10.)

## 9. Added assumptions (step 9)

14. A flow uses an existing direct dynamic tunnel in preference to the standing Gateway or Hub path.
15. When a flow could use an NSD tunnel or an overlay path, business policy (network service and destination) decides; no further precedence is modelled.
16. The "secure overlay multi-path tunnel possible" decision is true unless the topology says a NAT blocks incoming UDP 2426 to the peer (documented effect for unconfigured NAT, Section 2).
17. Edge Hub inbound requirement is UDP 2426 (not read on a page). (Step 12: not used; a Hub behind a NAT is returned as not modelled.)
18. In-flight flows move to the Gateway or Hub path without loss when a dynamic tunnel is torn down. (Step 12: the engine does not use this; it returns a not-modelled marker for in-flight flows at teardown.)
19. A cluster's member for an Edge is stable between rebalances; Cluster Score is a simple average of CPU, memory and tunnel use when simulated. (Step 12 models the assignment by logical ID only; Cluster Score and rebalancing are not modelled.)
20. Edges keep forwarding on last-known configuration when the Orchestrator is unreachable (simulator default, not documented). (Step 12: the engine does not use this; it returns not modelled and tunnel state unknown.)

## 10. Routing model (step 10)

Sources: `routing-overview`, `static-routes`, `ospf`, `bgp`, `route-redistribution-and-precedence`, `overlay-flow-control`, `vco-routing-ui` (7.0; WebFetch rows and local Troubleshooting Guide pages 36-47, 58-65, 80-85). (V) = VERIFIED; everything else is in section 12.

### Topology additions (YAML, authored)

```yaml
edges:
  - id: a-edge
    routing:
      static_routes:                     # per Edge; segment scope is assumption 28
        - { subnet: 192.168.50.0/24, next_hop: 10.0.0.2, cost: 0, preferred: false, advertise: true, probe: true }
      ospf:                              # Profile defines areas; Edge enables interfaces; Global Segment only (V)
        interfaces: [{ id: ge3, mode: active }]
        redistribute: { overlay_prefixes: false, bgp: false, default_route: none, metric: 20, metric_type: e2 }   # V defaults: metric 20, E2, default route None
        summaries: [{ subnet: 10.20.0.0/23, no_advertise: false, metric_type: e2, metric: 20, tag: 1 }]
      bgp:
        local_asn: 65010
        neighbors: [{ ip: 10.0.0.9, asn: 65001, inbound_filter: lp-high, outbound_filter: null, uplink: false }]
        redistribute: { overlay_prefix: false, connected: false, ospf: false, default_route: no }   # defaults assumption 22
        filters:
          - { id: lp-high, rules: [{ match: { prefix: 10.30.0.0/24 }, action: permit, set: { local_preference: 200 } }] }
      advertise_flags: { static: true, connected: true, internal: true, external_ospf: false, nsd_ibgp: false }  # V: external OSPF and NSD iBGP off
lan_routers:                              # simulated neighbors that supply routes
  - { id: lan-a1, edge: a-edge, protocol: ospf, routes: [{ prefix: 10.20.0.0/24, type: intra, cost: 10 }] }
overlay:
  dcc: true
  pins: [{ segment: global, prefix: 10.30.0.0/24, exits: [a-edge, hub-1] }]
```

### Route sources, per Edge and per segment

A route has: `prefix`, `segment`, `source` (connected, static, ospf, bgp, overlay-remote, nsd-static, nsd-bgp, cloud), `route_type` (V: OSPF O, O IA, O E1, O E2; BGP iBGP or eBGP; remote connected or static), `cost`, `preference`, `next_hop`, `reachable` (+ reason code), `advertise`, `owner_edge`.

### Learning (inputs to each Edge's table)

1. **Connected:** one route per Edge interface subnet (V). `advertise` follows the Global Advertise Flags.
2. **Static:** one route per configured entry; `advertise` = the Advertise box; `reachable` follows the ICMP probe state (SLA_STATE_UP or SLA_STATE_DOWN) (V).
3. **OSPF:** from LAN-router routes, only if OSPF is enabled on the interface and the neighbor state is Full (V diagnostics); passive on switched interfaces (V). Routes carry type O, O IA, O E1 or O E2.
4. **BGP:** from neighbors that are Established; inbound filter applied (permit or deny, set local preference, metric, community, prepend); `advertise` true and `reachable` yes needed for the overlay (V).
5. **Overlay (remote):** the Gateway reflects every advertised route to the other Edges of the Profile's segment; remote routes get a transit metric of 0 (peer Edge), 32 plus Hub order (via Hub) or 90 (via Gateway) (V).

### Redistribution rules (all directions are explicit switches)

| from to | switch | state in model |
|---|---|---|
| connected or static to overlay | `advertise` flag per route and Global Advertise Flags | V |
| OSPF to overlay | automatic (V), filtered by the external_ospf flag (V: off by default) | V |
| BGP to overlay | route advertise true and reachable yes (V); nsd_ibgp flag off by default (V) | V |
| overlay to OSPF | `ospf.redistribute.overlay_prefixes` | switch V, default assumption 22 |
| overlay to BGP | `bgp.redistribute.overlay_prefix` | switch V, default assumption 22 |
| OSPF to BGP | `bgp.redistribute.ospf`, metric 20 | V, default assumption 22 |
| BGP to OSPF | `ospf.redistribute.bgp` | V, default assumption 22 |
| connected to BGP | `bgp.redistribute.connected` | V, default assumption 22 |

Constraints applied during redistribution: only the best route of a prefix from the table is redistributed (V); a received prefix overlapping a locally connected subnet is not advertised into OFC and is not redistributed into BGP (V); OSPF and BGP may both be on (V).

### Preference and selection (per segment, per prefix)

1. **Candidate set:** all learned routes for the destination that cover it; unreachable routes are kept but not selectable (V: reason codes).
2. **Order of operations (V):** longest prefix, then local connected, local static (Preferred), local NSD static, remote NSD static, remote Edge routes, Partner Gateway static, then dynamic routes. A non-preferred static WAN route loses to a via-VeloCloud route (V).
3. **Dynamic routes:** lower `preference` wins. Defaults (V): OSPF O 1001, O IA 1002, eBGP 1003, iBGP 1004, NSD routes 997 to 1000; OE1 and OE2 are much larger (printed values VERIFY: simulator uses an assumption). Pins start at 1 (V). Without DCC the Orchestrator uses 512 as reference (V, not modelled further).
4. **Tie-breakers (V):** protocol attributes (BGP local preference higher, AS path shorter, metric lower; OSPF type order O, O IA, O E1, O E2, then lower metric), then Logical ID of the next-hop device, then route age (oldest wins). Order inside the BGP attributes is assumption 24.
5. **Output:** an ordered list per prefix (first = best path) plus a `lost_reason` per loser using the documented codes (LR_NP_ROUTE_TYPE, LR_BGP_LOCAL_PREF, LR_BGP_ASPATH_LEN, LR_BGP_METRIC, LR_EXT_OSPF_INTER, LR_EXT_BGP_RT, LR_EXT_METRIC_TYPE, LR_EXT_METRIC_VAL, LR_CONN_FLAG, and so on).
6. **Overlay Flow Control:** Preferred VPN Exits for a subnet override the default order of the exits (V: edit is pushed to relevant Edges); pinned routes get preference from 1 (V).

## 11. Reachability computation

For a question "can host in A reach prefix P (or host H) in B, in segment S?":

1. **Source side:** find the Edge of A and the best route for H in S (section 10, steps 1 to 5). No route: result `no_route` with the stage that stopped it (see step 4).
2. **Route origin:** follow the best route's `owner_edge`. If it is a local route, deliver to the LAN; if overlay, continue.
3. **Path:** ask the tunnel model (section 8) for a path from the Edge to the owner (direct dynamic tunnel, Hub, Gateway) and check `reachable` (PR_REACHABLE or PR_UNREACHABLE) (V).
4. **Explain the failure** by walking the route's life in the same order as the lesson 8 diagnostic: (a) owner Edge has the prefix (neighbor Full or Established, route learned), (b) advertised (Advertise, advertise true and reachable yes, flags), (c) not filtered (inbound filter permit, no overlap with a connected subnet), (d) redistributed at the destination LAN (overlay switches), (e) best route selection and its `lost_reason`, (f) the path is built. The first failing step is the answer; steps (c) overlap and (d) defaults use section 12 assumptions.
5. **Return traffic:** repeat from B to A. Asymmetric reachability is a normal result.
6. **Firewall and business policy:** applied after route selection (assumption 4 of section 7).

Outputs: `reachable` yes or no, the route list with `lost_reason`, the path, and the first failing stage as a string ("not advertised: External OSPF flag off").

## 12. Added assumptions (step 10)

21. Route preference for OE1, OE2 and BGP uplink: the printed values (about 1001006 to 1001008) are VERIFY; the simulator orders them after every non-external type and keeps OE1 ahead of OE2.
22. Default states of Overlay Prefix (BGP), Overlay Prefixes (OSPF), Connected Routes (BGP), OSPF into BGP, BGP into OSPF are all off, until observed.
23. Route filter default action when no rule matches is permit; rules are evaluated top to bottom, first match wins.
24. Between two BGP routes the Edge applies local preference, then AS-path length, then metric (standard BGP order; the reason codes do not state it).
25. A local preference set on one Edge influences other Edges only through the tie-breaker "protocol attributes" for overlay routes of equal preference; OFC exit order and pins override it.
26. Without DCC, preferences are computed in the simulator with the same defaults as with DCC.
27. Loop avoidance is limited to the documented rules (best route only, overlap suppression); the simulator reports a "possible redistribution loop" warning when OSPF and BGP redistribute into each other on two Edges of the same segment, flagged as a heuristic.
28. Static routes apply to the Edge's Global Segment unless the topology says otherwise; routes do not cross segments.
29. The summarization table (Subnet, No Advertise, Tag, Metric Type, Metric) affects only OSPF external routes; summarizing OSPF routes into the overlay is not modelled.
30. An unreachable BGP neighbor or OSPF neighbor (not Full) removes the routes learned through it immediately; BFD timing is not modelled.
31. A hybrid Hub and Branch Edge uses the Branch Profile's advertise flags (V) and the Hub's other settings.
32. The Gateway reflects a route only to Edges whose Profile and segment have it enabled for Cloud VPN; "the Gateway reflects based on Profile settings" is verified but its segment scoping is not.

## 13. Mock routing screens

| mock screen | data it needs | real screen |
|---|---|---|
| Edge > Device > Routing & NAT > Static Route Settings | static routes with Preferred, Advertise, probe | Configure > Edges > Device |
| Profile > Device > Routing & NAT > OSPF | areas, redistribution, route filters, summarization | Configure > Profiles > Device |
| Edge or Profile > BGP | local ASN, neighbors, Advanced Settings, Filter List | Routing & NAT > BGP |
| Overlay Route Control | community and AS path action | Routing & NAT > Overlay Route Control |
| Configure > Overlay Flow Control | Preferred VPN Exits, Global Advertise Flags, Routes List, Refresh Routes | same |
| Diagnostics: Route Table Dump, List Routes per Prefix | the ordered candidate list, Type, Cost, Preference, Reachable, Lost Reason | Remote Diagnostics |
| Diagnostics: OSPF and BGP tests | neighbor state, learned, advertised and redistributed routes | Show OSPF Neighbors, List OSPF Routes, Show BGP Summary, List BGP Routes and so on |
| Monitor > Routing | BGP Edge Neighbor State, Gateway Route Table | Monitor > Routing |

All read-only first, fed by the route computation. Marking and column layout beyond the documented field lists are assumptions.

## 14. Modelable now versus blocked on VERIFY (routing)

- **Modelable now:** order of operations, route type order, per-type default preferences for OSPF O and IA and BGP, tie-breakers, local preference, AS path and metric comparisons, filter actions, static route Preferred and Advertise, advertise flags with the two documented defaults, transit metrics, OFC exit edits and pins, Route Table Dump and test outputs for documented fields, reachability walk, OSPF and BGP neighbor states.
- **Blocked or assumed:** defaults of the five redistribution switches, filter default action and order, OE1 and OE2 preference numbers, cross-segment routing, per-segment static routes, loop behaviour beyond the documented rules, the name and default of the summarized-route suppression setting, OSPF timers and cost defaults, limits for OSPF, effect of overlay-side local preference, Gateway route table content, Monitor > Routing columns.

## 15. Build plan

| # | phase | inputs | outputs | tests | supports |
|---|---|---|---|---|---|
| 1 | Data model and validation | topology YAML (sections 3, 8 and 10), validation rules (section 5 plus routing rules: ASN set when BGP on, filters referenced exist, OSPF only on Global Segment, routed interface for active OSPF) | Go types, loader, validator reporting all errors as `file: message` | table tests per rule, golden invalid files, unknown-key rejection | all sections; labs |
| 2 | Effective configuration | validated topology | `EffectiveConfig` per Edge with `source` per setting (section 2), including routing settings | override and inheritance tests, Profile-switch side effects | Section 4 |
| 3 | Tunnels and paths | effective config, gateways, links | static and dynamic tunnels, List Paths data, path selection (section 8) | tunnel count, 16-tunnel cap, 180 s teardown with injected clock, Hub failover | Sections 2 and 5 |
| 4 | Routing and reachability | effective config, tunnels, LAN routers | per-segment tables with `lost_reason`, reachability with failing stage (sections 10 and 11) | selection tests for each documented reason code, flag defaults, redistribution switches, overlap suppression, property test: result independent of route learning order | Section 6 |
| 5 | Flow tracer | routes, paths, business policy, firewall | step-by-step trace of one flow (classification, route, path, link, remediation) | scenario fixtures matching the lesson diagnostics; deterministic with fixed seeds | Sections 3, 4, 5, 6 |
| 6 | Mock VCO screens | all of the above via an API | read-only screens in section 6 and 13 tables, Diagnostics outputs | e2e comparing screen fields to the computed data; documented field names only | all sections |
| 7 | Labs and scenarios | topologies, fault injection (neighbor down, flag off, filter wrong, link down), expected findings | `content/labs/*.yaml` scenarios with a check list and hints, validator for labs | each scenario solved by a scripted solution in CI; a scenario fails if an assumption changes its answer without a flag | Sections 3, 5, 6 (and later) |

**Status:** phases 1 and 2 were implemented in step 11 (section 16), phase 3 (tunnels and paths) in step 12 (section 17) and phase 4 (routing and reachability) in step 13 (section 18); phase 5 onward is not started.

Phases 1 and 2 need no further research; phase 4 starts from section 14's "modelable now" list and turns each section 12 assumption into a switch that a real-VCO observation can flip.

## 16. Implemented in step 11: topology format, validation and effective configuration (build plan phases 1 and 2)

Code: `backend/internal/sim` (pure Go, no DB or HTTP). Topology files: `content/topologies/<id>.yaml`, strict YAML, validated by the content loader, `make validate-content` and the startup check. API: `/api/topologies`, `/api/topologies/{id}`, `/api/topologies/{id}/edges/{edge}/effective-config`, `POST /api/topologies/{id}/what-if`. UI: `/sim`.

### Topology file

`version: 1`, `meta {id, title, summary, teaching: true, notes}`, `customer`, `segments`, `gateways`, `links`, `profiles`, `sites > edges`. Evolution is additive: later steps add OPTIONAL keys (tunnels, routing) so today's files stay valid under strict decoding.

| entity | fields |
|---|---|
| Profile | id, name, segments, `device` (setting id to value), `vlans`, `business_rules`, `firewall_rules` |
| Edge | id, name, role (branch or hub), profile, `interfaces`, own `vlans`, `overrides {device, vlans, business_rules, firewall {override, rules}}` |
| Interface | id, kind (wan, lan, routed, loopback), vlan, segments (empty = all of the Profile's), link, address, `override` |
| Business rule | id, segment, match, action (priority, network_service, steering, service_class, rate_limit, nat) |
| Firewall rule | id, match, action (allow, drop, reject, skip) |

Device settings are limited to a registry (`settings.go`) of settings documented as overridable: ARP Timeouts, BFD, ECMP (sections confirmed) and Wi-Fi Radio, Global IPv6, SNMP, RADIUS (section not confirmed, assumption 36), and Stateful Firewall (default off). Values are opaque scalars (assumption 33).

### Validation rules implemented (every finding reported with its path; errors fail, warnings are shown)

Unique slug ids per kind; teaching label required; version; segments (known type, at most the customer maximum, default 16 (V)); Profile segments exist and are not repeated; device settings known and typed; VLAN ids 1 to 4094, unique per Profile and per Edge, at most 32 per Edge (V); business rule segment known and part of the owner's Profile (V segment-aware); `internet_backhaul` only with destination internet (V); Service Class only for custom applications (V); `interface` and `wan_link` steering only on Edge-owned rules and only to an interface or link the Edge has (V); steering group values; Mandatory, Preferred or Available only with non-Auto modes; firewall protocol (V) and action; Reject and Skip need Stateful Firewall on (V), checked per Profile and per Edge after overrides; Edge firewall rules need the Override tick (V); every Edge has exactly one known Profile, one or more WAN interfaces; WAN interface needs a known link not used by another WAN interface; loopbacks are Edge-only and unique (V); LAN interfaces reference a VLAN the Edge has (assumption 40); interface segments exist and belong to the Profile (assumption 12); VLAN overrides only on Profile VLANs and never of `id` or `name` (V). Warnings: private wireless link (not a documented transport group), unused link, no Gateway in a site's region, more than 128 segments.

### Effective configuration

`EffectiveConfig(topology, edge)` returns per value: the value, its source (profile, edge-override, edge-only, customer, default, unset) with the Profile or Edge id, a `why` sentence and a **basis** (verified with fact sheet name, assumption N, or not modelled with the reason). Implemented: the layer merge (V), documented defaults (Stateful Firewall off, Premium Service on, 16 segments, interfaces on all segments, Auto steering), business rule sections "Rules From Profile" and "Edge Overrides" with supersession by identical match (V), Private-segment rewrite of Multipath to Direct (V), firewall Edge rules before Profile rules (V), VLAN inheritance with locked name and ID (V), loopbacks (V). Returned as `not modelled`: rule evaluation order, firewall versus business policy order, segment-to-segment traffic, per-segment firewall scope, how the real VCO marks an override, the complete overridable list, settings with no documented default (returned `unset`).

`WhatIf(topology, change)` supports set or remove a Profile setting, set or remove an Edge override, and assign an Edge to another Profile. It never mutates the input and returns, per Edge, `changed`, `kept-override` (assumption 5) or `unchanged`, the item diff, notes with bases, and validation issues the change introduces. A Profile switch removes VLANs the old Profile gave (V), matched by id (assumption 35), reverts interfaces tied to removed VLANs even when overridden (V), keeps other overrides and marks the BGP-filter removal and other override effects as not modelled.

### Citation convention (enforced by `rules_test.go`)

Every behaviour carries `// rule: <id>`; each id is registered in `rules.go` with its fact sheet and an exact row substring (the test checks the row exists with status VERIFIED or OBSERVED, or VERIFY for not-modelled rules) or an assumption number (the test checks it is listed here). Every registered rule must be cited.

### Added assumptions (step 11)

33. Setting values are opaque scalars: shown and compared, their meaning (units, ranges) is not modelled.
34. An Edge may define its own VLANs next to inherited Profile VLANs; the 32-VLAN limit counts both.
35. On a Profile switch an interface is tied to a removed VLAN when its VLAN id is not defined by the new Profile; a VLAN id both Profiles define is kept.
36. Settings whose Device tab section was not read are grouped as "Other (section not confirmed)".
37. Rules inside one group (Profile rules, Edge rules) are listed in authored order; this is a listing, not an evaluation order.
38. After a Profile switch the Edge's own rules are kept; rules on a segment the new Profile lacks are flagged.
39. Stateful Firewall is a Profile device setting (default off, documented) that an Edge may override (the pages say Edges inherit Stateful Firewall settings by default).
40. A LAN interface references a VLAN the Edge has (inherited or its own); wan, routed and loopback interfaces carry no VLAN.
41. An Edge business rule supersedes a Profile rule when both are on the same segment and have the same match (the guides say "an identical rule with new parameters" without defining identical).

## 17. Implemented in step 12: tunnels, dynamic lifecycle, path selection (build plan phase 3)

Code: `backend/internal/sim` (`overlay.go` types, `validate_overlay.go`, `tunnels.go`, `dynamic.go`, `timeline.go`, `pathselect.go`, `conditions.go`). API: `POST /api/topologies/{id}/tunnels`, `/timeline`, `/paths`. UI: `/sim/:topology/map`, `/monitor/:edge` (Links, Paths), `/paths`, `/timeline`, `/challenge`. Routing, redistribution and prefix reachability are NOT modelled (step 13): every path decision says reachability is assumed.

### Topology keys added (all optional)

| where | keys |
|---|---|
| top level | `clusters [{id, members, auto_rebalance}]`, `non_sdwan_destinations [{id, via gateway or edge, type, mode, tunnels, gateway, edge, segment}]` |
| gateway | `roles [data_plane, secure_vpn]` |
| link | `mtu` (at most 1500) |
| edge | `logical_id`, `gateways {primary, secondary}`, `nat_blocks_incoming_vcmp`, `overrides.cloud_vpn` |
| profile | `cloud_vpn {segments, branch_to_hub {enabled, hubs, backhaul_hubs}, branch_to_branch {enabled, via cloud_gateways or hubs_for_vpn, dynamic}, nsd_via_gateway, nsd_via_edge}` |

### Validation added

Gateway roles known; link MTU positive and not above 1500 (warning below 1300); Edge gateway assignment known and primary different from secondary; clusters (unique ids not equal to an Edge id, members are Edges listed once, warnings for fewer than two members, members in several clusters, mixed models); Hub lists name known Edges or clusters, without repeats (Hub order is unambiguous), Enable Branch to Hubs needs a Hub; Hubs for VPN needs Branch to Hub with Hubs; `via` required when Branch to Branch is enabled; Cloud VPN segments belong to the Profile; Non SD-WAN destination ids unique, type valid for its kind, via-Gateway mode and tunnel count within the documented cap (4, 2, 1, 1, Azure Virtual Hub 2), Gateway known and with the secure_vpn role when roles are listed, via-Edge destination with a known Edge and one service per Edge and segment; `nsd_via_gateway` never in an Edge override (Profile only); warnings when the link pairs of two Edges exceed the 16-tunnel cap.

### Tunnels (`ComputeTunnels(topology, conditions, clock)`)

| type | formed when | endpoints and paths | state |
|---|---|---|---|
| gateway | always, for the primary and secondary Gateway (V assignment by geolocation; derivation of the Gateway is assumption 44) | Edge and Gateway; one path per WAN link (public link to the public address, private link to the private address) | worst-link rule below |
| hub | Cloud VPN has a segment and Enable Branch to Hubs is on (V); one tunnel per entry of the Hub list; a cluster entry resolves to the member at logical ID modulo member count (V) | spoke and Hub Edge; same-type link pairs | as above; otherwise not-established with a reason code |
| dynamic | an entry exists for every Edge pair; built after traffic when Cloud VPN, Branch to Branch and Dynamic Branch to Branch are on for both Edges and the build stub says yes; torn down after 180 s of silence (V) | the two Edges; same-type link pairs | up, degraded, down, not-established or torn-down |
| nsd-gateway | the Profile lists it with Cloud VPN on; tunnel count from the topology up to the documented cap (V) | Gateway and destination; Hot-Standby tunnels after the first are standby (assumption 49) | up, down or not-established |
| nsd-edge | the destination is enabled for the builder Edge; one tunnel on its first public link | Edge and destination | as above |

A path state is the worse of its two links (up, degraded, down). A tunnel is up if any path is up, degraded if only degraded paths remain, else down. At most 16 paths per Edge-to-Edge or Edge-to-Gateway pair are kept (V cap). With the Orchestrator unreachable every state is `unknown` plus a not-modelled marker.

### Dynamic lifecycle and timeline

`Conditions.Traffic` tells a stateless request about history (seconds since first and last packet). The first packets use the standing path; the direct tunnel is built at once if the build stub allows (assumption 51); a tunnel silent for at least 180 s is torn down (assumption 52 fixes the boundary: 179 s keeps, 180 s tears down); later traffic restarts the cycle (assumption 53). The build stub is `auto` (blocked only by `nat_blocks_incoming_vcmp` on the peer, assumption 16), `always` or `never` (teaching switches, assumption 65); the real decision is not documented (not modelled). `Timeline(topology, conditions, events)` replays timed events (traffic start and stop, link, Hub and Gateway events, flush) on a virtual clock and returns every tunnel's state per frame plus the Hub failover state: local flows move to the secondary Hub when the primary goes down, peer flows are not moved back when it returns until flushed (both V); moving local flows back is not modelled.

### Path selection (`SelectPath`)

A flow is a source Edge, a destination Edge or NSD, an application and a traffic class (informational). The matched business rule gives the network service and the steering (first match, Edge rules above Profile rules: assumptions 1 and 3; no match is Multi-Path: assumption 2; Direct means no overlay path: assumption 63). Options in order: direct dynamic tunnel, static Hub tunnel to a destination that is a Hub (assumption 60), then the standing path (via Gateway for Cloud Gateways, via each Hub in order for Hubs for VPN: assumption 61), or the NSD tunnel set. The first usable option is chosen; each rejected option states why. Steering applies to the source Edge's first leg: Auto picks the first up path, else the first degraded one; Mandatory never leaves the chosen link or group and drops if it is down; Preferred stays while up, otherwise moves to another public link, then to a private link only if the public links are not up, and is not modelled for a private preferred link; Available stays while the link is up, whatever the SLA, and moves (as Auto) only if it is down. MTU: path MTU is the smaller of the two links of the first leg (default 1500, at most 1500), minus 97 to 112 bytes of VPN overhead. Not modelled: routes and prefixes, direct-versus-Hub preference beyond assumption 14, in-flight flows at teardown, load balancing, NSD MTU, Orchestrator-unreachable behaviour, SLA thresholds.

### Added assumptions (step 12)

42. Cloud VPN is off unless a Profile lists segments under `cloud_vpn` (the default state is not documented).
43. An Edge's `cloud_vpn` override replaces whole sub-sections, not single fields.
44. Without an explicit assignment, the primary Gateway is the first Gateway in the site's region, the secondary the next one in that region, else the first Gateway of another region.
45. Every Edge builds tunnels to both its primary and secondary Gateway, whatever Cloud VPN says.
46. A tunnel's paths are the pairs of same-type WAN links; a path is as healthy as the worse of its two links.
47. A public WAN link reaches a Gateway's public address, a private link its private address.
48. Cluster members keep their authored order; an Edge's logical ID is authored (default: its position in the file).
49. In Active/Hot-Standby the first tunnel is active and the others are established but standby.
50. A via-Gateway destination is built by the Gateway the topology names; a via-Edge destination by one tunnel on the Edge's first public WAN link.
51. The initiating Edge builds the direct tunnel at the moment of the first packet if the build decision allows it (no build delay).
52. Exactly 180 seconds of silence tears a dynamic tunnel down; 179 seconds keeps it.
53. After a teardown, new traffic starts the cycle again.
54. A dynamic tunnel needs Branch to Branch VPN and Dynamic Branch to Branch on for both Edges.
55. A degraded link is up but misses its SLA (a brownout); a down link is an outage; the numbers behind either are not modelled.
56. When several paths are equally healthy, Auto picks the first in a fixed order.
57. Transport Group steering follows the Mandatory, Preferred and Available rules of Interface and WAN Link steering; an empty strictness is treated as Available.
58. With Available, when the chosen link is down the flow moves to the path Auto would choose.
59. Routing attributes of Hub routes are assumed equal, so Hub order decides between Hubs.
60. For a flow to the Hub Edge itself, the static Hub tunnel is preferred over the Gateway path.
61. With Hubs for VPN the standing branch-to-branch path uses only Hubs (the 6.4 behaviour) in 7.0 too; with Cloud Gateways only the Gateway.
62. Link steering applies to the source Edge's first leg; the other legs are chosen as Auto.
63. A matched rule with network service Direct sends the flow outside the overlay, so no overlay path is chosen.
64. A link's MTU is its authored value (default 1500); the path MTU is the smaller of the two links of the first leg.
65. The build decision can be forced to always or never by the reader (a teaching switch).
66. Flows between two Edges over Cloud VPN are VPN (IPsec) traffic for the MTU calculation (97 to 112 bytes of overhead). The path MTU discovery nuance (public links share the minimum found toward the Gateways, private links get a result per tunnel) is not modelled: assumption 64 takes the smaller of the two links.
67. A path decision reflects the current link states, as if the documented one-minute hold before moving back to a preferred link had already elapsed (the hold needs history, and the API is stateless).
68. On-demand remediation (FEC, jitter buffer, NACK) is not modelled in path selection: with degraded links the decision only names the link the flow would use.

### Mock Monitor screens

Monitor > Edges > (Edge) has the tabs Links and Paths (names from `vco-overlay-ui` and `dmpo-vco-ui`). Links lists the Edge's WAN links with a Transport Groups view (Public Wired, Public Wireless, Private Wired) and the simulator's own state; Paths lists the Edge's paths to peers with the WAN link, the peer and the Mode (Static or Dynamic, from List Paths) and the simulator's own state. Columns that need numbers the model does not have (IP addresses, latency, jitter, loss, bandwidth) are not built. The real VCO's states (Stable, Unstable, BW_Measurement, Quiet) are not the simulator's (up, degraded, down, not established, torn down); the screen says so.

### Modelable now versus blocked (after step 12)

- **Modelled with a documented basis:** Gateway tunnels, Hub tunnels and Hub order, cluster assignment by logical ID, Cloud VPN per segment, Branch to Branch via Cloud Gateways or Hubs, the dynamic tunnel lifecycle with the 180 s teardown, Hub failover (local flows to the secondary Hub, peer flows not moved back), NSD tunnel counts and caps, the 16-tunnel cap, link steering with Mandatory, Preferred and Available, the VPN overhead and effective MTU.
- **Visible assumptions:** 42 to 68 above, notably the Cloud VPN default (42), the build decision stub (16, 65), direct tunnel preferred (14), degraded as a state only (55) and the hold and remediation not applied (67, 68).
- **Not modelled (VERIFY rows):** Edge behaviour with the Orchestrator unreachable, the real "secure tunnel possible" conditions, in-flight flows at teardown, direct-versus-Hub preference beyond assumption 14, Hub NAT, local flows moving back to the primary Hub, Preferred steering on a private link, SLA thresholds, load balancing, NSD MTU, route and prefix reachability (step 13).


## 18. Implemented in step 13: routing and reachability (build plan phase 4)

Code: `backend/internal/sim` (`routing.go` schema, `validate_routing.go`, `routes.go`, `routes_select.go`, `routes_lan.go`, `reachability.go`, `routing_assumptions.go`, `rules_routing.go`, `conditions_routing.go`). API: `POST /api/topologies/{id}/routes` and `/reachability`. UI: `/sim/:topology/routes`, `/sim/:topology/reachability`, `/sim/:topology/routing-challenge`. Same rules as before: nothing is shown without a basis (verified row, numbered assumption, or "not modelled"), and the topology is never changed by conditions or assumptions.

### Topology keys (all optional)

| key | meaning |
|---|---|
| `profiles[].routing.ospf` | `enabled`, `areas[{id}]` (Normal type), `redistribute {overlay_prefixes, bgp, default_route none/e1/e2, default_conditional, metric (20), metric_type (e2)}`, `summaries[{subnet, no_advertise, metric_type, metric, tag}]` |
| `edges[].routing.static_routes` | `id, segment (global), subnet, next_hop, interface (a WAN interface), cost, preferred, advertise, icmp_probe` |
| `edges[].routing.lan_routes` | what a LAN router would offer: `id, prefix, segment, protocol ospf/bgp, via (OSPF interface or BGP neighbor id), ospf_type intra/inter/e1/e2, metric, bgp {as_path, local_preference, med}` |
| `edges[].routing.ospf.interfaces` | `id, mode active/passive, area` |
| `edges[].routing.bgp` | `local_asn`, `neighbors[{id, ip, asn, type, segment, inbound_filter, outbound_filter, nsd}]`, `redistribute {overlay_prefix, connected, ospf, ospf_metric, default_route no/yes/conditional}`, `filters[{id, rules[{match {prefix, exact_match, metric, local_preference}, action permit/deny, set {local_preference, metric, as_path_prepend}}]}]` |
| `overlay_flow_control` | `advertise_flags {static, connected, external_ospf}`, `preferred_exits[{segment, prefix, exits}]`, `pins[{segment, prefix, exit}]` |
| `non_sdwan_destinations[].prefixes` | `prefix, cost, advertise` |

Connected subnets come from LAN and routed interface addresses. A nil switch (for example `overlay_prefixes`) takes the value of the request's Assumptions object (assumption 22); a value written in the topology wins.

### Validation (all errors at once, with paths)

Prefix syntax (IPv4, canonical, no host bits); unique ids (static routes, LAN routes, filters, neighbors, areas, summaries); no duplicate neighbor (same IP in a segment); segments exist and belong to the Profile; OSPF must be activated in the Profile before an Edge enables an interface; active OSPF only on routed interfaces (a switched LAN interface is passive only); OSPF on WAN or loopback interfaces is rejected as not modelled; OSPF only on the Global Segment; ASN ranges and iBGP/eBGP type agreeing with the ASNs; filter references exist; with iBGP to a Non SD-WAN destination an outbound filter must not set local preference; LAN routes reference an enabled OSPF interface or an existing neighbor (a passive interface is a warning); Overlay Flow Control exits are Edges and no (segment, prefix) is listed twice; static route next hops are inside a connected subnet unless a WAN interface is named; cost range 0 to 255.

### Conditions and assumptions

`Conditions` gains `neighbors_down` (`<edge>/<OSPF interface or BGP neighbor id>`) and `probes_down` (`<edge>/<static route id>`). `RoutingAssumptions` (request body, never stored) holds the ten selectable unverified values: the five redistribution switches (assumption 22, all off), filter default action (23, permit) and order (23, first match), the static and connected Global Advertise Flags (69, on) and the Logical ID direction (74, lower). Every output carries the list with defaults, current values, the assumption number, and which values the result used.

### Route selection (`ComputeRoutes`)

Per Edge and segment a table with every candidate: source (connected, local static, NSD local/remote, remote connected/static/dynamic, OSPF, BGP), type (O, O IA, O E1, O E2, eBGP, iBGP), preference (1001, 1002, 1003, 1004; E1 and E2 "not modelled"), reachability, Lost Reason, reason and basis. Order: unreachable candidates last; then the order of operations (connected, local static with Preferred, local NSD, remote NSD, remote Edge routes, dynamic); a static route without Preferred is a last resort (assumption 70); inside the dynamic pool: pinned routes, Preferred VPN Exits, then OSPF O, O IA, eBGP, iBGP, E1, E2; then BGP local preference, AS-path length, MED (assumption 24), OSPF metric, Hub order, Logical ID (assumption 74); equal in every documented key is flagged "not modelled" (route age). The reason of a route is relative to the next preferred route, with the documented code where one exists. A routed route of the Edge against a remote Edge connected or static route follows the order of operations and carries a not-modelled flag (the rows conflict).

### Advertisement and redistribution

Into the overlay: connected and static routes by flag and Advertise box (one static route per network), OSPF automatically (E1 and E2 only with the External OSPF flag, off by default), BGP from an Established neighbor (advertise True, reachable Yes); everything needs Cloud VPN on the segment (stated for static routes, assumption 82) and a prefix that overlaps a connected subnet is held back. To the LAN: overlay prefixes into OSPF (external, E2, metric 20) or BGP, BGP into OSPF, OSPF and connected into BGP, default routes, OSPF summaries, BGP outbound filters. Only the best route of a prefix is redistributed.

### Reachability (`Reachability`)

Six stages in the order of the Section 6 diagnostic (assumption 83): known at the owner, advertised, filtered or held back, redistributed into the source's LAN (only when the source runs OSPF or BGP), winner at the source, path built. The first failing stage and a reason are returned, with per-owner verdicts (several exits for one prefix), the source's candidate routes, the path (from step 12), the assumptions in effect, and `depends_on`: every assumption switch whose flip would change the answer. Cross-segment questions return not modelled.

### Added assumptions (step 13)

69. The Global Advertise Flags for static and connected routes default to on when the topology does not set them (only the External OSPF and NSD iBGP defaults are documented).
70. A static route without Preferred is a last resort behind every other candidate for the same prefix, including routes learned from OSPF or BGP on the Edge (the sheet states it against overlay routes).
71. A static route without a Cost has cost 0; among static routes of the same type for one prefix the lower cost wins.
72. A static route whose ICMP probe is down is unreachable with the code SLA_STATE_DOWN.
73. A passive OSPF interface forms no adjacency, so no route is learned through it.
74. The Logical ID tie-break between Edges prefers the lower Logical ID by default; the direction is a selectable assumption.
75. A BGP route without a local preference compares as 100 (the common BGP default).
76. A Non SD-WAN destination built by an Edge is a local NSD static route on that Edge and a remote NSD static route elsewhere; one built by a Gateway is a local NSD static route on every Edge that enables it.
77. An OSPF default route that is not Conditional is advertised whether or not the Edge has a default route.
78. BGP is authored per Edge (as if overridden); Profile-level BGP inheritance is not modelled.
79. Two prefixes overlap when one contains the other (or they are equal).
80. A summary without No Advertise is advertised as one external route with the summary's Metric Type and Metric (default E2 and 20).
81. A remote route keeps the attributes of the owning Edge's route and its next hop is that Edge.
82. An Edge advertises a route into the overlay only when Cloud VPN is enabled for the route's segment (stated for static routes, assumed for every other route).
83. The reachability stages follow the Section 6 diagnostic order (our teaching method): known, advertised, filtered, redistributed, winner, path. The redistribution stage applies only to a source Edge that runs OSPF or BGP.
84. An origin route matches a queried prefix when it is the same or a less specific prefix; the most specific one is used.

### Modelable now versus blocked (after step 13)

- **Modelled with a documented basis:** the order of operations, longest prefix, the four default preferences and the OSPF type order, BGP attribute comparison and Lost Reason codes, static route Preferred, Advertise and ICMP probe, the External OSPF flag, OSPF to the overlay automatically, BGP advertise/reachable, overlap suppression, best-route-only redistribution, OSPF external defaults (metric 20, E2), the Overlay Prefix(es) and BGP/OSPF switches, OSPF summaries for external routes, filters (prefix match, Permit or Deny, local preference, metric, prepend), Preferred VPN Exits and pins, unreachable overlay routes (PR_UNREACHABLE).
- **Visible assumptions:** 22, 23, 24, 69 to 84 (switchable: 22, 23, 69, 74).
- **Not modelled (VERIFY rows):** cross-segment routing, E1 and E2 preference numbers, connected and static routes into OSPF, whether a routed route beats a remote Edge route, the transit metric, route age and the Logical ID direction, OSPF filters, the other Global Advertise Flags, Eligible VPN Exits, redistribution loops, what a passive OSPF interface does beyond the support statement, how the overlay changes BGP attributes.
- **Not built:** the flow tracer and labs (step 14), Partner Gateway routes, Distributed Cost Calculation versus Orchestrator preferences (assumption 26), BGP community and AS-path exclude, per-neighbor default-route options, IPv6.

## 19. Implemented in step 14: the flow tracer and labs (build plan phases 5 and 7)

Code: `backend/internal/sim` (`trace.go`, `trace_stages.go`, `trace_assumptions.go`, `policy_eval.go`, `rules_trace.go`). API: `POST /api/topologies/{id}/trace`. UI: `/sim/:topology/trace`. Labs are described in section 20.

### The trace (`TraceFlow`)

One flow (source Edge, optional LAN host IP or interface, destination IP, prefix, `edge:<id>` or `nsd:<id>`, segment, application, protocol, port, DSCP, packet size) is followed through eight stages. Stage order is the simulator's presentation order (assumption 91); the real order of the policy, firewall and routing decisions is not documented. Every stage returns `input`, `decision`, `reason`, a basis, "not modelled" markers, the matching lesson and the mock screen to inspect.

| # | stage | models | marks "not modelled" |
|---|---|---|---|
| 1 | Segment and source interface | the segment must be in the Edge's Profile (V); the interface that contains the source IP, or the named one; an interface with no segment list is in every Profile segment (V) | a flow with no source host is LAN traffic passing through the Edge (assumption 87) |
| 2 | Business policy | every rule of the segment evaluated in precedence order, the first match wins, superseded Profile rules listed as skipped (V supersession); private-segment rewrite (V) | the evaluation order beyond "highest precedence" (assumptions 1, 3 with a switch); the default when no rule matches (continues as Multi-Path by assumption 2) |
| 3 | Firewall | every rule evaluated independently; Edge override rules before Profile rules (V); Allow, Drop, Reject, Skip (V); Reject and Skip need Stateful Firewall (V); the Edge firewall applies to traffic passing through (V) | the order and the no-match action (assumptions 85 and 86, both switches); firewall versus business policy versus routing order |
| 4 | Route lookup | `ComputeRoutes` and `Reachability`: longest prefix, order of operations, selected route, candidates and Lost Reasons | everything routing already marks (section 18) |
| 5 | Path selection | `SelectPath` with the matched policy: direct tunnel, Hub, Gateway, Non SD-WAN, rejected options with reasons, link steering Mandatory, Preferred, Available | steering details already marked in section 17 |
| 6 | Remediation | the documented conditions of FEC, jitter buffer (7 ms, 5 ms) and NACK, Premium Service (V); which are candidates for the flow (assumption 89) | whether any is active now, and the loss thresholds |
| 7 | Overhead and MTU | 97 to 112 bytes for VPN traffic, 1500 maximum, fragmentation when too large (V) | Non SD-WAN tunnels |
| 8 | Return path | the reverse path decision; with Stateful Firewall on, a differing kind of path is blocked (V statement, criterion assumption 88) | return traffic from outside the SD-WAN |

Verdicts: delivered, blocked (stage N), unreachable (stage N), not modelled (stage N). Business policy cannot block on its own: a rule blocks only through Mandatory link steering to an unavailable link (V). The result also carries `depends_on`: every assumption switch whose flip changes the verdict.

### Added assumptions (step 14)

85. Firewall rules are evaluated first match wins in list order (Edge override rules before Profile rules; Skip moves to the next rule); the evaluation order is not documented, so last match is a selectable alternative.
86. When no firewall rule decides, the flow is allowed; the default action is not documented and is a selectable assumption.
87. A flow with no source IP and no source interface is treated as LAN traffic passing through the Edge, so policy and firewall apply.
88. With Stateful Firewall on, a flow counts as asymmetric when the forward and reverse path decisions choose different kinds of option.
89. Remediation candidates are classified from the flow: TCP is a NACK candidate; UDP real-time traffic (traffic class real_time) is an FEC and jitter buffer candidate.
90. A business or firewall rule matches the flow's own fields: addresses and prefixes by containment, ports as a number, list or range, "internet" for a destination outside every prefix of the topology; a rule that needs a field the flow does not carry does not match.
91. The trace presents segment, policy, firewall, route, path, remediation, MTU and return path in that order; the real order of the policy, firewall and routing decisions is not documented.

### The business policy and routing joins (step 14)

Stage 2 and `SelectPath` share one matcher (`policy_eval.go`): a rule matches the flow's application, source, destination (a keyword, address or prefix), protocol, port and DSCP (assumption 90). Stage 4 adds lesson 8's check 4 to the route lookup: an Edge that runs OSPF or BGP toward its LAN must hand the overlay route to its LAN routers, otherwise the flow never reaches the Edge (assumption 83). Business policy cannot block on its own: the only documented way a rule stops a flow is Mandatory link steering to a link that is down (stage 5).

## 20. Labs (step 14)

Files: `content/labs/<id>.yaml` (id equals the file name), strict YAML, validated at startup and by `make validate-content` together with every other content error.

```yaml
id, title, summary, difficulty (1-3), estimated_minutes, topology
goals: [lesson ids]                      # the lessons the lab reinforces
screens: [{label, path}]                 # mock screens to inspect, relative to /sim/<topology>/
fault: {conditions: {...}, assumptions: {switch: value}, patches: [{op, path, value}]}
symptom: text                            # must say so if the lab relies on assumptions
flow: {...}                              # the flow in question (a FlowSpec)
relies_on_assumptions: [numbers]
tasks:
  - {id, type: inspect|diagnose|predict, prompt, probe: {...}, choices: [{id, text, value}], explanation}
  - {id, type: fix, prompt, probe: {...}, equals: value, explanation}
hints: [{task?, text}]                   # revealed one at a time
fixes: [{id, title, description, patches: [...]}]   # allowed fixes, wrong ones included
```

Patches use `set`, `remove` or `append` on a path such as `edges[spoke-1-edge]/overrides/business_rules[x]/action`; `[id]` picks the array element with that id and `edges[<id>]` finds an Edge in any site. They are applied to a copy; unknown fields and wrong types are errors. Probe kinds: `trace_verdict`, `trace_blocked_stage`, `trace_path_kind`, `trace_path_via`, `trace_steered_link`, `trace_matched_rule`, `trace_matched_owner`, `trace_mtu_min`, `trace_mtu_limiting`, `trace_mtu_fit`, `path_rejected_code`, `reach_first_failing`, `reach_outcome`, `route_field`, `tunnel_state`, `tunnel_reason`, `timeline_peer_hub`, `timeline_local_hub` (timeline probes start from a healthy network and apply their own events).

Validation (all reported together): fields and ranges; lessons, topology and assumption numbers exist; the fault and every fix apply and leave a valid topology; every task has a probe; a choice task has at least two choices and EXACTLY ONE whose value equals what the engine computes (two correct root causes, or none, is an error); a fix task's check must fail in the faulty state and some allowed fix must make it pass; no answer may depend on a not-modelled result; every assumption an answer's evidence cites, and every switch whose flip would change the answer, must be listed under `relies_on_assumptions` and the symptom must mention assumptions.

Progress: table `lab_progress` (status started, completed or skipped, attempts, hints used, last attempt, completion). Completed is never downgraded; orphaned lab ids are ignored; nothing is locked and labs are never required.

### The six labs

| lab | topology | reinforces | screens |
|---|---|---|---|
| no-direct-tunnel | hub-and-spoke-demo | ov-dynamic, ov-cloud-vpn | map, Monitor > Edges > Paths, path explainer, timeline |
| hub-down | hub-and-spoke-demo | ov-hubs, ov-diagnosis | map, path explainer, timeline |
| branch-b-cannot-reach | routing-demo | rt-diagnosis, rt-redistribution, rt-bgp | reachability explainer, Route Table Dump |
| which-route-wins | routing-demo | rt-redistribution, rt-bgp, rt-ofc | Route Table Dump, reachability explainer |
| video-wrong-link | hub-and-spoke-demo | cfg-business-policy, cfg-overrides, dmpo-steering | Edge Business Policy tab, path explainer |
| mtu-ipsec (diagnose only: no documented fix) | hub-and-spoke-demo | arch-mtu-overhead | path explainer, Monitor > Edges > Links |

### Modelable now versus blocked (after step 14)

- **Modelable now and modelled:** everything in sections 16 to 19, followed end to end for one flow, plus guided practice on it.
- **Blocked on VERIFY rows (returned as not modelled, never asked in a lab):** the business-policy default when no rule matches, firewall order and default, firewall versus policy versus routing order, local flows after a Hub returns, Orchestrator unreachable, Non SD-WAN MTU and return traffic, whether remediation is active, route age and the Logical ID direction, cross-segment routing, Gateway route reflection details.
- **Not built:** Partner Gateway routes, DCC versus Orchestrator preferences, BGP community and AS-path exclude, IPv6, per-flow load balancing, a lab editor.


## 21. Implemented in step 15: capstone, self-check, observation workflow

No new engine behaviour and no new assumption number. What was added around the engine:

- **Topology `capstone-network`** (`content/topologies/capstone-network.yaml`, teaching, invented): a Data center site with a Primary and a Secondary Hub, three branch sites on ONE Profile (North runs OSPF to its LAN router, South runs BGP, East has a static route and an Edge override), two WAN links per branch (Internet and MPLS), Cloud VPN (Branch to Hub with both Hubs, Branch to Branch via the Hubs, dynamic allowed), one Non SD-WAN Destination via Gateway, a Profile business rule per application and a device setting (BFD) that East overrides. It uses only keys earlier steps already accept.
- **Three capstone labs** in the existing lab format (`content/labs/capstone-1-monday-morning.yaml`, `capstone-2-hub-incident.yaml`, `capstone-3-change-request.yaml`); the validator proves every fix passes its check and every choice task has exactly one correct answer, as for the other labs.
- **Two read-only probe kinds** for labs, `effective_value` and `effective_source` (`Probe.Edge` and `Probe.Field` = device setting id): they read `EffectiveConfig` and add no behaviour.
- **Lab `debrief`** (optional list of `{finding, where, unverified, note}`): where each finding would be found in the real VCO. `where` must be wording of a VERIFIED or OBSERVED fact-sheet row (the content validator checks); a location the guides do not confirm is written with `unverified: true` and a `note`, and is shown as "unverified path". The debrief names the root causes, so `GET /api/labs/{id}/debrief` serves it only once the lab is completed or skipped.
- **Self-check** (`/sim/self-check`, `GET /api/self-check`, `POST /api/self-check/grade`, `POST /api/self-check/screens/{task}/check`): 12 questions drawn from the quiz bank (two per section, distinct topics where possible), graded with the quiz grading code, plus three "read this screen" tasks that are the tasks of `content/labs/self-check-screens.yaml` (a lab-format file that the lab endpoints never list or serve). The server stores nothing for any of these calls; the checklist and the last result live in the browser (localStorage).
- **Goal coverage** on `/progress` (`frontend/src/progress/goals.ts`): four goals (read an Edge's configuration, how traffic flows, how tunnels form, how routes flow), each shown from lesson progress, best quiz scores, weak topics, labs and the last self-check. Informational; nothing reads it.
- **Observation workflow:** `content/reference/observations/README.md` (template), `docs/HOW-TO-ADD-AN-OBSERVATION.md`, `make observations-report` (`docs/OBSERVATION-PRIORITIES.md`), the validators in `internal/content/observations.go`, `make course-map` (`docs/COURSE-MAP.md`).

### Capstone claims that are not VERIFIED or OBSERVED, by assumption number

Every answer in the three labs that rests on an assumption is listed in the lab's `relies_on_assumptions`, said in its symptom text, and shown as a badge when the task is checked.

| lab | assumptions it rests on | what they are about |
|---|---|---|
| capstone-1-monday-morning | 1, 3, 22, 32, 62, 67, 81, 83, 86, 90 | rule order and Edge-over-Profile precedence (1, 3, 90); link steering on the first leg and current link states (62, 67); the redistribution defaults and the stage order of the reachability explainer, where the redistribution stage applies to an Edge running OSPF or BGP (22, 83); Gateway reflection and what a remote route keeps (32, 81); firewall default with no matching rule (86) |
| capstone-2-hub-incident | 32, 46, 59, 61, 81 | tunnel paths are same-type link pairs and a tunnel stays up while one is up (46); Hub order decides (59); Hubs for VPN uses only Hubs (61); reflection (32, 81). The effect of a degraded link on traffic is deliberately not asked (remediation is not modelled) |
| capstone-3-change-request | 33 | device setting values are opaque scalars |
| self-check screen tasks | 1, 3, 32, 59, 61, 81, 90 | as above |
