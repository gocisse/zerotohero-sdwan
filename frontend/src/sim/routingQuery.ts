import type { SimEdge, Topology } from '../api'
import { sortPrefixes } from './routingText'

export interface Origin {
  prefix: string
  label: string
  edge: string
}

/** Every prefix an Edge originates in the topology file: connected subnets, static routes and LAN-learned prefixes. */
export function originatedPrefixes(t: Topology): Origin[] {
  const out: Origin[] = []
  for (const s of t.sites)
    for (const e of s.edges) {
      for (const i of e.interfaces)
        if ((i.kind === 'lan' || i.kind === 'routed') && i.address) {
          const [ip, bits] = i.address.split('/')
          const o = ip!.split('.').map(Number)
          const b = Number(bits)
          const mask = b === 0 ? 0 : (0xffffffff << (32 - b)) >>> 0
          const n = [(o[0]! << 24) | (o[1]! << 16) | (o[2]! << 8) | o[3]!].map((x) => (x & mask) >>> 0)[0]!
          out.push({
            prefix: `${[24, 16, 8, 0].map((sh) => (n >>> sh) & 255).join('.')}/${b}`,
            label: `connected on ${e.name}`,
            edge: e.id,
          })
        }
      for (const r of e.routing?.static_routes ?? [])
        out.push({ prefix: r.subnet, label: `static route on ${e.name}`, edge: e.id })
      for (const r of e.routing?.lan_routes ?? [])
        out.push({ prefix: r.prefix, label: `${r.protocol.toUpperCase()} behind ${e.name}`, edge: e.id })
    }
  const seen = new Set<string>()
  return out
    .filter((o) => (seen.has(o.prefix + o.label) ? false : (seen.add(o.prefix + o.label), true)))
    .sort((a, b) => sortPrefixes(a.prefix, b.prefix))
}

/** The default question: an Edge that runs BGP or OSPF asking for the first LAN prefix behind another Edge. */
export function defaultQuery(t: Topology): { source: string; destination: string } {
  const edges: SimEdge[] = t.sites.flatMap((s) => s.edges)
  const owner = edges.find((e) => (e.routing?.lan_routes ?? []).length > 0) ?? edges[0]
  const dest = owner?.routing?.lan_routes?.[0]?.prefix ?? originatedPrefixes(t)[0]?.prefix ?? ''
  const src =
    edges.find((e) => e.id !== owner?.id && (e.routing?.bgp || e.routing?.ospf)) ??
    edges.find((e) => e.id !== owner?.id) ??
    edges[0]
  return { source: src?.id ?? '', destination: dest }
}

export function lessonHref(ref: string): string {
  return ref.startsWith('ov-') ? `/course/overlay/${ref}` : `/course/routing/${ref}`
}
