import {
  Component,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
  type ErrorInfo,
  type ReactNode,
} from 'react'
import { fetchDiagram, type Diagram as DiagramDef, type LinkStyle } from '../../api'
import ErrorPanel from '../../components/ErrorPanel'
import Markdown from '../../components/Markdown'
import DiagramCanvas from './DiagramCanvas'
import { hiddenAt } from './geometry'
import { LINK_STYLES } from './linkStyles'
import { packetDurationMs } from './packets'
import { panTarget } from './panTarget'

const reducedQuery = '(prefers-reduced-motion: reduce)'
function useReducedMotion(): boolean {
  return useSyncExternalStore(
    (cb) => {
      const m = window.matchMedia(reducedQuery)
      m.addEventListener('change', cb)
      return () => m.removeEventListener('change', cb)
    },
    () => window.matchMedia(reducedQuery).matches,
    () => false,
  )
}

type Load = { kind: 'loading' } | { kind: 'ready'; diagram: DiagramDef } | { kind: 'error'; error: Error }

/** `<Diagram id="..." />`: loads a diagram definition from the backend and renders it with a step player. */
export default function Diagram({ id }: { id?: string }) {
  const [load, setLoad] = useState<Load>({ kind: 'loading' })

  useEffect(() => {
    if (!id) return
    let cancelled = false
    fetchDiagram(id)
      .then((diagram) => !cancelled && setLoad({ kind: 'ready', diagram }))
      .catch((error: Error) => !cancelled && setLoad({ kind: 'error', error }))
    return () => {
      cancelled = true
    }
  }, [id])

  if (!id) return <ErrorPanel title="Diagram error" error={new Error('<Diagram> needs an id="..." attribute.')} />
  if (load.kind === 'loading') return <p className="not-prose my-6 text-sm text-slate-500">Loading diagram…</p>
  if (load.kind === 'error') return <ErrorPanel title={`Diagram “${id}” could not be loaded`} error={load.error} />
  return (
    <DiagramBoundary id={id}>
      <DiagramPlayer key={load.diagram.id} diagram={load.diagram} />
    </DiagramBoundary>
  )
}

class DiagramBoundary extends Component<{ id: string; children: ReactNode }, { error?: Error }> {
  state: { error?: Error } = {}
  static getDerivedStateFromError(error: Error) {
    return { error }
  }
  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Diagram render error', error, info.componentStack)
  }
  render() {
    if (this.state.error)
      return <ErrorPanel title={`Diagram “${this.props.id}” failed to render`} error={this.state.error} />
    return this.props.children
  }
}

const btn =
  'rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-sky-400 hover:text-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-700 dark:hover:border-sky-600 dark:hover:text-sky-300'

function DiagramPlayer({ diagram: d }: { diagram: DiagramDef }) {
  const reduced = useReducedMotion()
  const scrollRef = useRef<HTMLDivElement>(null)
  const [pannable, setPannable] = useState(false)
  // Changes when the canvas has laid out at a new width: the callout bubbles move with it, so the pan target is computed again.
  const [layoutWidth, setLayoutWidth] = useState(0)
  const linkPaths = useRef(new Map<string, SVGPathElement>())
  const registerPath = useCallback((id: string, el: SVGPathElement | null) => {
    if (el) linkPaths.current.set(id, el)
    else linkPaths.current.delete(id)
  }, [])
  const getPath = useCallback((id: string) => linkPaths.current.get(id), [])
  const [index, setIndex] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [replays, setReplays] = useState(0)

  const stepped = d.steps.length > 0
  const last = d.steps.length - 1
  const step = stepped ? d.steps[Math.min(index, last)] : undefined
  const hidden = useMemo(() => hiddenAt(d, stepped ? index : -1), [d, index, stepped])
  const highlight = step && step.highlight.length > 0 ? new Set(step.highlight) : null
  const stepKey = `${d.id}:${index}:${replays}`

  const go = useCallback((i: number) => setIndex(Math.min(Math.max(i, 0), Math.max(last, 0))), [last])

  // Autoplay: wait long enough to read the narration and for packets to finish, then advance.
  useEffect(() => {
    if (!playing || !step) return
    if (index >= last) {
      const t = setTimeout(() => setPlaying(false), 0)
      return () => clearTimeout(t)
    }
    const lengthOf = (id: string) => getPath(id)?.getTotalLength() ?? 0
    const travel = reduced ? 0 : Math.max(0, ...step.packets.map((p) => packetDurationMs(p, lengthOf)))
    const t = setTimeout(() => setIndex((i) => i + 1), Math.max(4000, travel + 2200))
    return () => clearTimeout(t)
  }, [playing, index, last, step, reduced, getPath])

  // On narrow screens the diagram scrolls sideways; follow the part of the picture each step is about.
  useEffect(() => {
    const box = scrollRef.current
    const svg = box?.querySelector('svg')
    if (!box || !svg) return
    setPannable(box.scrollWidth > box.clientWidth + 4)
    if (box.scrollWidth <= box.clientWidth + 4) return
    const focusIds = new Set<string>()
    if (step) {
      step.highlight.forEach((id) => focusIds.add(id))
      step.show.forEach((id) => focusIds.add(id))
      step.packets.forEach((p) => p.route?.nodes.forEach((n) => focusIds.add(n)))
    }
    const nodes = d.nodes.filter((n) => focusIds.has(n.id))
    const xs = (nodes.length > 0 ? nodes : d.nodes).map((n) => n.x)
    const vb = svg.viewBox.baseVal
    const ratio = svg.clientWidth / vb.width
    const boxRect = box.getBoundingClientRect()
    const bubbles = Array.from(box.querySelectorAll('[data-callout-bubble]')).map((el) => {
      const r = el.getBoundingClientRect()
      return { left: r.left - boxRect.left + box.scrollLeft, right: r.right - boxRect.left + box.scrollLeft }
    })
    const left = panTarget({
      nodes: {
        left: (Math.min(...xs) - vb.x) * ratio - 41 * ratio,
        right: (Math.max(...xs) - vb.x) * ratio + 41 * ratio,
      },
      bubbles,
      view: box.clientWidth,
      maxScroll: box.scrollWidth - box.clientWidth,
    })
    box.scrollTo({ left, behavior: reduced ? 'auto' : 'smooth' })
  }, [index, step, d, reduced, layoutWidth])

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (!stepped || e.altKey || e.ctrlKey || e.metaKey) return
    if (e.key === 'ArrowRight') {
      e.preventDefault()
      go(index + 1)
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      go(index - 1)
    }
  }

  const used = useMemo(() => [...new Set(d.links.map((l) => l.style))] as LinkStyle[], [d])

  return (
    <figure
      id={`diagram-${d.id}`}
      className="not-prose my-6 rounded-xl border border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-900"
      data-diagram={d.id}
    >
      <figcaption className="mb-2 px-1 text-sm font-semibold">{d.title}</figcaption>
      <div
        tabIndex={0}
        role="group"
        aria-label={stepped ? `${d.title}. Step player: use the left and right arrow keys to change step.` : d.title}
        onKeyDown={onKeyDown}
        className="rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
      >
        <div ref={scrollRef} className="overflow-x-auto">
          <DiagramCanvas
            onLayout={setLayoutWidth}
            diagram={d}
            hidden={hidden}
            highlight={highlight}
            step={step}
            stepKey={stepKey}
            animate={!reduced}
            registerPath={registerPath}
            getPath={getPath}
          />
        </div>

        {pannable && (
          <p className="mt-1 px-1 text-xs text-slate-500 dark:text-slate-400" data-pan-hint>
            Swipe sideways to pan; the view follows each step.
          </p>
        )}

        {stepped && step && (
          <div className="mt-3 border-t border-slate-200 px-1 pt-3 dark:border-slate-800">
            <div className="flex flex-wrap items-center gap-2" role="toolbar" aria-label="Diagram steps">
              <button type="button" className={btn} onClick={() => go(index - 1)} disabled={index === 0}>
                ← Previous
              </button>
              <button type="button" className={btn} onClick={() => go(index + 1)} disabled={index >= last}>
                Next →
              </button>
              <button type="button" className={btn} onClick={() => setPlaying((p) => !p)} aria-pressed={playing}>
                {playing ? '⏸ Pause' : '▶ Play'}
              </button>
              <button
                type="button"
                className={btn}
                onClick={() => {
                  setIndex(0)
                  setReplays((r) => r + 1)
                  setPlaying(true)
                }}
              >
                ↻ Replay
              </button>
              <span className="ml-auto text-sm text-slate-600 dark:text-slate-400" data-step-indicator>
                Step {index + 1} of {d.steps.length}
              </span>
            </div>
            <div aria-live="polite" aria-atomic="true" className="mt-2" data-step-narration>
              <div className="font-semibold">{step.title}</div>
              <Markdown className="text-sm">{step.narration}</Markdown>
            </div>
          </div>
        )}
      </div>

      <ul
        aria-label="Legend"
        className="mt-3 flex flex-wrap gap-x-5 gap-y-1 px-1 text-xs text-slate-600 dark:text-slate-400"
        data-legend
      >
        {used.map((s) => {
          const look = LINK_STYLES[s]
          return (
            <li key={s} className="flex items-center gap-2" data-legend-style={s}>
              <svg width={36} height={10} aria-hidden>
                <line
                  x1={2}
                  x2={34}
                  y1={5}
                  y2={5}
                  className={look.stroke}
                  strokeWidth={Math.min(look.width, 6)}
                  strokeDasharray={look.dash}
                  strokeLinecap={look.round ? 'round' : 'butt'}
                />
              </svg>
              {look.label}
            </li>
          )
        })}
      </ul>
      <p className="sr-only">{d.alt}</p>
    </figure>
  )
}
