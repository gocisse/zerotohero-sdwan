import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import type { GlossaryTerm, StudyGuide } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import Markdown from '../components/Markdown'
import { useCourseState } from '../course/context'
import { useGlossary } from '../glossary/context'
import { lessonPath } from '../lib/course'
import { useStudyGuide } from '../studyguide/context'
import { BEYOND_KEY, groupGlossary, primaryGroup, splitMatches, termMatches } from '../studyguide/glossaryGroups'

type View = 'guide' | 'az'

const alpha = (a: GlossaryTerm, b: GlossaryTerm) => a.term.localeCompare(b.term, undefined, { sensitivity: 'base' })

function Highlight({ text, query }: { text: string; query: string }) {
  return (
    <>
      {splitMatches(text, query).map((p, i) =>
        p.match ? (
          <mark key={i} className="rounded bg-yellow-200 px-0.5 text-inherit dark:bg-yellow-700/60">
            {p.text}
          </mark>
        ) : (
          <span key={i}>{p.text}</span>
        ),
      )}
    </>
  )
}

function Chip({ label, onClick, active }: { label: string; onClick: () => void; active?: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      title="Show only this study guide topic"
      className={`rounded-full border px-2 py-0.5 text-xs focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
        active
          ? 'border-sky-500 bg-sky-100 text-sky-900 dark:border-sky-400 dark:bg-sky-900/40 dark:text-sky-100'
          : 'border-slate-300 text-slate-600 hover:border-sky-400 dark:border-slate-700 dark:text-slate-300'
      }`}
    >
      {label}
    </button>
  )
}

export default function GlossaryPage() {
  const glossary = useGlossary()
  const course = useCourseState()
  const sg = useStudyGuide()
  const { hash } = useLocation()
  const [query, setQuery] = useState('')
  const [view, setView] = useState<View>('guide')
  const [topicFilter, setTopicFilter] = useState<string | null>(null)
  const [open, setOpen] = useState<Set<string>>(new Set())

  const guide: StudyGuide | null = sg.status === 'ready' ? sg.guide : null
  const effectiveView: View = guide ? view : 'az'
  const ready = glossary.status === 'ready'
  const terms = useMemo(() => (glossary.status === 'ready' ? [...glossary.terms].sort(alpha) : []), [glossary])
  const sections = useMemo(() => (guide ? groupGlossary(guide, terms) : []), [guide, terms])
  const topicTitle = useMemo(() => {
    const m = new Map<string, string>([[BEYOND_KEY, guide?.beyond.title ?? '']])
    for (const s of guide?.sections ?? []) for (const t of s.topics) m.set(t.id, t.title)
    return m
  }, [guide])

  const q = query.trim()
  const matching = q ? terms.filter((t) => termMatches(t, q)) : terms

  // A deep link (#term-<id>) opens the term's primary group. State is adjusted during render, when the hash changes.
  const [seenHash, setSeenHash] = useState('')
  // Wait for the study guide too: handling the hash while the view is still A-Z would mark it done without opening a group.
  if (ready && sg.status !== 'loading' && hash !== seenHash) {
    setSeenHash(hash)
    const t = hash.startsWith('#term-') ? terms.find((x) => `#term-${x.id}` === hash) : undefined
    if (t && effectiveView === 'guide' && !open.has(primaryGroup(t))) setOpen(new Set(open).add(primaryGroup(t)))
  }
  // Then the target is scrolled into view once it is rendered.
  useEffect(() => {
    if (!ready || !hash) return
    const raf = requestAnimationFrame(() => document.getElementById(hash.slice(1))?.scrollIntoView({ block: 'start' }))
    return () => cancelAnimationFrame(raf)
  }, [ready, hash, seenHash])

  if (glossary.status === 'error') return <ErrorPanel title="The glossary is unavailable" error={glossary.error} />
  if (glossary.status === 'loading' || sg.status === 'loading') return <p className="text-slate-500">Loading…</p>
  const byId = glossary.byId

  const toggle = (key: string) =>
    setOpen((prev) => {
      const next = new Set(prev)
      if (!next.delete(key)) next.add(key)
      return next
    })
  const filterTo = (key: string) => {
    setView('guide')
    setTopicFilter(key)
    setOpen((prev) => new Set(prev).add(key))
  }
  const jump = (key: string) => {
    const el = document.getElementById(`guide-${key}`)
    el?.scrollIntoView({ block: 'start' })
    if (el) history.replaceState(null, '', `#guide-${key}`)
  }

  const chips = (t: GlossaryTerm) => {
    const keys = t.guide_topics.length > 0 ? t.guide_topics : [BEYOND_KEY]
    return (
      <p className="mt-2 flex flex-wrap items-center gap-1.5" data-term-chips>
        <span className="sr-only">Study guide topics:</span>
        {keys.map((k) => (
          <Chip key={k} label={topicTitle.get(k) ?? k} active={topicFilter === k} onClick={() => filterTo(k)} />
        ))}
      </p>
    )
  }

  const entry = (t: GlossaryTerm, primary: boolean) => {
    const where = t.lesson_ref && course.status === 'ready' ? lessonPath(course.course, t.lesson_ref) : undefined
    return (
      <div
        key={t.id}
        id={primary ? `term-${t.id}` : undefined}
        data-glossary-term={t.id}
        className={`scroll-mt-4 rounded-lg border p-4 ${hash === `#term-${t.id}` && primary ? 'border-violet-400 bg-violet-50 dark:border-violet-700 dark:bg-violet-950/30' : 'border-slate-200 dark:border-slate-800'}`}
      >
        <dt className="text-lg font-semibold">
          <Highlight text={t.term} query={q} />
          {t.aliases.length > 0 && (
            <span className="ml-2 text-sm font-normal text-slate-500 dark:text-slate-400">
              also: <Highlight text={t.aliases.join(', ')} query={q} />
            </span>
          )}
        </dt>
        <dd className="mt-1">
          <p>{t.short}</p>
          {t.long && <Markdown className="mt-2 text-sm">{t.long}</Markdown>}
          {t.related.length > 0 && (
            <p className="mt-2 text-sm">
              Related:{' '}
              {t.related.map((r, i) => (
                <span key={r}>
                  {i > 0 && ', '}
                  <Link
                    to={{ pathname: '/glossary', hash: `#term-${r}` }}
                    onClick={() => {
                      setQuery('')
                      setTopicFilter(null)
                    }}
                    className="text-sky-700 underline dark:text-sky-300"
                  >
                    {byId.get(r)?.term ?? r}
                  </Link>
                </span>
              ))}
            </p>
          )}
          {where && (
            <p className="mt-1 text-sm">
              <Link to={where} className="text-sky-700 underline dark:text-sky-300">
                Where this is taught →
              </Link>
            </p>
          )}
          {guide && chips(t)}
        </dd>
      </div>
    )
  }

  const count = (
    <p className="mt-2 text-sm text-slate-500 dark:text-slate-400" aria-live="polite" data-glossary-count>
      {matching.length} of {terms.length} term{terms.length === 1 ? '' : 's'}
    </p>
  )

  // ---- A to Z ----
  if (effectiveView === 'az') {
    return (
      <div className="max-w-3xl">
        <h1 className="text-2xl font-semibold">Glossary</h1>
        {renderToolbar()}
        {count}
        <dl className="mt-4 space-y-4" data-glossary-view="az">
          {matching.map((t) => entry(t, true))}
        </dl>
        {matching.length === 0 && <p className="mt-4 text-slate-600 dark:text-slate-400">No terms match “{query}”.</p>}
      </div>
    )
  }

  // ---- by study guide ----
  const visible = sections
    .map((s) => ({
      ...s,
      groups: s.groups
        .filter((g) => (topicFilter ? g.key === topicFilter : true))
        .map((g) => ({ ...g, matches: q ? g.terms.filter((t) => termMatches(t, q)) : g.terms }))
        .filter((g) => !q || g.matches.length > 0),
    }))
    .filter((s) => s.groups.length > 0)
  const isOpen = (key: string, hasMatches: boolean) => open.has(key) || (q !== '' && hasMatches) || topicFilter === key
  const sectionCount = (s: (typeof visible)[number]) =>
    new Set(s.groups.flatMap((g) => g.matches.map((t) => t.id))).size
  const sectionLabel = (s: (typeof visible)[number]) => (s.key === BEYOND_KEY ? s.title : `${s.key} · ${s.title}`)

  return (
    <div className="max-w-5xl">
      <h1 className="text-2xl font-semibold">Glossary</h1>
      {renderToolbar()}
      {count}
      {topicFilter && (
        <p className="mt-2 flex flex-wrap items-center gap-2 text-sm" data-topic-filter={topicFilter}>
          Showing only <strong>{topicTitle.get(topicFilter) ?? topicFilter}</strong>
          <button
            type="button"
            onClick={() => setTopicFilter(null)}
            className="rounded-md border border-slate-300 px-2 py-0.5 text-xs hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
          >
            Show all topics ✕
          </button>
        </p>
      )}

      <div className="mt-4 lg:grid lg:grid-cols-[15rem_minmax(0,1fr)] lg:gap-8">
        <div>
          <label className="block lg:hidden">
            <span className="text-sm text-slate-600 dark:text-slate-400">Jump to section</span>
            <select
              aria-label="Jump to section"
              defaultValue=""
              onChange={(e) => e.target.value && jump(e.target.value)}
              className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900"
            >
              <option value="" disabled>
                Choose a section…
              </option>
              {visible.map((s) => (
                <option key={s.key} value={s.key}>
                  {sectionLabel(s)} ({sectionCount(s)})
                </option>
              ))}
            </select>
          </label>
          <nav aria-label="Study guide sections" className="hidden lg:sticky lg:top-0 lg:block" data-glossary-index>
            <ul className="space-y-1 text-sm">
              {visible.map((s) => (
                <li key={s.key}>
                  <Link
                    to={{ pathname: '/glossary', hash: `#guide-${s.key}` }}
                    className="flex justify-between gap-2 rounded-md px-2 py-1.5 text-slate-700 hover:bg-slate-100 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:text-slate-200 dark:hover:bg-slate-800"
                  >
                    <span>{sectionLabel(s)}</span>
                    <span className="text-slate-500 dark:text-slate-400">{sectionCount(s)}</span>
                  </Link>
                </li>
              ))}
            </ul>
            <p className="mt-3 flex gap-2 px-2 text-xs">
              <button
                type="button"
                className="underline"
                onClick={() => setOpen(new Set(sections.flatMap((s) => s.groups.map((g) => g.key))))}
              >
                Expand all
              </button>
              <button type="button" className="underline" onClick={() => setOpen(new Set())}>
                Collapse all
              </button>
            </p>
          </nav>
        </div>

        <div data-glossary-view="guide" className="mt-4 min-w-0 lg:mt-0">
          {visible.map((s) => (
            <section key={s.key} id={`guide-${s.key}`} data-guide-section={s.key} className="mb-8 scroll-mt-4">
              <h2 className="text-lg font-semibold">
                {sectionLabel(s)}{' '}
                <span className="text-sm font-normal text-slate-500 dark:text-slate-400">
                  ({sectionCount(s)} terms)
                </span>
              </h2>
              {s.key === BEYOND_KEY && guide && (
                <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{guide.beyond.reason}</p>
              )}
              <div className="mt-2 space-y-2">
                {s.groups.map((g) => {
                  const expanded = isOpen(g.key, g.matches.length > 0)
                  return (
                    <div
                      key={g.key}
                      id={`topic-${g.key}`}
                      data-guide-topic={g.key}
                      className="rounded-lg border border-slate-200 dark:border-slate-800"
                    >
                      <h3>
                        <button
                          type="button"
                          aria-expanded={expanded}
                          aria-controls={`panel-${g.key}`}
                          onClick={() => toggle(g.key)}
                          className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-medium hover:bg-slate-100 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:hover:bg-slate-800"
                        >
                          <span aria-hidden className={`text-xs transition-transform ${expanded ? 'rotate-90' : ''}`}>
                            ▶
                          </span>
                          <span className="flex-1">{g.title}</span>
                          <span className="text-xs font-normal text-slate-500 dark:text-slate-400" data-group-count>
                            {g.matches.length} term{g.matches.length === 1 ? '' : 's'}
                          </span>
                        </button>
                      </h3>
                      {expanded && (
                        <dl id={`panel-${g.key}`} className="space-y-3 px-3 pb-3" data-group-terms>
                          {g.matches.length === 0 && (
                            <p className="text-sm text-slate-600 dark:text-slate-400">
                              No glossary terms for this topic yet.
                            </p>
                          )}
                          {g.matches.map((t) => entry(t, primaryGroup(t) === g.key))}
                        </dl>
                      )}
                    </div>
                  )
                })}
              </div>
            </section>
          ))}
          {visible.length === 0 && <p className="text-slate-600 dark:text-slate-400">No terms match “{query}”.</p>}
        </div>
      </div>
    </div>
  )

  // A plain function, not a component: a component declared here would remount (and drop focus) on every keystroke.
  function renderToolbar() {
    return (
      <>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <div
            role="group"
            aria-label="Glossary view"
            className="inline-flex overflow-hidden rounded-md border border-slate-300 text-sm dark:border-slate-700"
          >
            {(
              [
                ['guide', 'By study guide'],
                ['az', 'A-Z'],
              ] as const
            ).map(([v, label]) => (
              <button
                key={v}
                type="button"
                aria-pressed={effectiveView === v}
                disabled={v === 'guide' && !guide}
                onClick={() => {
                  setView(v)
                  setTopicFilter(null)
                }}
                className={`px-3 py-1.5 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-50 ${
                  effectiveView === v
                    ? 'bg-sky-600 text-white dark:bg-sky-500'
                    : 'hover:bg-slate-100 dark:hover:bg-slate-800'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
          {!guide && <span className="text-sm text-slate-500">The study guide view is unavailable right now.</span>}
        </div>
        <label className="mt-3 block">
          <span className="sr-only">Search terms</span>
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search terms and aliases…"
            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900"
          />
        </label>
      </>
    )
  }
}
