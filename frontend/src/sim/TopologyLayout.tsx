import { useState } from 'react'
import { NavLink, Outlet, useParams } from 'react-router-dom'
import { fetchTopology, type Conditions, type RoutingAssumptions } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import type { TopologyCtx } from './topologyContext'
import { useAsync } from './useAsync'

const tab = (active: boolean) =>
  `rounded-md px-3 py-1.5 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
    active
      ? 'bg-sky-100 font-medium text-sky-900 dark:bg-sky-900/40 dark:text-sky-100'
      : 'text-slate-600 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800'
  }`

/** /sim/:topologyId: loads the topology once and shares it with the Edge list, what-if and challenge screens. */
export default function TopologyLayout() {
  const { topologyId = '' } = useParams()
  const state = useAsync(() => fetchTopology(topologyId), topologyId)
  const [conditions, setConditions] = useState<{ topologyId: string; value: Conditions }>({ topologyId, value: {} })
  const [assume, setAssume] = useState<{ topologyId: string; value: Partial<RoutingAssumptions> }>({
    topologyId,
    value: {},
  })
  const currentAssume = assume.topologyId === topologyId ? assume.value : {}
  // Conditions belong to one topology: switching topology starts from none.
  const current = conditions.topologyId === topologyId ? conditions.value : {}
  if (state.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (state.status === 'error') {
    return (
      <ErrorPanel title={`Topology "${topologyId}" is unavailable`} error={state.error}>
        <p className="mt-3 text-sm">
          <NavLink to="/sim" className="underline">
            Back to the topology list
          </NavLink>
        </p>
      </ErrorPanel>
    )
  }
  const { topology } = state.data
  return (
    <div>
      <p className="text-sm text-slate-500 dark:text-slate-400">
        <NavLink to="/sim" className="underline">
          Simulator
        </NavLink>{' '}
        / {topology.meta.title}
      </p>
      <h1 className="mt-1 text-2xl font-semibold">{topology.meta.title}</h1>
      <p className="mt-1 max-w-3xl text-sm text-slate-600 dark:text-slate-300">{topology.meta.summary}</p>
      {topology.meta.notes && <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{topology.meta.notes}</p>}
      <nav
        aria-label="Simulator sections"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200 pb-2 dark:border-slate-800"
      >
        <NavLink to={`/sim/${topologyId}`} end className={({ isActive }) => tab(isActive)}>
          Configure &gt; Edges
        </NavLink>
        <NavLink to={`/sim/${topologyId}/what-if`} className={({ isActive }) => tab(isActive)}>
          What if?
        </NavLink>
        <NavLink to={`/sim/${topologyId}/challenge`} className={({ isActive }) => tab(isActive)}>
          Challenge
        </NavLink>
        <span className="mx-1 self-center text-xs text-slate-400" aria-hidden>
          |
        </span>
        <NavLink to={`/sim/${topologyId}/map`} className={({ isActive }) => tab(isActive)}>
          Map
        </NavLink>
        <NavLink to={`/sim/${topologyId}/monitor`} className={({ isActive }) => tab(isActive)}>
          Monitor &gt; Edges
        </NavLink>
        <NavLink to={`/sim/${topologyId}/paths`} className={({ isActive }) => tab(isActive)}>
          Path explainer
        </NavLink>
        <NavLink to={`/sim/${topologyId}/timeline`} className={({ isActive }) => tab(isActive)}>
          Timeline
        </NavLink>
        <NavLink to={`/sim/${topologyId}/overlay-challenge`} className={({ isActive }) => tab(isActive)}>
          Overlay challenge
        </NavLink>
        <span className="mx-1 self-center text-xs text-slate-400" aria-hidden>
          |
        </span>
        <NavLink to={`/sim/${topologyId}/routes`} className={({ isActive }) => tab(isActive)}>
          Route Table Dump
        </NavLink>
        <NavLink to={`/sim/${topologyId}/reachability`} className={({ isActive }) => tab(isActive)}>
          Reachability explainer
        </NavLink>
        <NavLink to={`/sim/${topologyId}/routing-challenge`} className={({ isActive }) => tab(isActive)}>
          Routing challenge
        </NavLink>
      </nav>
      {state.data.warnings.length > 0 && (
        <ul className="mt-3 space-y-1 text-xs text-amber-800 dark:text-amber-200" data-topology-warnings>
          {state.data.warnings.map((w, i) => (
            <li key={i}>
              Warning in the topology file: <code>{w.path}</code>: {w.message}
            </li>
          ))}
        </ul>
      )}
      <div className="mt-5">
        <Outlet
          context={
            {
              detail: state.data,
              topologyId,
              conditions: current,
              setConditions: (value: Conditions) => setConditions({ topologyId, value }),
              assumptions: currentAssume,
              setAssumptions: (value: Partial<RoutingAssumptions>) => setAssume({ topologyId, value }),
            } satisfies TopologyCtx
          }
        />
      </div>
    </div>
  )
}
