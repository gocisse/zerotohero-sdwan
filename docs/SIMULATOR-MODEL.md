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
| Customer | id, name, `max_segments` (default 16), `premium_service` (default true), `stateful_firewall` (default false) | defaults are VERIFIED (segments, dmpo, firewall sheets) |
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
| 2. Tunnel formation | static tunnels (Edge to Gateway, branch to Hub), dynamic branch to branch (on demand, torn down after 180 s silence), NSD tunnels; see section 8 | cloud-vpn, hubs-and-branches, non-sdwan-destinations, tunnel-types, vcmp-tunnels, gateway-details | further build conditions beyond "secure overlay multi-path tunnel possible"; Edge-as-Hub inbound ports; NAT cases; Edge behaviour when the Orchestrator is unreachable |
| 3. Flow path selection | which peer path (Gateway, Hub, direct, NSD) and which WAN link a flow takes; see section 8 | overlay-paths, business-policy, dmpo-steering, tunnel-types | rule evaluation order; default action; Edge vs Profile rule interleaving; direct-tunnel preference over Hub/Gateway path; route-type precedence (step 10) |
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
17. Edge Hub inbound requirement is UDP 2426 (not read on a page).
18. In-flight flows move to the Gateway or Hub path without loss when a dynamic tunnel is torn down.
19. A cluster's member for an Edge is stable between rebalances; Cluster Score is a simple average of CPU, memory and tunnel use when simulated.
20. Edges keep forwarding on last-known configuration when the Orchestrator is unreachable (simulator default, not documented).

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

Phases 1 and 2 need no further research; phase 4 starts from section 14's "modelable now" list and turns each section 12 assumption into a switch that a real-VCO observation can flip.
