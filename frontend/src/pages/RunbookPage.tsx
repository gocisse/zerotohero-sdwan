import { lazy, Suspense, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError, fetchRunbook, type Runbook } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import NotFound from './NotFound'

const MdxRenderer = lazy(() => import('../mdx/MdxRenderer'))

type State =
  { kind: 'loading' } | { kind: 'ready'; book: Runbook } | { kind: 'missing' } | { kind: 'error'; error: Error }

/** One runbook page under /troubleshooting/:runbookId. */
export default function RunbookPage() {
  const { runbookId = '' } = useParams()
  return <RunbookBody key={runbookId} id={runbookId} />
}

function RunbookBody({ id }: { id: string }) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false
    fetchRunbook(id)
      .then((book) => !cancelled && setState({ kind: 'ready', book }))
      .catch((error: Error) => {
        if (cancelled) return
        setState(error instanceof ApiError && error.status === 404 ? { kind: 'missing' } : { kind: 'error', error })
      })
    return () => {
      cancelled = true
    }
  }, [id])

  if (state.kind === 'missing') return <NotFound />
  return (
    <article className="max-w-3xl" data-runbook-page={id}>
      <p className="text-sm">
        <Link className="underline decoration-sky-400 underline-offset-2" to="/troubleshooting">
          ← Troubleshooting
        </Link>
      </p>
      {state.kind === 'loading' && <p className="mt-4 text-slate-500">Loading…</p>}
      {state.kind === 'error' && <ErrorPanel title="Could not load this runbook" error={state.error} />}
      {state.kind === 'ready' && (
        <>
          <header className="mt-3 mb-6">
            <h1 className="text-3xl font-semibold">{state.book.title}</h1>
            <p className="mt-2 text-slate-600 dark:text-slate-400">{state.book.summary}</p>
            <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
              {state.book.estimated_minutes} min · the course&apos;s own working aid
            </p>
          </header>
          <div className="prose prose-slate max-w-none dark:prose-invert prose-pre:bg-[#0d1117] prose-pre:p-0 prose-code:before:content-none prose-code:after:content-none">
            <Suspense fallback={<p className="text-slate-500">Rendering…</p>}>
              <MdxRenderer key={state.book.source} source={state.book.source} />
            </Suspense>
          </div>
        </>
      )}
    </article>
  )
}
