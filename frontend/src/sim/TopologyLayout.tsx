import { useState } from 'react'
import { NavLink, Outlet, useParams, useSearchParams } from 'react-router-dom'
import { fetchTopology, setLabContext, type Conditions, type TraceAssumptions } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import type { TopologyCtx } from './topologyContext'
import { useAsync } from './useAsync'

const tab = (active: boolean) =>
  `rounded-md px-3 py-1.5 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
    active
      ? 'bg-sky-100 font-medium text-sky-900 dark:bg-sky-900/40 dark:text-sky-100'
      : 'text-slate-600 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800'
  }`

/** /sim/:topologyId: loads the topology once and shares it with the Edge list, what-if and challenge screens. With
 * ?lab=<id>&fixes=a,b the screens show a lab's faulted state instead of the authored topology. */
export default function TopologyLayout() {
  const { topologyId = '' } = useParams()
  const [params] = useSearchParams()
  const lab = params.get('lab') ?? ''
  const fixes = params.get('fixes') ?? ''
  // The API client attaches the lab to every request of this layout's screens.
  setLabContext(lab ? { lab, fixes } : null)
  const labKey = `${topologyId}|${lab}|${fixes}`
  const labSearch = lab ? `?${new URLSearchParams(fixes ? { lab, fixes } : { lab }).toString()}` : ''
  const state = useAsync(() => fetchTopology(topologyId), labKey)
  // What the reader changed; until then the lab's own starting conditions and assumptions (or none) apply.
  const [condEdit, setCondEdit] = useState<{ key: string; value: Conditions } | null>(null)
  const [assumeEdit, setAssumeEdit] = useState<{ key: string; value: Partial<TraceAssumptions> } | null>(null)
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
  // Conditions and assumption switches start from the lab's fault; the first edit takes over (a what-if, nothing stored).
  const current = condEdit?.key === labKey ? condEdit.value : (state.data.lab?.conditions ?? {})
  const currentAssume =
    assumeEdit?.key === labKey ? assumeEdit.value : ((state.data.lab?.assumptions ?? {}) as Partial<TraceAssumptions>)
  return (
    <div>
      {state.data.lab && (
        <p
          role="note"
          data-lab-banner
          className="mb-3 rounded-md border border-sky-300 bg-sky-50 px-3 py-2 text-sm text-sky-950 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-100"
        >
          {state.data.lab.id.startsWith('self-check-')
            ? 'You are looking at the network of a self-check task.'
            : `You are looking at the network of a lab, with its fault${state.data.lab.fixes.length > 0 ? ' and the fixes you applied' : ''}.`}{' '}
          <NavLink
            to={state.data.lab.id.startsWith('self-check-') ? '/sim/self-check' : `/sim/labs/${state.data.lab.id}`}
            className="font-medium underline"
          >
            {state.data.lab.id.startsWith('self-check-') ? 'Back to the self-check' : 'Back to the lab'}
          </NavLink>
        </p>
      )}
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
        <NavLink to={`/sim/${topologyId}${labSearch}`} end className={({ isActive }) => tab(isActive)}>
          Configure &gt; Edges
        </NavLink>
        <NavLink to={`/sim/${topologyId}/what-if${labSearch}`} className={({ isActive }) => tab(isActive)}>
          What if?
        </NavLink>
        <NavLink to={`/sim/${topologyId}/challenge${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Challenge
        </NavLink>
        <span className="mx-1 self-center text-xs text-slate-400" aria-hidden>
          |
        </span>
        <NavLink to={`/sim/${topologyId}/map${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Map
        </NavLink>
        <NavLink to={`/sim/${topologyId}/monitor${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Monitor &gt; Edges
        </NavLink>
        <NavLink to={`/sim/${topologyId}/paths${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Path explainer
        </NavLink>
        <NavLink to={`/sim/${topologyId}/timeline${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Timeline
        </NavLink>
        <NavLink to={`/sim/${topologyId}/overlay-challenge${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Overlay challenge
        </NavLink>
        <span className="mx-1 self-center text-xs text-slate-400" aria-hidden>
          |
        </span>
        <NavLink to={`/sim/${topologyId}/routes${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Route Table Dump
        </NavLink>
        <NavLink to={`/sim/${topologyId}/reachability${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Reachability explainer
        </NavLink>
        <NavLink to={`/sim/${topologyId}/routing-challenge${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Routing challenge
        </NavLink>
        <span className="mx-1 self-center text-xs text-slate-400" aria-hidden>
          |
        </span>
        <NavLink to={`/sim/${topologyId}/trace${labSearch}`} className={({ isActive }) => tab(isActive)}>
          Flow tracer
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
              setConditions: (value: Conditions) => setCondEdit({ key: labKey, value }),
              assumptions: currentAssume,
              setAssumptions: (value: Partial<TraceAssumptions>) => setAssumeEdit({ key: labKey, value }),
              labSearch,
            } satisfies TopologyCtx
          }
        />
      </div>
    </div>
  )
}
