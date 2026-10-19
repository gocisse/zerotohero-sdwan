import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchHealth, type Health } from '../api'
import { flattenLessons, useCourseState } from '../course/context'

type State = { kind: 'loading' } | { kind: 'ready'; health: Health } | { kind: 'unreachable'; message: string }

function Badge({ label, status }: { label: string; status: 'ok' | 'error' | 'loading' }) {
  const styles = {
    ok: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200',
    error: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200',
    loading: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300',
  }[status]
  const text = status === 'ok' ? 'OK' : status === 'error' ? 'ERROR' : '…'
  return (
    <div className="flex items-center justify-between rounded-lg border border-slate-200 px-4 py-3 dark:border-slate-800">
      <span className="font-medium">{label}</span>
      <span className={`rounded-full px-3 py-0.5 text-xs font-semibold ${styles}`}>{text}</span>
    </div>
  )
}

export default function Home() {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const course = useCourseState()
  const first = course.status === 'ready' ? flattenLessons(course.course)[0] : undefined

  useEffect(() => {
    let cancelled = false
    fetchHealth()
      .then((health) => !cancelled && setState({ kind: 'ready', health }))
      .catch(
        (err: unknown) =>
          !cancelled && setState({ kind: 'unreachable', message: err instanceof Error ? err.message : String(err) }),
      )
    return () => {
      cancelled = true
    }
  }, [])

  const backend = state.kind === 'loading' ? 'loading' : state.kind === 'unreachable' ? 'error' : 'ok'
  const database =
    state.kind === 'ready' ? state.health.database.status : state.kind === 'loading' ? 'loading' : 'error'

  return (
    <div className="max-w-xl">
      <h1 className="text-2xl font-semibold">Welcome to VeloCloud Academy</h1>
      <p className="mt-2 text-slate-600 dark:text-slate-400">
        The course content is coming in a later step. System status:
      </p>
      {first && (
        <Link
          to={`/course/${first.sectionId}/${first.id}`}
          className="mt-5 inline-block rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
        >
          Start learning →
        </Link>
      )}
      {course.status === 'error' && (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400">
          Course content unavailable: {course.error.message}
        </p>
      )}
      <div className="mt-6 space-y-3">
        <Badge label="Backend" status={backend} />
        <Badge label="Database" status={database} />
      </div>
      {state.kind === 'unreachable' && (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400">Could not reach the backend: {state.message}</p>
      )}
      {state.kind === 'ready' && state.health.database.error && (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400">Database: {state.health.database.error}</p>
      )}
    </div>
  )
}
