import { useMemo } from 'react'
import type { Topology, Tunnel, TunnelType } from '../api'
import NodeIcon from '../mdx/diagram/icons'
import { LINK_STYLES } from '../mdx/diagram/linkStyles'
import { layoutTopology, type MapNode } from './layout'
import { stateLabel, typeLabel } from './overlayText'

interface Props {
  topology: Topology
  tunnels: Tunnel[]
  types: Set<TunnelType>
  secondaryGateway: boolean
  showPotential: boolean
  unreachable: { edges: string[]; gateways: string[]; nsds: string[] }
}

const GHOST = new Set(['not-established', 'torn-down'])

function curve(a: MapNode, b: MapNode, bend: number): { d: string; mid: { x: number; y: number } } {
  const mx = (a.x + b.x) / 2
  const my = (a.y + b.y) / 2
  if (!bend) return { d: `M ${a.x} ${a.y} L ${b.x} ${b.y}`, mid: { x: mx, y: my } }
  const len = Math.hypot(b.x - a.x, b.y - a.y) || 1
  const cx = mx + (-(b.y - a.y) / len) * bend * 2
  const cy = my + ((b.x - a.x) / len) * bend * 2
  return {
    d: `M ${a.x} ${a.y} Q ${cx} ${cy} ${b.x} ${b.y}`,
    mid: { x: 0.25 * a.x + 0.5 * cx + 0.25 * b.x, y: 0.25 * a.y + 0.5 * cy + 0.25 * b.y },
  }
}

const symbol: Record<string, string> = { degraded: '!', down: '✕', unknown: '?' }

/** The topology map: sites, Hubs, Gateways, destinations and the tunnels that exist now. Down or degraded elements are marked. */
export default function TunnelMap({ topology, tunnels, types, secondaryGateway, showPotential, unreachable }: Props) {
  const layout = useMemo(() => layoutTopology(topology), [topology])
  const byKey = useMemo(() => new Map(layout.nodes.map((n) => [n.key, n])), [layout])
  const shown = tunnels.filter((t) => {
    if (!types.has(t.type)) return false
    if (t.type === 'gateway' && t.role !== 'primary' && !secondaryGateway) return false
    if (GHOST.has(t.state)) return showPotential
    return true
  })
  const counts = shown.reduce<Record<string, number>>((m, t) => ({ ...m, [t.state]: (m[t.state] ?? 0) + 1 }), {})
  const summary = Object.entries(counts)
    .map(([s, n]) => `${n} ${stateLabel[s as keyof typeof stateLabel] ?? s}`)
    .join(', ')
  const isDown = (n: MapNode) =>
    (n.kind === 'edge' && unreachable.edges.includes(n.id)) ||
    (n.kind === 'gateway' && unreachable.gateways.includes(n.id)) ||
    (n.kind === 'nsd' && unreachable.nsds.includes(n.id))

  return (
    <div
      className="overflow-x-auto rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/40"
      data-tunnel-map
    >
      <svg
        viewBox={`0 0 ${layout.width} ${layout.height}`}
        role="img"
        aria-label={`Topology map: ${layout.nodes.length} nodes and ${shown.length} tunnels shown (${summary || 'none'}).`}
        style={{ minWidth: 640, width: '100%', height: 'auto' }}
      >
        {layout.boxes.map((b) => (
          <g key={b.id}>
            <rect
              x={b.x}
              y={b.y}
              width={b.width}
              height={b.height}
              rx={14}
              fill="none"
              strokeDasharray="6 5"
              strokeWidth={2}
              className="stroke-slate-400 dark:stroke-slate-500"
            />
            <text
              x={b.x + 12}
              y={b.y + 18}
              fontSize={14}
              fontWeight={600}
              className="fill-slate-600 dark:fill-slate-300"
            >
              Hub cluster: {b.label}
            </text>
          </g>
        ))}
        {shown.map((t) => {
          const a = byKey.get(`${t.a.kind}:${t.a.id}`)
          const b = byKey.get(`${t.b.kind}:${t.b.id}`)
          if (!a || !b) return null
          const look = LINK_STYLES[t.type === 'nsd-gateway' || t.type === 'nsd-edge' ? 'ipsec' : 'overlay-tunnel']
          const ghost = GHOST.has(t.state)
          const { d, mid } = curve(a, b, t.type === 'dynamic' ? 45 : 0)
          let stroke = look.stroke
          let dash = look.dash
          let width = look.width
          if (t.state === 'degraded') [stroke, dash] = ['stroke-amber-500 dark:stroke-amber-400', '9 5']
          if (t.state === 'down') [stroke, dash, width] = ['stroke-red-600 dark:stroke-red-400', '3 6', look.width + 1]
          if (t.state === 'unknown') [stroke, dash] = ['stroke-violet-500 dark:stroke-violet-400', '2 7']
          if (ghost) dash = '2 9'
          return (
            <g key={t.id} data-map-tunnel={t.id} data-state={t.state} opacity={ghost ? 0.4 : 1}>
              <title>{`${typeLabel[t.type]}: ${t.a.name} to ${t.b.name}, ${stateLabel[t.state]}. ${t.reason}`}</title>
              <path
                d={d}
                fill="none"
                className={stroke}
                strokeWidth={width}
                strokeDasharray={dash}
                strokeLinecap={look.round ? 'round' : 'butt'}
              />
              {symbol[t.state] && (
                <g transform={`translate(${mid.x} ${mid.y})`}>
                  <circle r={11} className="fill-white stroke-slate-500 dark:fill-slate-900" strokeWidth={1.5} />
                  <text
                    textAnchor="middle"
                    y={5}
                    fontSize={14}
                    fontWeight={700}
                    className="fill-slate-900 dark:fill-slate-100"
                  >
                    {symbol[t.state]}
                  </text>
                </g>
              )}
            </g>
          )
        })}
        {layout.nodes.map((n) => (
          <g
            key={n.key}
            transform={`translate(${n.x} ${n.y})`}
            data-map-node={n.key}
            data-unreachable={isDown(n) || undefined}
          >
            <title>{`${n.label}${n.sub ? `, ${n.sub}` : ''}${isDown(n) ? ', unreachable' : ''}`}</title>
            <circle
              r={38}
              className="fill-white stroke-slate-300 dark:fill-slate-900 dark:stroke-slate-600"
              strokeWidth={1.5}
            />
            {isDown(n) && (
              <circle
                r={45}
                fill="none"
                strokeDasharray="5 4"
                strokeWidth={3}
                className="stroke-red-600 dark:stroke-red-400"
              />
            )}
            <g className="stroke-current text-slate-800 dark:text-slate-100">
              <NodeIcon kind={n.icon} />
            </g>
            <text
              y={62}
              textAnchor="middle"
              fontSize={15}
              fontWeight={600}
              paintOrder="stroke"
              strokeWidth={5}
              className="fill-slate-900 stroke-slate-50 dark:fill-slate-100 dark:stroke-slate-900"
            >
              {n.label}
            </text>
            {(n.sub || isDown(n)) && (
              <text
                y={79}
                textAnchor="middle"
                fontSize={12}
                paintOrder="stroke"
                strokeWidth={4}
                className={`${isDown(n) ? 'fill-red-700 dark:fill-red-300' : 'fill-slate-500 dark:fill-slate-400'} stroke-slate-50 dark:stroke-slate-900`}
              >
                {isDown(n) ? 'unreachable' : n.sub}
              </text>
            )}
          </g>
        ))}
      </svg>
    </div>
  )
}

/** Legend: link looks from the shared diagram vocabulary plus the state marks. */
export function MapLegend() {
  const items = [
    { look: LINK_STYLES['overlay-tunnel'], label: 'Overlay tunnel (Gateway, Hub, dynamic)' },
    { look: LINK_STYLES.ipsec, label: 'IPsec to a Non SD-WAN destination' },
  ]
  return (
    <ul
      className="flex flex-wrap gap-x-5 gap-y-1 text-xs text-slate-600 dark:text-slate-300"
      aria-label="Legend"
      data-map-legend
    >
      {items.map((i) => (
        <li key={i.label} className="flex items-center gap-2">
          <svg width={36} height={10} aria-hidden>
            <line
              x1={0}
              y1={5}
              x2={36}
              y2={5}
              className={i.look.stroke}
              strokeWidth={i.look.width}
              strokeDasharray={i.look.dash}
            />
          </svg>
          {i.label}
        </li>
      ))}
      <li>
        <b aria-hidden>✕</b> down
      </li>
      <li>
        <b aria-hidden>!</b> degraded
      </li>
      <li>
        <b aria-hidden>?</b> unknown
      </li>
      <li>faint dotted line: potential tunnel that is not established or was torn down</li>
      <li>dashed red ring: unreachable</li>
    </ul>
  )
}
