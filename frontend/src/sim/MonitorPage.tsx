import { useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { SimEdge, SimLink, Tunnel } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import ConditionsPanel from './ConditionsPanel'
import { linkSetting } from './conditions'
import { StateBadge } from './overlayUi'
import { modeLabel, typeLabel } from './overlayText'
import { useTopologyCtx } from './topologyContext'
import { useTunnels } from './useTunnels'
import { BasisBadge, Bases } from './ui'

// Tab names come from content/reference/vco-overlay-ui.md and dmpo-vco-ui.md: Monitor > Edges > (Edge) has the tabs
// QoE, Links, Paths, Flows and System. Only Links and Paths have contents in a fact sheet, so only those are built.
const TABS = [
  { id: 'links', label: 'Links' },
  { id: 'paths', label: 'Paths' },
] as const

const GROUPS = ['Public Wired', 'Public Wireless', 'Private Wired'] as const

function groupOf(l: SimLink): (typeof GROUPS)[number] | null {
  if (l.type === 'public' && l.medium === 'wired') return 'Public Wired'
  if (l.type === 'public' && l.medium === 'wireless') return 'Public Wireless'
  if (l.type === 'private' && l.medium === 'wired') return 'Private Wired'
  return null
}

function wanLinks(edge: SimEdge, links: SimLink[]) {
  return edge.interfaces
    .filter((i) => i.kind === 'wan')
    .map((i) => ({ iface: i.id, link: links.find((l) => l.id === i.link) }))
    .filter((x): x is { iface: string; link: SimLink } => !!x.link)
}

/** Mock Monitor > Edges > (Edge) with the Links and Paths tabs. */
export default function MonitorPage() {
  const { detail, topologyId, conditions } = useTopologyCtx()
  const t = detail.topology
  const edges = t.sites.flatMap((s) => s.edges)
  const [params, setParams] = useSearchParams()
  const edge = edges.find((e) => e.id === params.get('edge')) ?? edges[0]
  const tab = TABS.find((x) => x.id === params.get('tab'))?.id ?? 'links'
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([])
  const [grouped, setGrouped] = useState(false)
  const [potential, setPotential] = useState(false)
  const state = useTunnels(topologyId, conditions)

  if (!edge) return <p className="text-sm text-slate-500">This topology has no Edges.</p>
  const select = (i: number) => {
    const n = (i + TABS.length) % TABS.length
    setParams({ edge: edge.id, tab: TABS[n]!.id }, { replace: true })
    tabRefs.current[n]?.focus()
  }
  const links = wanLinks(edge, t.links)
  const mine: Tunnel[] = (state.data?.tunnels ?? []).filter((x) => x.a.id === edge.id || x.b.id === edge.id)

  return (
    <div>
      <p className="text-sm text-slate-500 dark:text-slate-400">Monitor &gt; Edges &gt; {edge.name}</p>
      <div className="mt-1 flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-2 text-sm">
          <span className="font-medium">Edge</span>
          <select
            className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-900"
            value={edge.id}
            onChange={(e) => setParams({ edge: e.target.value, tab }, { replace: true })}
            data-monitor-edge
          >
            {edges.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </select>
        </label>
        {detail.ui_bases.edge_tabs && <BasisBadge basis={detail.ui_bases.edge_tabs} />}
      </div>
      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
        The real Edge page also has QoE, Flows and System tabs. Their contents are not in the fact sheets, so they are
        not built here.
      </p>

      <div className="mt-4 grid gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="min-w-0">
          <div
            role="tablist"
            aria-label="Edge monitoring"
            className="flex gap-1 border-b border-slate-200 dark:border-slate-800"
          >
            {TABS.map((x, i) => (
              <button
                key={x.id}
                ref={(el) => {
                  tabRefs.current[i] = el
                }}
                type="button"
                role="tab"
                id={`mtab-${x.id}`}
                aria-selected={tab === x.id}
                aria-controls={`mpanel-${x.id}`}
                tabIndex={tab === x.id ? 0 : -1}
                onClick={() => setParams({ edge: edge.id, tab: x.id }, { replace: true })}
                onKeyDown={(e) => {
                  if (e.key === 'ArrowRight') {
                    e.preventDefault()
                    select(i + 1)
                  } else if (e.key === 'ArrowLeft') {
                    e.preventDefault()
                    select(i - 1)
                  }
                }}
                className={`-mb-px rounded-t-md border-b-2 px-4 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
                  tab === x.id
                    ? 'border-sky-600 font-semibold text-sky-900 dark:border-sky-400 dark:text-sky-100'
                    : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
                }`}
              >
                {x.label}
              </button>
            ))}
          </div>

          <div
            role="tabpanel"
            id={`mpanel-${tab}`}
            aria-labelledby={`mtab-${tab}`}
            className="mt-4"
            data-monitor-panel={tab}
          >
            {tab === 'links' && (
              <div>
                <div className="flex flex-wrap items-center gap-3">
                  <label className="flex items-center gap-1.5 text-sm">
                    <input
                      type="checkbox"
                      checked={grouped}
                      onChange={(e) => setGrouped(e.target.checked)}
                      data-links-groups
                    />
                    Transport Groups view
                  </label>
                  {detail.ui_bases.links_tab && <BasisBadge basis={detail.ui_bases.links_tab} />}
                  {detail.ui_bases.transport_groups && <BasisBadge basis={detail.ui_bases.transport_groups} />}
                </div>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  The real tab lists per-link metrics. This model has no measurements, so it shows the link and its
                  simulated state only.
                </p>
                {(grouped ? [...GROUPS, null] : [undefined]).map((g) => {
                  const rows = links.filter((l) => (g === undefined ? true : groupOf(l.link) === g))
                  if (g !== undefined && rows.length === 0) return null
                  return (
                    <section key={String(g)} className="mt-3" data-links-group={g ?? 'all'}>
                      {g !== undefined && <h3 className="text-sm font-semibold">{g ?? 'No transport group'}</h3>}
                      <ul className="mt-1 space-y-2">
                        {rows.map((l) => (
                          <li
                            key={l.link.id}
                            data-link-row={l.link.id}
                            data-state={linkSetting(conditions, l.link.id)}
                            className="flex flex-wrap items-center gap-2 rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
                          >
                            <span className="font-medium">{l.link.id}</span>
                            <span className="text-xs text-slate-500 dark:text-slate-400">
                              {l.iface} · {groupOf(l.link) ?? `${l.link.type} ${l.link.medium}`}
                            </span>
                            <StateBadge state={linkSetting(conditions, l.link.id)} />
                          </li>
                        ))}
                      </ul>
                    </section>
                  )
                })}
              </div>
            )}

            {tab === 'paths' && (
              <div>
                {state.error && <ErrorPanel title="The tunnels could not be computed" error={state.error} />}
                <div className="flex flex-wrap items-center gap-3">
                  <label className="flex items-center gap-1.5 text-sm">
                    <input
                      type="checkbox"
                      checked={potential}
                      onChange={(e) => setPotential(e.target.checked)}
                      data-paths-potential
                    />
                    Also list peers with no tunnel
                  </label>
                  {detail.ui_bases.paths_tab && <BasisBadge basis={detail.ui_bases.paths_tab} />}
                  {detail.ui_bases.paths_mode && <BasisBadge basis={detail.ui_bases.paths_mode} />}
                </div>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Mode (Static or Dynamic) is the List Paths column. The state here is this simulator's own (up,
                  degraded, down, not established, torn down); the real VCO shows Stable, Unstable, BW_Measurement or
                  Quiet. IP addresses, latency, jitter, loss and bandwidth are not modelled.
                </p>
                {!state.data && <p className="mt-3 text-slate-500">Loading…</p>}
                <ul className="mt-3 space-y-3" data-paths-list>
                  {mine
                    .filter((x) => x.paths.length > 0 || potential)
                    .map((x) => {
                      const peer = x.a.id === edge.id ? x.b : x.a
                      return (
                        <li
                          key={x.id}
                          data-path-peer={peer.id}
                          data-state={x.state}
                          className="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
                        >
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-medium">{peer.name}</span>
                            <span className="text-xs text-slate-500 dark:text-slate-400">
                              {typeLabel[x.type]} · Mode {modeLabel(x.mode)}
                            </span>
                            <StateBadge state={x.state} />
                          </div>
                          {x.paths.length === 0 ? (
                            <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">No path: {x.reason}</p>
                          ) : (
                            <ul className="mt-2 space-y-1">
                              {x.paths.map((p, i) => (
                                <li key={i} data-path-row className="flex flex-wrap items-center gap-2 text-xs">
                                  <span className="font-medium">WAN link {p.local_link}</span>
                                  <span className="text-slate-500 dark:text-slate-400">to {p.remote}</span>
                                  <StateBadge state={p.state} />
                                  {p.reason && <span className="text-slate-500 dark:text-slate-400">{p.reason}</span>}
                                </li>
                              ))}
                            </ul>
                          )}
                          <div className="mt-2 flex flex-wrap items-center gap-1">
                            <Bases basis={x.basis} also={x.also} />
                          </div>
                        </li>
                      )
                    })}
                </ul>
              </div>
            )}
          </div>
        </div>
        <div className="min-w-0">
          <ConditionsPanel />
        </div>
      </div>
    </div>
  )
}
