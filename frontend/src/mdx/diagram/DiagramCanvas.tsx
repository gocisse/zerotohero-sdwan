import { useEffect, useId, useMemo, useRef, useState } from 'react'
import type { Diagram, DiagramStep } from '../../api'
import { bubbleSize, labelRect, placeCallout, type Rect } from './calloutPlacement'
import { nodeDisc } from './packetLabel'
import { groupBox, groupMembers, linkGeometry } from './geometry'
import NodeIcon from './icons'
import { KIND_NAMES, LINK_STYLES } from './linkStyles'
import PacketLayer from './PacketLayer'

interface Props {
  diagram: Diagram
  hidden: Set<string>
  /** Elements to emphasize; everything else dims. null = no dimming. */
  highlight: Set<string> | null
  step: DiagramStep | undefined
  /** Changes whenever packets should restart. */
  stepKey: string
  animate: boolean
  registerPath: (linkId: string, el: SVGPathElement | null) => void
  getPath: (linkId: string) => SVGPathElement | undefined
}

const stateClass = (hidden: boolean, dimmed: boolean) =>
  `transition-opacity duration-300 ${hidden ? 'pointer-events-none opacity-0' : dimmed ? 'opacity-25' : 'opacity-100'}`

/** The responsive SVG, plus HTML overlays (tooltip, callouts) positioned in viewBox percentages. */
export default function DiagramCanvas({
  diagram: d,
  hidden,
  highlight,
  step,
  stepKey,
  animate,
  registerPath,
  getPath,
}: Props) {
  const [hover, setHover] = useState<string | null>(null)
  const [focus, setFocus] = useState<string | null>(null)
  const nodeById = useMemo(() => new Map(d.nodes.map((n) => [n.id, n])), [d])
  const linkById = useMemo(() => new Map(d.links.map((l) => [l.id, l])), [d])
  const geoms = useMemo(
    () =>
      new Map(
        d.links.map((l) => {
          const a = nodeById.get(l.from)!
          const b = nodeById.get(l.to)!
          return [l.id, linkGeometry(a, b, l.curve)] as const
        }),
      ),
    [d, nodeById],
  )
  const idSuffix = useId()

  // Fit the SVG to its container. When it is scaled well below 1 (phones), grow the text in viewBox units
  // so labels stay readable, and drop sublabels (the tooltip still shows them).
  const wrapRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(0)
  useEffect(() => {
    const el = wrapRef.current
    if (!el) return
    const ro = new ResizeObserver(([entry]) => setWidth(entry?.contentRect.width ?? 0))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  const scale = width > 0 ? width / d.width : 1
  const boost = Math.min(Math.max(1, 0.66 / scale), 2.2)
  const showSub = scale >= 0.55
  const showLinkLabels = scale >= 0.55
  // Side padding so wide labels near the edges are not clipped.
  const pad = 30
  const totalW = d.width + 2 * pad

  const linkHidden = (id: string) => {
    const l = linkById.get(id)!
    return hidden.has(id) || hidden.has(l.from) || hidden.has(l.to)
  }
  const dimmed = (id: string) => highlight !== null && !highlight.has(id)

  const tipNode = nodeById.get(focus ?? hover ?? '')
  const pct = (x: number, total: number) => `${(x / total) * 100}%`
  const clampX = (x: number) => `${Math.min(Math.max(((x + pad) / totalW) * 100, 14), 86)}%`

  // Callouts avoid link labels, node discs and node labels (see calloutPlacement.ts).
  const obstacles: Rect[] = [
    ...d.links.flatMap((l) =>
      l.label && showLinkLabels && !linkHidden(l.id)
        ? [labelRect(geoms.get(l.id)!.mid.x, geoms.get(l.id)!.mid.y - 8, l.label, 13 * boost)]
        : [],
    ),
    ...d.nodes.flatMap((n) =>
      hidden.has(n.id)
        ? []
        : [
            {
              x: n.x - Math.max(41, (n.label.length * 17 * boost * 0.56) / 2),
              y: n.y - 41,
              width: 2 * Math.max(41, (n.label.length * 17 * boost * 0.56) / 2),
              height: 41 + 58 + 17 * boost + (n.sublabel && showSub ? 20 : 0),
            },
          ],
    ),
  ]
  const discs = useMemo(() => d.nodes.filter((n) => !hidden.has(n.id)).map((n) => nodeDisc(n.x, n.y)), [d, hidden])
  const canvasRect: Rect = { x: -pad, y: 0, width: totalW, height: d.height }
  const callouts = (step?.callouts ?? []).flatMap((c) => {
    const n = nodeById.get(c.target)
    const l = linkById.get(c.target)
    const anchor =
      n && !hidden.has(n.id) ? { x: n.x, y: n.y - 41 } : l && !linkHidden(l.id) ? geoms.get(l.id)!.mid : null
    if (!anchor) return []
    const size = bubbleSize(c.text, scale)
    const placed = placeCallout({
      anchor,
      size,
      obstacles,
      canvas: canvasRect,
      prefer: anchor.y < d.height * 0.3 ? 'below' : 'above',
    })
    return [{ ...c, ...placed }]
  })

  return (
    <div ref={wrapRef} className="relative" style={{ minWidth: Math.min(d.width, 560) }}>
      <svg
        viewBox={`${-pad} 0 ${totalW} ${d.height}`}
        role="group"
        aria-label={d.alt}
        className="block h-auto w-full"
        data-diagram-svg
      >
        {d.groups.map((g) => {
          const box = groupBox(d, g)
          if (!box) return null
          const members = groupMembers(d, g)
          const gone = members.length > 0 && members.every((m) => hidden.has(m))
          return (
            <g
              key={g.id}
              className={`transition-opacity duration-300 ${gone ? 'opacity-0' : 'opacity-100'}`}
              data-group={g.id}
            >
              <rect
                x={box.x}
                y={box.y}
                width={box.width}
                height={box.height}
                rx={16}
                className="fill-slate-100 stroke-slate-300 dark:fill-slate-800/60 dark:stroke-slate-600"
                strokeWidth={1.5}
              />
              <text
                x={box.x + 14}
                y={box.y + 8 + 14 * Math.min(boost, 1.25)}
                fontSize={14 * Math.min(boost, 1.25)}
                fontWeight={600}
                className="fill-slate-500 dark:fill-slate-400"
              >
                {g.label}
              </text>
            </g>
          )
        })}

        {d.links.map((l) => {
          const look = LINK_STYLES[l.style]
          const geo = geoms.get(l.id)!
          const gone = linkHidden(l.id)
          return (
            <g key={l.id} className={stateClass(gone, dimmed(l.id))} data-link={l.id} data-style={l.style}>
              <path
                ref={(el) => registerPath(l.id, el)}
                d={geo.d}
                fill="none"
                className={look.stroke}
                strokeWidth={look.width}
                strokeDasharray={look.dash}
                strokeLinecap={look.round ? 'round' : 'butt'}
              />
              {l.label && showLinkLabels && (
                <text
                  x={geo.mid.x}
                  y={geo.mid.y - 8}
                  textAnchor="middle"
                  fontSize={13 * boost}
                  paintOrder="stroke"
                  strokeWidth={4}
                  className="fill-slate-700 stroke-slate-50 dark:fill-slate-200 dark:stroke-slate-900"
                >
                  {l.label}
                </text>
              )}
            </g>
          )
        })}

        {d.nodes.map((n) => {
          const gone = hidden.has(n.id)
          const desc = [n.label, KIND_NAMES[n.kind], n.sublabel].filter(Boolean).join(', ')
          return (
            <g
              key={n.id}
              transform={`translate(${n.x} ${n.y})`}
              className={`group outline-none ${stateClass(gone, dimmed(n.id))}`}
              role="img"
              aria-label={desc}
              aria-hidden={gone || undefined}
              aria-describedby={tipNode?.id === n.id ? `tip-${idSuffix}` : undefined}
              tabIndex={gone ? -1 : 0}
              data-node={n.id}
              onMouseEnter={() => setHover(n.id)}
              onMouseLeave={() => setHover((h) => (h === n.id ? null : h))}
              onFocus={() => setFocus(n.id)}
              onBlur={() => setFocus((f) => (f === n.id ? null : f))}
            >
              <circle
                r={36}
                className="fill-slate-50 stroke-slate-300 dark:fill-slate-900 dark:stroke-slate-600"
                strokeWidth={1.5}
              />
              <circle
                r={41}
                fill="none"
                className="stroke-sky-500 opacity-0 group-focus-visible:opacity-100"
                strokeWidth={3}
              />
              <g className="stroke-current text-slate-800 dark:text-slate-100">
                <NodeIcon kind={n.kind} />
              </g>
              <text
                y={58 + 17 * (boost - 1)}
                textAnchor="middle"
                fontSize={17 * boost}
                fontWeight={600}
                paintOrder="stroke"
                strokeWidth={5}
                className="fill-slate-900 stroke-slate-50 dark:fill-slate-100 dark:stroke-slate-900"
              >
                {n.label}
              </text>
              {n.sublabel && showSub && (
                <text
                  y={75}
                  textAnchor="middle"
                  fontSize={12}
                  paintOrder="stroke"
                  strokeWidth={4}
                  className="fill-slate-500 stroke-slate-50 dark:fill-slate-400 dark:stroke-slate-900"
                >
                  {n.sublabel}
                </text>
              )}
            </g>
          )
        })}

        {step && step.packets.length > 0 && (
          <PacketLayer
            key={stepKey}
            packets={step.packets}
            links={linkById}
            getPath={getPath}
            animate={animate}
            labelScale={boost}
            discs={discs}
          />
        )}
      </svg>

      {callouts.map((c, i) => (
        <div
          key={`${stepKey}-${i}`}
          data-callout-bubble
          className={`pointer-events-none absolute z-10 max-w-[200px] -translate-x-1/2 rounded-md border border-amber-300 bg-amber-50 px-2.5 py-1.5 text-xs text-amber-950 shadow dark:border-amber-700 dark:bg-amber-950 dark:text-amber-50`}
          style={{ left: pct(c.cx + pad, totalW), top: pct(c.top, d.height) }}
        >
          {c.text}
        </div>
      ))}

      {tipNode && !hidden.has(tipNode.id) && (
        <div
          id={`tip-${idSuffix}`}
          role="tooltip"
          className="pointer-events-none absolute z-20 -translate-x-1/2 -translate-y-full rounded-md bg-slate-900 px-2.5 py-1.5 text-xs text-white shadow-lg dark:bg-slate-100 dark:text-slate-900"
          style={{ left: clampX(tipNode.x), top: pct(tipNode.y - 44, d.height) }}
        >
          <div className="font-semibold">{tipNode.label}</div>
          <div className="opacity-80">
            {KIND_NAMES[tipNode.kind]}
            {tipNode.sublabel ? ` · ${tipNode.sublabel}` : ''}
          </div>
        </div>
      )}
    </div>
  )
}
