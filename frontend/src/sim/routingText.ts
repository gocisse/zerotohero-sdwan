import type { AssumptionSwitch, Conditions, SimRoute, StageId } from '../api'

export const SOURCE_LABEL: Record<string, string> = {
  connected: 'Connected',
  'local-static': 'Static (this Edge)',
  'nsd-local': 'Non SD-WAN static (this Edge)',
  'nsd-remote': 'Non SD-WAN static (remote)',
  'remote-connected': 'Remote Edge: connected',
  'remote-static': 'Remote Edge: static',
  ospf: 'OSPF',
  bgp: 'BGP',
  'remote-dynamic': 'Remote Edge: dynamic',
}

export function sourceText(r: Pick<SimRoute, 'source' | 'route_type'>): string {
  const base = SOURCE_LABEL[r.source] ?? r.source
  return r.route_type ? `${base} ${r.route_type}` : base
}

/** The dotted netmask of a prefix like 10.20.0.0/24. */
export function netmask(prefix: string): string {
  const bits = Number(prefix.split('/')[1] ?? 32)
  const m = bits === 0 ? 0 : (0xffffffff << (32 - bits)) >>> 0
  return [24, 16, 8, 0].map((s) => (m >>> s) & 255).join('.')
}

export function addressOf(prefix: string): string {
  return prefix.split('/')[0] ?? prefix
}

export function sortPrefixes(a: string, b: string): number {
  const pa = a.split('/')[0]!.split('.').map(Number)
  const pb = b.split('/')[0]!.split('.').map(Number)
  for (let i = 0; i < 4; i++) if (pa[i] !== pb[i]) return (pa[i] ?? 0) - (pb[i] ?? 0)
  return Number(a.split('/')[1]) - Number(b.split('/')[1])
}

export const STAGE_TITLE: Record<StageId, string> = {
  known: 'The owning Edge never learned the prefix',
  advertised: 'The owning Edge does not advertise it into the overlay',
  filtered: 'A filter or the overlap rule holds it back',
  redistributed: 'The source Edge does not hand it to its LAN routers',
  winner: 'No route for it reaches the source Edge (or another route wins)',
  path: 'The route exists but its path is not built or reachable',
}

export const STATUS_TEXT: Record<string, string> = {
  ok: 'Passed',
  failed: 'Failed',
  skipped: 'Not needed',
  'not-reached': 'Not reached',
  'not-modelled': 'Not modelled',
}

export function switchValueText(s: AssumptionSwitch): string {
  if (typeof s.value === 'boolean') return s.value ? 'on' : 'off'
  return String(s.value).replace(/_/g, ' ')
}

export function withNeighbor(c: Conditions, id: string, down: boolean): Conditions {
  const rest = (c.neighbors_down ?? []).filter((x) => x !== id)
  return { ...c, neighbors_down: down ? [...rest, id] : rest }
}

export function withProbe(c: Conditions, id: string, down: boolean): Conditions {
  const rest = (c.probes_down ?? []).filter((x) => x !== id)
  return { ...c, probes_down: down ? [...rest, id] : rest }
}
