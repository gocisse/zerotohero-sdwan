import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import ErrorPanel from '../components/ErrorPanel'
import Markdown from '../components/Markdown'
import { useCourseState } from '../course/context'
import { useGlossary } from '../glossary/context'
import { lessonPath } from '../lib/course'

export default function GlossaryPage() {
  const glossary = useGlossary()
  const course = useCourseState()
  const { hash } = useLocation()
  const [query, setQuery] = useState('')

  const terms = useMemo(
    () =>
      glossary.status === 'ready'
        ? [...glossary.terms].sort((a, b) => a.term.localeCompare(b.term, undefined, { sensitivity: 'base' }))
        : [],
    [glossary],
  )

  // Matches the term or any alias, case-insensitively.
  const q = query.trim().toLowerCase()
  const shown = q ? terms.filter((t) => [t.term, ...t.aliases].some((n) => n.toLowerCase().includes(q))) : terms

  // Scroll to #term-<id> once the entries exist.
  const ready = glossary.status === 'ready'
  useEffect(() => {
    if (ready && hash) document.getElementById(hash.slice(1))?.scrollIntoView({ block: 'start' })
  }, [ready, hash])

  if (glossary.status === 'error') return <ErrorPanel title="The glossary is unavailable" error={glossary.error} />
  if (glossary.status === 'loading') return <p className="text-slate-500">Loading…</p>
  const byId = glossary.byId

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-semibold">Glossary</h1>
      <label className="mt-4 block">
        <span className="sr-only">Search terms</span>
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search terms and aliases…"
          className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900"
        />
      </label>
      <p className="mt-2 text-sm text-slate-500 dark:text-slate-400" aria-live="polite" data-glossary-count>
        {shown.length} of {terms.length} term{terms.length === 1 ? '' : 's'}
      </p>

      <dl className="mt-4 space-y-4">
        {shown.map((t) => {
          const where = t.lesson_ref && course.status === 'ready' ? lessonPath(course.course, t.lesson_ref) : undefined
          return (
            <div
              key={t.id}
              id={`term-${t.id}`}
              data-glossary-term={t.id}
              className={`scroll-mt-4 rounded-lg border p-4 ${hash === `#term-${t.id}` ? 'border-violet-400 bg-violet-50 dark:border-violet-700 dark:bg-violet-950/30' : 'border-slate-200 dark:border-slate-800'}`}
            >
              <dt className="text-lg font-semibold">
                {t.term}
                {t.aliases.length > 0 && (
                  <span className="ml-2 text-sm font-normal text-slate-500 dark:text-slate-400">
                    also: {t.aliases.join(', ')}
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
                          onClick={() => setQuery('')}
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
              </dd>
            </div>
          )
        })}
      </dl>
      {shown.length === 0 && <p className="mt-4 text-slate-600 dark:text-slate-400">No terms match “{query}”.</p>}
    </div>
  )
}
