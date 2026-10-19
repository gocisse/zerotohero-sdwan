import { useEffect, useRef } from 'react'
import type { DiagramLink, DiagramPacket } from '../../api'
import { LINK_STYLES } from './linkStyles'
import { PACKET_MIN_MS, PACKET_SPEED } from './packets'
import type { Rect } from './calloutPlacement'
import { packetLabelDy } from './packetLabel'

interface Segment {
  el: SVGPathElement
  len: number
  reverse: boolean
}

interface Props {
  packets: DiagramPacket[]
  links: Map<string, DiagramLink>
  getPath: (linkId: string) => SVGPathElement | undefined
  /** false (reduced motion): packets are drawn at their destination, with no movement. */
  animate: boolean
  /** Text growth factor on small screens. */
  labelScale?: number
  /** Node discs the packet label must not cover (visible nodes only). */
  discs?: Rect[]
}

/**
 * Packets travelling along their resolved routes. Positions are written straight to the DOM from a
 * requestAnimationFrame loop (no React re-render per frame). Mount with a fresh `key` per step.
 */
export default function PacketLayer({ packets, links, getPath, animate, labelScale = 1, discs = [] }: Props) {
  const nodes = useRef<(SVGGElement | null)[]>([])

  useEffect(() => {
    const items = packets.map((p, i) => {
      const g = nodes.current[i]
      const segments: Segment[] = []
      if (p.route) {
        p.route.links.forEach((id, k) => {
          const el = getPath(id)
          const link = links.get(id)
          if (el && link) segments.push({ el, len: el.getTotalLength(), reverse: p.route!.nodes[k] !== link.from })
        })
      }
      const total = segments.reduce((s, x) => s + x.len, 0)
      return { g, p, segments, total, duration: Math.max(PACKET_MIN_MS, (total / PACKET_SPEED) * 1000) }
    })

    const place = (it: (typeof items)[number], dist: number, state: string) => {
      if (!it.g || it.segments.length === 0) return
      let rest = Math.min(Math.max(dist, 0), it.total)
      for (const seg of it.segments) {
        if (rest <= seg.len || seg === it.segments[it.segments.length - 1]) {
          const at = seg.reverse ? seg.len - Math.min(rest, seg.len) : Math.min(rest, seg.len)
          const pt = seg.el.getPointAtLength(at)
          it.g.setAttribute('transform', `translate(${pt.x} ${pt.y})`)
          const text = it.g.querySelector('text')
          if (text && it.p.label) text.setAttribute('y', String(packetLabelDy(pt, it.p.label, 13 * labelScale, discs)))
          break
        }
        rest -= seg.len
      }
      it.g.setAttribute('visibility', 'visible')
      it.g.dataset.state = state
    }

    if (!animate) {
      items.forEach((it) => place(it, it.total, 'arrived'))
      return
    }

    let raf = 0
    const start = performance.now()
    const frame = (now: number) => {
      let running = false
      for (const it of items) {
        const t = now - start - it.p.delay_ms
        if (t < 0) {
          running = true
          continue
        }
        const k = Math.min(1, t / it.duration)
        place(it, k * it.total, k >= 1 ? 'arrived' : 'moving')
        if (k < 1) running = true
      }
      if (running) raf = requestAnimationFrame(frame)
    }
    raf = requestAnimationFrame(frame)
    return () => cancelAnimationFrame(raf)
  }, [packets, links, getPath, animate, labelScale, discs])

  return (
    <g aria-hidden>
      {packets.map((p, i) => {
        const look =
          LINK_STYLES[
            p.color_style ?? (p.route ? links.get(p.route.links[0] ?? '')?.style : undefined) ?? 'overlay-tunnel'
          ]
        return (
          <g
            key={i}
            ref={(el) => {
              nodes.current[i] = el
            }}
            data-packet
            data-state="waiting"
            visibility="hidden"
          >
            <circle r={9} className={`${look.fill} stroke-white dark:stroke-slate-950`} strokeWidth={2.5} />
            {p.label && (
              <text
                y={-15 * labelScale}
                textAnchor="middle"
                fontSize={13 * labelScale}
                fontWeight={700}
                paintOrder="stroke"
                strokeWidth={4}
                className="fill-slate-900 stroke-slate-50 dark:fill-slate-100 dark:stroke-slate-900"
              >
                {p.label}
              </text>
            )}
          </g>
        )
      })}
    </g>
  )
}
