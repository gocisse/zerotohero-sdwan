import { useRef } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { fetchEffectiveConfig } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import BusinessPolicyTab from './BusinessPolicyTab'
import DeviceTab from './DeviceTab'
import FirewallTab from './FirewallTab'
import { useTopologyCtx } from './topologyContext'
import { useAsync } from './useAsync'

// Tab names come from content/reference/vco-config-ui.md: an Edge has Device, Business Policy and Firewall tabs.
// The real Edge also has an Overview tab whose contents were not read, so it is deliberately not built here.
const TABS = [
  { id: 'device', label: 'Device' },
  { id: 'business-policy', label: 'Business Policy' },
  { id: 'firewall', label: 'Firewall' },
] as const

/** Mock Edge editor: /sim/:topologyId/edges/:edgeId?tab=device|business-policy|firewall. */
export default function EdgePage() {
  const { topologyId, detail } = useTopologyCtx()
  const { edgeId = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([])
  const active = TABS.find((t) => t.id === params.get('tab'))?.id ?? 'device'
  const state = useAsync(() => fetchEffectiveConfig(topologyId, edgeId), `${topologyId}/${edgeId}`)

  const select = (i: number) => {
    const t = TABS[(i + TABS.length) % TABS.length]
    if (!t) return
    setParams({ tab: t.id }, { replace: true })
    tabRefs.current[(i + TABS.length) % TABS.length]?.focus()
  }

  const known = detail.topology.sites.some((s) => s.edges.some((e) => e.id === edgeId))
  if (!known) {
    return (
      <ErrorPanel title={`Edge "${edgeId}" is not in this topology`}>
        <p className="mt-2 text-sm">
          <Link to={`/sim/${topologyId}`} className="underline">
            Back to Configure &gt; Edges
          </Link>
        </p>
      </ErrorPanel>
    )
  }
  if (state.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (state.status === 'error')
    return <ErrorPanel title="Could not compute this Edge's configuration" error={state.error} />
  const ec = state.data

  return (
    <div>
      <p className="text-sm text-slate-500 dark:text-slate-400">
        <Link to={`/sim/${topologyId}`} className="underline">
          Configure &gt; Edges
        </Link>{' '}
        / {ec.edge.name}
      </p>
      <h2 className="mt-1 text-xl font-semibold" data-edge-title>
        {ec.edge.name}
      </h2>
      <p className="text-sm text-slate-600 dark:text-slate-300">
        Site {ec.edge.site} · role {ec.edge.role} · Profile{' '}
        <span className="font-medium" data-edge-profile>
          {ec.edge.profile_name}
        </span>
      </p>

      <div
        role="tablist"
        aria-label="Edge configuration"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200 dark:border-slate-800"
      >
        {TABS.map((t, i) => (
          <button
            key={t.id}
            ref={(el) => {
              tabRefs.current[i] = el
            }}
            type="button"
            role="tab"
            id={`tab-${t.id}`}
            aria-selected={active === t.id}
            aria-controls={`panel-${t.id}`}
            tabIndex={active === t.id ? 0 : -1}
            onClick={() => setParams({ tab: t.id }, { replace: true })}
            onKeyDown={(e) => {
              if (e.key === 'ArrowRight') {
                e.preventDefault()
                select(i + 1)
              } else if (e.key === 'ArrowLeft') {
                e.preventDefault()
                select(i - 1)
              } else if (e.key === 'Home') {
                e.preventDefault()
                select(0)
              } else if (e.key === 'End') {
                e.preventDefault()
                select(TABS.length - 1)
              }
            }}
            className={`-mb-px rounded-t-md border-b-2 px-4 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
              active === t.id
                ? 'border-sky-600 font-semibold text-sky-900 dark:border-sky-400 dark:text-sky-100'
                : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div
        role="tabpanel"
        id={`panel-${active}`}
        aria-labelledby={`tab-${active}`}
        className="mt-4"
        data-tabpanel={active}
      >
        {active === 'device' && <DeviceTab ec={ec} />}
        {active === 'business-policy' && <BusinessPolicyTab ec={ec} />}
        {active === 'firewall' && <FirewallTab ec={ec} />}
      </div>
    </div>
  )
}
