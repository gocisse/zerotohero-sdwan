import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useGlossary } from '../glossary/context'

/**
 * `<Term id="..." />` (or `<Term id="...">custom text</Term>`): an inline glossary word with a popover holding
 * the short definition and a link to the full entry. Opens on hover, keyboard focus, and tap/click (click pins it
 * open; Escape or a click elsewhere closes it).
 */
export default function Term({ id, children }: { id?: string; children?: ReactNode }) {
  const glossary = useGlossary()
  const [hover, setHover] = useState(false)
  const [focus, setFocus] = useState(false)
  const [pinned, setPinned] = useState(false)
  const wrapRef = useRef<HTMLSpanElement>(null)
  const popId = useId()

  useEffect(() => {
    if (!pinned) return
    const onDown = (e: Event) => {
      if (!wrapRef.current?.contains(e.target as Node)) setPinned(false)
    }
    document.addEventListener('pointerdown', onDown)
    return () => document.removeEventListener('pointerdown', onDown)
  }, [pinned])

  const term = id && glossary.status === 'ready' ? glossary.byId.get(id) : undefined

  if (glossary.status === 'ready' && !term) {
    // The backend validator rejects unknown ids, so this only shows while authoring.
    return import.meta.env.DEV ? (
      <span
        data-term-error
        title={`Unknown glossary term "${id ?? ''}"`}
        className="rounded bg-red-100 px-1 font-mono text-xs text-red-800 dark:bg-red-950 dark:text-red-200"
      >
        ⚠ unknown term “{id ?? '(no id)'}”
      </span>
    ) : (
      <>{children}</>
    )
  }
  if (!term) return <>{children ?? id}</> // glossary still loading or unavailable: plain text

  const open = hover || focus || pinned
  return (
    <span
      ref={wrapRef}
      className="relative inline-block"
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onFocus={() => setFocus(true)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setFocus(false)
      }}
      onKeyDown={(e) => {
        if (e.key === 'Escape') {
          setPinned(false)
          setHover(false)
          setFocus(false)
        }
      }}
    >
      <span
        role="button"
        tabIndex={0}
        aria-expanded={open}
        aria-describedby={open ? popId : undefined}
        data-term={term.id}
        onClick={() => setPinned((p) => !p)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            setPinned((p) => !p)
          }
        }}
        className="cursor-help rounded bg-violet-100 px-0.5 font-medium text-violet-900 underline decoration-violet-400 decoration-dotted underline-offset-2 focus-visible:ring-2 focus-visible:ring-violet-500 focus-visible:outline-none dark:bg-violet-900/40 dark:text-violet-100"
      >
        {children ?? term.term}
      </span>
      {open && (
        <span
          id={popId}
          role="tooltip"
          data-term-popover
          className="not-prose absolute top-full left-0 z-30 mt-1 block w-64 max-w-[80vw] rounded-md border border-slate-200 bg-white p-3 text-left text-sm font-normal text-slate-800 shadow-lg dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
        >
          <span className="block font-semibold">{term.term}</span>
          <span className="mt-0.5 block">{term.short}</span>
          <Link
            to={{ pathname: '/glossary', hash: `#term-${term.id}` }}
            className="mt-2 inline-block text-sky-700 underline dark:text-sky-300"
          >
            Full glossary entry →
          </Link>
        </span>
      )}
    </span>
  )
}
