import { Link } from 'react-router-dom'
import { fetchTopologies } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useAsync } from './useAsync'

/** /sim: choose a teaching topology. */
export default function TopologyPicker() {
  const state = useAsync(fetchTopologies, 'topologies')
  return (
    <div>
      <h1 className="text-2xl font-semibold">Simulator</h1>
      <p className="mt-2 max-w-2xl text-slate-600 dark:text-slate-300">
        Practice reading an Orchestrator: why does this Edge have this setting, and where does it come from? Pick a
        teaching topology. Each one is an invented network, not a real customer.
      </p>
      {state.status === 'loading' && <p className="mt-6 text-slate-500">Loading…</p>}
      {state.status === 'error' && (
        <div className="mt-6">
          <ErrorPanel title="The simulator topologies are unavailable" error={state.error} />
        </div>
      )}
      {state.status === 'ready' && (
        <ul className="mt-6 grid gap-4 sm:grid-cols-2" data-topology-list>
          {state.data.map((t) => (
            <li
              key={t.id}
              className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
              data-topology={t.id}
            >
              <h2 className="text-lg font-semibold">
                <Link
                  to={`/sim/${t.id}`}
                  className="underline decoration-sky-400 underline-offset-2 hover:text-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:hover:text-sky-300"
                >
                  {t.title}
                </Link>
              </h2>
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{t.summary}</p>
              <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">
                {t.sites} sites · {t.edges} Edges · {t.profiles} Profile{t.profiles === 1 ? '' : 's'} · teaching
                topology
              </p>
              {t.warnings.length > 0 && (
                <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
                  {t.warnings.length} warning{t.warnings.length === 1 ? '' : 's'} in this topology file
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
