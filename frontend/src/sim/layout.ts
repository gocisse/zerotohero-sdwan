import type { NodeKind, Topology } from '../api'

export interface MapNode {
  /** `<kind>:<id>`, unique across Edges, Gateways and destinations. */
  key: string
  kind: 'edge' | 'gateway' | 'nsd'
  id: string
  label: string
  sub?: string
  icon: NodeKind
  x: number
  y: number
  hub: boolean
}

export interface MapBox {
  id: string
  label: string
  x: number
  y: number
  width: number
  height: number
}

export interface MapLayout {
  width: number
  height: number
  nodes: MapNode[]
  boxes: MapBox[]
}

const ROW_GAP = 190
const TOP = 80
const MIN_STEP = 150

const spread = (n: number, width: number) => Array.from({ length: n }, (_, i) => ((i + 0.5) * width) / Math.max(n, 1))

/**
 * Lays the topology out in rows: Gateways (with destinations built by a Gateway) on top, Hubs and Hub clusters
 * in the middle, the other Edges below, destinations built by an Edge at the bottom under their Edge.
 * Pure geometry in viewBox units so it can be unit tested.
 */
export function layoutTopology(t: Topology): MapLayout {
  const edges = t.sites.flatMap((s) => s.edges)
  const clusters = t.clusters ?? []
  const hubIds = new Set<string>()
  for (const p of t.profiles) for (const h of p.cloud_vpn?.branch_to_hub?.hubs ?? []) hubIds.add(h)
  const clusterMembers = new Set(clusters.flatMap((c) => c.members))
  for (const c of clusters) if (hubIds.has(c.id)) c.members.forEach((m) => hubIds.add(m))
  for (const e of edges) if (e.role === 'hub') hubIds.add(e.id)
  const isHub = (id: string) => hubIds.has(id) || clusterMembers.has(id)

  // Hubs in Hub-list order, cluster members kept together.
  const hubs: string[] = []
  const pushHub = (id: string) => {
    if (!hubs.includes(id)) hubs.push(id)
  }
  for (const p of t.profiles)
    for (const h of p.cloud_vpn?.branch_to_hub?.hubs ?? []) {
      const c = clusters.find((x) => x.id === h)
      if (c) c.members.forEach(pushHub)
      else if (edges.some((e) => e.id === h)) pushHub(h)
    }
  for (const e of edges) if (isHub(e.id)) pushHub(e.id)
  const branches = edges.filter((e) => !isHub(e.id)).map((e) => e.id)

  const gwNsd = (t.non_sdwan_destinations ?? []).filter((n) => n.via === 'gateway')
  const edgeNsd = (t.non_sdwan_destinations ?? []).filter((n) => n.via === 'edge')
  // Gateways that build a destination go last in the row, next to the destination, so their tunnel crosses nothing.
  const builders = new Set(gwNsd.map((n) => n.gateway))
  const gateways = [...t.gateways.filter((g) => !builders.has(g.id)), ...t.gateways.filter((g) => builders.has(g.id))]
  const topItems = gateways.length + gwNsd.length
  const perRow = 6
  const branchRows = Math.max(1, Math.ceil(branches.length / perRow))
  const width = Math.max(
    1000,
    Math.max(topItems, hubs.length, Math.min(branches.length, perRow), edgeNsd.length) * MIN_STEP,
  )

  const nodes: MapNode[] = []
  const yTop = TOP
  const yHub = TOP + ROW_GAP
  const yBranch = TOP + 2 * ROW_GAP
  const xsTop = spread(topItems, width)
  gateways.forEach((g, i) =>
    nodes.push({
      key: `gateway:${g.id}`,
      kind: 'gateway',
      id: g.id,
      label: g.id,
      sub: g.region,
      icon: 'vcg',
      x: xsTop[i] ?? 0,
      y: yTop,
      hub: false,
    }),
  )
  gwNsd.forEach((n, i) =>
    nodes.push({
      key: `nsd:${n.id}`,
      kind: 'nsd',
      id: n.id,
      label: n.name ?? n.id,
      sub: 'via Gateway',
      icon: 'cloud',
      x: xsTop[gateways.length + i] ?? 0,
      y: yTop,
      hub: false,
    }),
  )
  const name = (id: string) => edges.find((e) => e.id === id)?.name ?? id
  const xsHub = spread(hubs.length, width)
  hubs.forEach((id, i) =>
    nodes.push({
      key: `edge:${id}`,
      kind: 'edge',
      id,
      label: name(id),
      sub: 'Hub',
      icon: 'datacenter',
      x: xsHub[i] ?? 0,
      y: yHub,
      hub: true,
    }),
  )
  for (let r = 0; r < branchRows; r++) {
    const row = branches.slice(r * perRow, (r + 1) * perRow)
    const xs = spread(row.length, width)
    row.forEach((id, i) =>
      nodes.push({
        key: `edge:${id}`,
        kind: 'edge',
        id,
        label: name(id),
        icon: 'vce',
        x: xs[i] ?? 0,
        y: yBranch + r * ROW_GAP,
        hub: false,
      }),
    )
  }
  let height = yBranch + (branchRows - 1) * ROW_GAP + 110
  if (edgeNsd.length > 0) {
    const yN = yBranch + branchRows * ROW_GAP
    edgeNsd.forEach((n) => {
      const builder = nodes.find((x) => x.key === `edge:${n.edge}`)
      nodes.push({
        key: `nsd:${n.id}`,
        kind: 'nsd',
        id: n.id,
        label: n.name ?? n.id,
        sub: 'via Edge',
        icon: 'cloud',
        x: builder?.x ?? width / 2,
        y: yN,
        hub: false,
      })
    })
    height = yN + 110
  }

  const boxes: MapBox[] = clusters.map((c) => {
    const members = nodes.filter((n) => n.kind === 'edge' && c.members.includes(n.id))
    const xs = members.map((m) => m.x)
    const x = Math.min(...xs) - 70
    return { id: c.id, label: c.name ?? c.id, x, y: yHub - 62, width: Math.max(...xs) + 70 - x, height: 150 }
  })
  return { width, height, nodes, boxes }
}
