import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchLabs, fetchRunbooks, type LabSummary, type RunbookSummary } from '../api'
import ErrorPanel from '../components/ErrorPanel'

const KIND_LABEL: Record<RunbookSummary['kind'], string> = {
  checklist: 'Checklist',
  cheatsheet: 'Cheat sheet',
  template: 'Template',
  reference: 'Reference',
}

type State<T> = { kind: 'loading' } | { kind: 'ready'; value: T } | { kind: 'error'; error: Error }

/** /troubleshooting: the one-page runbooks, and the way into the Section 10 lessons and the troubleshooting labs. Nothing here is locked. */
export default function TroubleshootingPage() {
  const [books, setBooks] = useState<State<RunbookSummary[]>>({ kind: 'loading' })
  const [labs, setLabs] = useState<LabSummary[]>([])

  useEffect(() => {
    fetchRunbooks()
      .then((r) => setBooks({ kind: 'ready', value: r.runbooks }))
      .catch((error: Error) => setBooks({ kind: 'error', error }))
    fetchLabs()
      .then((all) => setLabs(all.filter((l) => l.id.startsWith('ts-'))))
      .catch(() => setLabs([]))
  }, [])

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-semibold">Troubleshooting</h1>
      <p className="mt-2 text-slate-600 dark:text-slate-400">
        One-page aids for working a ticket, the lessons that explain them, and practice tickets. The method and the
        templates are this course&apos;s own; the tools they name are documented.
      </p>

      <h2 className="mt-8 text-lg font-semibold">Learn</h2>
      <p className="mt-1 text-sm">
        <Link className="font-medium underline decoration-sky-400 underline-offset-2" to="/course/troubleshooting">
          Section 10: Troubleshooting, from ticket to fix
        </Link>{' '}
        (study guide section G6, added by the course).
      </p>

      <h2 className="mt-8 text-lg font-semibold">Runbooks</h2>
      {books.kind === 'loading' && <p className="mt-2 text-slate-500">Loading…</p>}
      {books.kind === 'error' && <ErrorPanel title="Runbooks are unavailable" error={books.error} />}
      {books.kind === 'ready' && (
        <ul className="mt-3 grid gap-3 sm:grid-cols-2" data-runbook-list>
          {books.value.map((b) => (
            <li key={b.id}>
              <Link
                to={`/troubleshooting/${b.id}`}
                data-runbook={b.id}
                className="block h-full rounded-lg border border-slate-200 p-4 hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-800 dark:hover:border-sky-600"
              >
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{KIND_LABEL[b.kind]}</span>
                <span className="block font-medium">{b.title}</span>
                <span className="mt-1 block text-sm text-slate-600 dark:text-slate-400">{b.summary}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <h2 className="mt-8 text-lg font-semibold">Practice tickets</h2>
      {labs.length === 0 ? (
        <p className="mt-2 text-sm text-slate-500">No troubleshooting labs are loaded.</p>
      ) : (
        <ul className="mt-3 space-y-2" data-ts-labs>
          {labs.map((l) => (
            <li key={l.id}>
              <Link
                to={`/sim/labs/${l.id}`}
                className="block rounded-lg border border-slate-200 p-3 hover:border-sky-400 dark:border-slate-800 dark:hover:border-sky-600"
              >
                <span className="font-medium">{l.title}</span>
                <span className="mt-0.5 block text-sm text-slate-600 dark:text-slate-400">{l.summary}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
