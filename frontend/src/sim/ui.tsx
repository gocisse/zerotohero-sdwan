import { useEffect, useId, useRef, useState } from 'react'
import type { Basis, NotModelled, SimSource } from '../api'
import { sourceLabel } from './format'

const chip = 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap'

const sourceStyle: Record<SimSource['kind'], string> = {
  profile: 'border-sky-300 bg-sky-50 text-sky-900 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-100',
  'edge-override':
    'border-amber-400 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100',
  'edge-only':
    'border-violet-300 bg-violet-50 text-violet-900 dark:border-violet-800 dark:bg-violet-950/40 dark:text-violet-100',
  customer: 'border-slate-300 bg-slate-100 text-slate-800 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100',
  default: 'border-slate-300 bg-slate-100 text-slate-800 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100',
  unset: 'border-dashed border-slate-400 bg-transparent text-slate-600 dark:border-slate-500 dark:text-slate-300',
}

/** Where a value comes from: Profile, an override on this Edge, a documented default, or nothing. */
export function SourceBadge({ source }: { source: SimSource }) {
  return (
    <span className={`${chip} ${sourceStyle[source.kind]}`} data-source={source.kind} data-source-id={source.id}>
      {sourceLabel(source)}
    </span>
  )
}

const basisStyle: Record<Basis['status'], string> = {
  verified:
    'border-emerald-400 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-100',
  assumption:
    'border-orange-400 bg-orange-50 text-orange-900 dark:border-orange-700 dark:bg-orange-950/40 dark:text-orange-100',
  'not-modelled':
    'border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-100',
}

function basisLabel(b: Basis): string {
  if (b.status === 'verified') return 'verified'
  if (b.status === 'assumption') return b.ref
  return 'not modelled'
}

function basisSource(b: Basis): string {
  if (b.status === 'assumption') return `docs/SIMULATOR-MODEL.md, ${b.ref}`
  return `fact sheet: content/reference/${b.ref}.md`
}

/** A basis chip with a tooltip (hover, focus or tap) naming the fact sheet or assumption behind it. */
export function BasisBadge({ basis, extra = [] }: { basis: Basis; extra?: Basis[] }) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLSpanElement>(null)
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    const onDown = (e: MouseEvent) => !root.current?.contains(e.target as Node) && setOpen(false)
    window.addEventListener('keydown', onKey)
    window.addEventListener('mousedown', onDown)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('mousedown', onDown)
    }
  }, [open])
  const all = [basis, ...extra]
  return (
    <span
      ref={root}
      className="relative inline-block"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        aria-describedby={open ? id : undefined}
        aria-expanded={open}
        data-basis={basis.status}
        data-basis-rule={basis.rule}
        onClick={() => setOpen((o) => !o)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        className={`${chip} ${basisStyle[basis.status]} cursor-help focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none`}
      >
        {basisLabel(basis)}
      </button>
      {open && (
        <span
          role="tooltip"
          id={id}
          className="absolute top-full left-0 z-30 mt-1 w-72 max-w-[80vw] rounded-md border border-slate-300 bg-white p-3 text-left text-xs font-normal whitespace-normal text-slate-800 shadow-lg dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
        >
          {all.map((b, i) => (
            <span
              key={`${b.rule}-${i}`}
              className={`block ${i > 0 ? 'mt-2 border-t border-slate-200 pt-2 dark:border-slate-700' : ''}`}
            >
              <span className="block font-semibold">{basisLabel(b)}</span>
              <span className="block">{b.text}</span>
              <span className="mt-1 block text-slate-500 dark:text-slate-400">{basisSource(b)}</span>
            </span>
          ))}
        </span>
      )}
    </span>
  )
}

/** The primary basis plus any extra assumptions or not-modelled markers (extra verified rows live in the tooltip). */
export function Bases({ basis, also = [] }: { basis: Basis; also?: Basis[] }) {
  const visible = also.filter((b) => b.status !== 'verified')
  const hidden = also.filter((b) => b.status === 'verified')
  return (
    <span className="inline-flex flex-wrap items-center gap-1" data-bases>
      <BasisBadge basis={basis} extra={hidden} />
      {visible.map((b) => (
        <BasisBadge key={b.rule} basis={b} />
      ))}
    </span>
  )
}

/** An explicit "the documentation does not say" box. Never a guess. */
export function NotModelledNote({ nm, className = '' }: { nm: NotModelled; className?: string }) {
  return (
    <div
      data-not-modelled={nm.rule}
      className={`rounded-md border border-dashed border-rose-300 bg-rose-50/60 p-3 text-sm text-rose-950 dark:border-rose-800 dark:bg-rose-950/20 dark:text-rose-100 ${className}`}
    >
      <p className="font-semibold">Not modelled: {nm.item}</p>
      <p className="mt-1">{nm.reason}</p>
      <p className="mt-1 text-xs text-rose-800 dark:text-rose-200">Fact sheet: content/reference/{nm.basis.ref}.md</p>
    </div>
  )
}
