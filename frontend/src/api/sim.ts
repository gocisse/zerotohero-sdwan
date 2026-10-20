import { getJSON, sendJSON } from './client'

export type BasisStatus = 'verified' | 'assumption' | 'not-modelled'

/** The evidence behind a value: a fact sheet (verified), a numbered assumption, or a "not modelled" marker. */
export interface Basis {
  status: BasisStatus
  rule: string
  ref: string
  text: string
}

export interface SimSource {
  kind: 'profile' | 'edge-override' | 'edge-only' | 'customer' | 'default' | 'unset'
  id?: string
  name?: string
}

export interface NotModelled {
  rule: string
  item: string
  reason: string
  basis: Basis
}

export interface Issue {
  level: 'error' | 'warning'
  path: string
  message: string
}

export interface TopologySummary {
  id: string
  title: string
  summary: string
  teaching: boolean
  sites: number
  edges: number
  profiles: number
  warnings: Issue[]
}

export interface SimVlan {
  id: number
  name: string
}

export interface SimInterface {
  id: string
  kind: 'wan' | 'lan' | 'routed' | 'loopback'
  vlan?: number
  segments?: string[]
  link?: string
  address?: string
  override?: boolean
}

export interface SimEdge {
  id: string
  name: string
  model?: string
  role: 'branch' | 'hub'
  profile: string
  interfaces: SimInterface[]
  vlans?: SimVlan[]
  overrides: { device?: Record<string, unknown>; business_rules?: { id: string; match: { application?: string } }[] }
  logical_id?: number
  nat_blocks_incoming_vcmp?: boolean
  routing?: {
    static_routes?: { id: string; subnet: string; icmp_probe?: boolean }[]
    lan_routes?: { id: string; prefix: string; protocol: 'ospf' | 'bgp'; via: string; segment?: string }[]
    ospf?: { interfaces?: { id: string; mode: 'active' | 'passive' }[] }
    bgp?: { local_asn: number; neighbors?: { id: string; ip: string; asn: number; segment?: string }[] }
  }
}

export interface SimSite {
  id: string
  name: string
  region: string
  edges: SimEdge[]
}

export interface CloudVpn {
  segments?: string[]
  branch_to_hub?: { enabled: boolean; hubs?: string[]; backhaul_hubs?: string[] }
  branch_to_branch?: { enabled: boolean; via?: 'cloud_gateways' | 'hubs_for_vpn'; dynamic?: boolean }
  nsd_via_gateway?: string[]
  nsd_via_edge?: string[]
}

export interface SimProfile {
  id: string
  name: string
  description?: string
  segments: string[]
  device?: Record<string, unknown>
  cloud_vpn?: CloudVpn
  business_rules?: { id: string; match: { application?: string } }[]
}

export interface SimGateway {
  id: string
  region: string
  roles?: string[]
}

export interface SimLink {
  id: string
  type: 'public' | 'private'
  medium: 'wired' | 'wireless'
  up_mbps: number
  down_mbps: number
  mtu?: number
}

export interface SimCluster {
  id: string
  name?: string
  members: string[]
}

export interface SimNsd {
  id: string
  name?: string
  via: 'gateway' | 'edge'
  type: string
  mode?: string
  tunnels?: number
  gateway?: string
  edge?: string
  segment?: string
}

export interface SimSegment {
  id: string
  name: string
  type: 'regular' | 'private'
}

export interface Topology {
  version: number
  meta: { id: string; title: string; summary: string; teaching: boolean; notes?: string }
  customer: { id: string; name: string }
  segments: SimSegment[]
  gateways: SimGateway[]
  links: SimLink[]
  profiles: SimProfile[]
  sites: SimSite[]
  clusters?: SimCluster[]
  non_sdwan_destinations?: SimNsd[]
}

export interface SettingDef {
  id: string
  label: string
  section: string
  kind: 'bool' | 'scalar'
  section_known: boolean
  has_default: boolean
  default?: unknown
  basis: Basis
}

export interface TopologyDetail {
  topology: Topology
  warnings: Issue[]
  settings: SettingDef[]
  /** Evidence for screen-level claims (tab names, transport groups, the build stub). */
  ui_bases: Record<string, Basis>
}

export interface EffectiveSetting {
  id: string
  label: string
  section: string
  value: unknown
  source: SimSource
  profile_value?: unknown
  overridable: boolean
  why: string
  basis: Basis
  also?: Basis[]
  not_modelled?: NotModelled
}

export interface EffectiveVlan {
  id: number
  name: string
  source: SimSource
  settings?: Record<string, unknown>
  name_id_locked: boolean
  basis: Basis
  also?: Basis[]
}

export interface EffectiveInterface {
  id: string
  kind: string
  vlan?: number
  segments: string[]
  segments_source: 'interface' | 'profile-default'
  link?: string
  address?: string
  override: boolean
  source: SimSource
  basis: Basis
  also?: Basis[]
}

export interface BusinessRule {
  id: string
  name?: string
  segment: string
  match: Record<string, string | boolean | undefined>
  action: {
    priority?: string
    network_service?: string
    steering: { mode?: string; group?: string; interface?: string; link?: string; strictness?: string }
    service_class?: string
    rate_limit?: boolean
    nat?: boolean
  }
}

export interface EffectiveRule {
  rule: BusinessRule
  owner: SimSource
  superseded_by?: string
  supersedes?: string[]
  effective_steering_mode: string
  rewritten?: boolean
  effective_network_service?: string
  basis: Basis
  also?: Basis[]
}

export interface SegmentPolicy {
  segment: { id: string; name: string; type: string; basis: Basis }
  edge_overrides: EffectiveRule[]
  from_profile: EffectiveRule[]
  precedence: NotModelled
  basis: Basis
}

export interface FirewallRule {
  id: string
  name?: string
  match: Record<string, string | undefined>
  action: string
}

export interface EffectiveFirewallRule {
  rule: FirewallRule
  owner: SimSource
  group: 'edge-override' | 'profile'
  basis: Basis
  also?: Basis[]
}

export interface EffectiveConfig {
  topology: string
  edge: { id: string; name: string; site: string; role: string; profile: string; profile_name: string }
  customer: { id: string; label: string; value: unknown; source: SimSource; basis: Basis }[]
  segments: { id: string; name: string; type: string; basis: Basis }[]
  device: EffectiveSetting[]
  vlans: EffectiveVlan[]
  interfaces: EffectiveInterface[]
  business_policy: SegmentPolicy[]
  firewall: {
    override: boolean
    stateful_firewall: EffectiveSetting
    rules: EffectiveFirewallRule[]
    evaluation_order: NotModelled
    group_priority: Basis
  }
  not_modelled: NotModelled[]
}

export type ChangeType =
  'set_profile_setting' | 'remove_profile_setting' | 'set_edge_override' | 'remove_edge_override' | 'assign_profile'

export interface Change {
  type: ChangeType
  profile?: string
  edge?: string
  setting?: string
  value?: unknown
}

export interface DiffItem {
  key: string
  kind: string
  label: string
  before: unknown
  after: unknown
  before_source?: SimSource
  after_source?: SimSource
  basis: Basis
}

export interface EdgeDiff {
  edge_id: string
  edge_name: string
  profile: string
  status: 'changed' | 'kept-override' | 'unchanged'
  items: DiffItem[]
  notes: { text: string; basis: Basis }[]
  reverted_interfaces?: string[]
}

export interface WhatIfResult {
  change: Change
  summary: string
  edges: EdgeDiff[]
  issues: Issue[]
  not_modelled: NotModelled[]
}

export const fetchTopologies = () =>
  getJSON<{ topologies: TopologySummary[] }>('/api/topologies').then((r) => r.topologies)
export const fetchTopology = (id: string) => getJSON<TopologyDetail>(`/api/topologies/${encodeURIComponent(id)}`)
export const fetchEffectiveConfig = (topologyId: string, edgeId: string) =>
  getJSON<EffectiveConfig>(
    `/api/topologies/${encodeURIComponent(topologyId)}/edges/${encodeURIComponent(edgeId)}/effective-config`,
  )
export const postWhatIf = (topologyId: string, change: Change) =>
  sendJSON<WhatIfResult>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/what-if`, change)

// ---- Overlay: tunnels, paths, timeline (step 12) ----

export interface TrafficSpec {
  a: string
  b: string
  first_packet_ago_s: number
  last_packet_ago_s: number
}

/** An overlay on the authored topology: never stored, never changes it. */
export interface Conditions {
  links_down?: string[]
  links_degraded?: string[]
  hubs_unreachable?: string[]
  gateways_unreachable?: string[]
  nsds_unreachable?: string[]
  orchestrator_unreachable?: boolean
  dynamic_build?: 'auto' | 'always' | 'never'
  traffic?: TrafficSpec[]
  /** "<edge>/<OSPF interface or BGP neighbor id>" */
  neighbors_down?: string[]
  /** "<edge>/<static route id>" */
  probes_down?: string[]
}

export type TunnelState = 'up' | 'degraded' | 'down' | 'not-established' | 'torn-down' | 'unknown'
export type TunnelType = 'gateway' | 'hub' | 'dynamic' | 'nsd-gateway' | 'nsd-edge'

export interface Endpoint {
  kind: 'edge' | 'gateway' | 'nsd'
  id: string
  name: string
}

export interface TunnelPath {
  local_interface: string
  local_link: string
  remote: string
  state: 'up' | 'degraded' | 'down'
  reason?: string
}

export interface Tunnel {
  id: string
  type: TunnelType
  mode: 'static' | 'dynamic'
  a: Endpoint
  b: Endpoint
  role?: string
  hub_index?: number
  paths: TunnelPath[]
  state: TunnelState
  reason_code: string
  reason: string
  basis: Basis
  also?: Basis[]
  notes?: { text: string; basis: Basis }[]
  not_modelled?: NotModelled[]
  initiator?: string
}

export interface TunnelSet {
  topology: string
  conditions: Conditions
  tunnels: Tunnel[]
  gateway_assignments: { edge: string; primary: string; secondary?: string; derived: boolean; basis: Basis }[]
  cluster_assignments: {
    spoke: string
    cluster: string
    member: string
    logical_id: number
    members: number
    basis: Basis
  }[]
  hub_orders: { edge: string; hubs: string[]; basis: Basis }[]
  not_modelled: NotModelled[]
}

export interface Flow {
  source: string
  destination: { kind: 'edge' | 'nsd'; id: string }
  application?: string
  traffic_class?: string
  segment?: string
}

export interface PathOption {
  kind: string
  label: string
  via?: string
  tunnel_ids: string[]
  usable: boolean
  chosen: boolean
  reason_code: string
  reason: string
  basis: Basis
  also?: Basis[]
}

export interface LinkChoice {
  mode: string
  strictness?: string
  configured: { mode?: string; group?: string; interface?: string; link?: string; strictness?: string }
  action: 'use' | 'move' | 'drop' | 'undetermined' | 'none'
  chosen?: TunnelPath
  reason: string
  basis: Basis
  also?: Basis[]
  not_modelled?: NotModelled[]
}

export interface MtuResult {
  path_mtu: number
  limiting: string
  overhead_min: number
  overhead_max: number
  effective_min: number
  effective_max: number
  basis: Basis
  also?: Basis[]
}

export interface PathDecision {
  flow: Flow
  reachability: { assumed: boolean; text: string; not_modelled: NotModelled }
  rule_id?: string
  rule_owner?: string
  network_service: string
  options: PathOption[]
  chosen?: PathOption
  why: string
  steering?: LinkChoice
  mtu?: MtuResult
  notes: { text: string; basis: Basis }[]
  not_modelled: NotModelled[]
  tunnels: Tunnel[]
}

export interface TimelineEvent {
  at_s: number
  type: string
  a?: string
  b?: string
  target?: string
}

export interface TimelineFrame {
  at_s: number
  label: string
  cause: 'event' | 'derived'
  event?: TimelineEvent
  states: { id: string; state: TunnelState; reason_code: string }[]
  changes: { id: string; from: TunnelState; to: TunnelState; reason_code: string; reason: string }[]
  failovers: {
    spoke: string
    hubs: string[]
    local_hub: string
    local_note?: string
    peer_hub: string
    peer_note?: string
    basis: Basis
    not_modelled?: NotModelled[]
  }[]
  first_packet?: { label: string; kind: string; tunnel_ids: string[]; reason: string; basis: Basis }
  basis?: Basis[]
}

export interface TimelineResult {
  topology: string
  tunnels: Tunnel[]
  frames: TimelineFrame[]
  not_modelled: NotModelled[]
}

export const postTunnels = (topologyId: string, conditions: Conditions) =>
  sendJSON<TunnelSet>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/tunnels`, conditions)
export const postTimeline = (topologyId: string, conditions: Conditions, events: TimelineEvent[]) =>
  sendJSON<TimelineResult>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/timeline`, { conditions, events })
export const postPath = (topologyId: string, flow: Flow, conditions: Conditions) =>
  sendJSON<PathDecision>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/paths`, { flow, conditions })

// ---- routing (step 13) ----

/** The ten selectable unverified values of the routing model. Sent with every request, never stored. */
export interface RoutingAssumptions {
  ospf_overlay_prefixes: boolean
  ospf_redistribute_bgp: boolean
  bgp_overlay_prefix: boolean
  bgp_connected: boolean
  bgp_redistribute_ospf: boolean
  filter_default_action: 'permit' | 'deny'
  filter_order: 'first_match' | 'last_match'
  advertise_static: boolean
  advertise_connected: boolean
  logical_id_tiebreak: 'lower' | 'higher'
}

export interface AssumptionSwitch {
  key: keyof RoutingAssumptions
  label: string
  kind: 'bool' | 'choice'
  choices?: string[]
  value: boolean | string
  default: boolean | string
  changed: boolean
  assumption: number
  text: string
  basis: Basis
  used: boolean
}

export interface SimRoute {
  id: string
  edge: string
  segment: string
  prefix: string
  source: string
  route_type?: string
  origin?: string
  origin_name?: string
  via?: string
  next_hop: string
  cost?: number
  preference?: number
  preference_note?: string
  local_preference?: number
  metric?: number
  as_path?: number[]
  reachable: boolean
  reachable_code?: string
  reachable_reason?: string
  selected: boolean
  rank: number
  lost_reason_code?: string
  reason: string
  basis: Basis
  also?: Basis[]
  not_modelled?: NotModelled[]
  notes?: string[]
}

export interface LanAdvertisement {
  edge: string
  protocol: 'ospf' | 'bgp'
  neighbor?: string
  segment: string
  prefix: string
  from_route?: string
  kind: string
  advertised: boolean
  reason: string
  ospf_type?: string
  metric?: number
  attributes?: string[]
  summarized_into?: string
  switch?: string
  basis: Basis
  also?: Basis[]
}

export interface Withheld {
  edge: string
  segment: string
  prefix: string
  source: string
  via: string
  stage: 'known' | 'filtered'
  reason: string
  basis: Basis
  also?: Basis[]
}

export interface EdgeRoutes {
  edge: string
  name: string
  segments: { segment: string; routes: SimRoute[] }[]
  lan_advertisements: LanAdvertisement[]
  withheld: Withheld[]
}

export interface RouteSet {
  topology: string
  conditions: Conditions
  assumptions: AssumptionSwitch[]
  edges: EdgeRoutes[]
  not_modelled: NotModelled[]
}

export type StageId = 'known' | 'advertised' | 'filtered' | 'redistributed' | 'winner' | 'path'

export interface ReachStage {
  number: number
  id: StageId
  question: string
  status: 'ok' | 'failed' | 'skipped' | 'not-reached' | 'not-modelled'
  detail: string
  evidence?: string[]
  basis: Basis
  also?: Basis[]
  lesson_ref: string
  where: string
}

export interface OriginView {
  edge: string
  name: string
  prefix: string
  source: string
  via: string
  stage: StageId | 'arrived'
  status: 'ok' | 'failed'
  reason: string
  basis: Basis
  also?: Basis[]
  selected: boolean
}

export interface ReachResult {
  query: { source: string; destination: string; segment?: string; destination_segment?: string }
  prefix: string
  outcome: 'reachable' | 'unreachable' | 'not-modelled'
  first_failing: StageId | ''
  reason: string
  lesson_ref: string
  stages: ReachStage[]
  origins: OriginView[]
  selected?: SimRoute
  candidates: SimRoute[]
  path?: PathOption
  assumptions: AssumptionSwitch[]
  depends_on: {
    switch: AssumptionSwitch
    flipped_value: boolean | string
    new_outcome: string
    new_first_failing: string
  }[]
  not_modelled: NotModelled[]
  return_traffic: string
}

export const postRoutes = (topologyId: string, conditions: Conditions, assumptions?: Partial<RoutingAssumptions>) =>
  sendJSON<RouteSet>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/routes`, { conditions, assumptions })
export const postReachability = (
  topologyId: string,
  query: { source: string; destination: string; segment?: string; destination_segment?: string },
  conditions: Conditions,
  assumptions?: Partial<RoutingAssumptions>,
) =>
  sendJSON<ReachResult>('POST', `/api/topologies/${encodeURIComponent(topologyId)}/reachability`, {
    query,
    conditions,
    assumptions,
  })
