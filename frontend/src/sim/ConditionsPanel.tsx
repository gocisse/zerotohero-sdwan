import { useState } from 'react'
import type { Conditions } from '../api'
import { useTopologyCtx } from './topologyContext'
import { isEmptyConditions, linkSetting, withLink, withToggled, type LinkSetting } from './conditions'
import { BasisBadge } from './ui'

const field =
  'max-w-full rounded-md border border-slate-300 bg-white px-2 py-1 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

/**
 * The Conditions panel: a what-if tool. It overlays link, Hub, Gateway, destination and Orchestrator conditions on the
 * topology and sends them with each request; nothing is stored and the topology never changes.
 */
export default function ConditionsPanel({ wide = false }: { wide?: boolean }) {
  const { detail, conditions, setConditions } = useTopologyCtx()
  const t = detail.topology
  const edges = t.sites.flatMap((s) => s.edges)
  const hubIds = new Set<string>(edges.filter((e) => e.role === 'hub').map((e) => e.id))
  for (const p of t.profiles) for (const h of p.cloud_vpn?.branch_to_hub?.hubs ?? []) hubIds.add(h)
  for (const c of t.clusters ?? []) if (hubIds.has(c.id)) c.members.forEach((m) => hubIds.add(m))
  const hubs = edges.filter((e) => hubIds.has(e.id))
  const nsds = t.non_sdwan_destinations ?? []
  const set = (c: Conditions) => setConditions(c)
  const [trafficA, setTrafficA] = useState(edges[0]?.id ?? '')
  const [trafficB, setTrafficB] = useState(edges[1]?.id ?? '')
  const [silent, setSilent] = useState('5')

  return (
    <section
      aria-labelledby="cond-h"
      className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
      data-conditions
    >
      <div className="flex items-center justify-between gap-2">
        <h2 id="cond-h" className="text-base font-semibold">
          Conditions
        </h2>
        <button
          type="button"
          onClick={() => set({})}
          disabled={isEmptyConditions(conditions)}
          className="rounded-md border border-slate-300 px-2 py-1 text-xs hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-50 dark:border-slate-700"
          data-cond-reset
        >
          Reset
        </button>
      </div>
      <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
        What-if only: nothing is stored and the topology never changes. Reload the page to start again.
      </p>

      <fieldset className="mt-3 min-w-0">
        <legend className="text-sm font-medium">WAN links</legend>
        <div className={`mt-1 grid grid-cols-1 gap-x-6 gap-y-1 ${wide ? 'sm:grid-cols-2 lg:grid-cols-3' : ''}`}>
          {t.links.map((l) => (
            <label key={l.id} className="flex items-center justify-between gap-2 text-sm">
              <span className="min-w-0 flex-1">
                {l.id}{' '}
                <span className="text-xs text-slate-500 dark:text-slate-400">
                  ({l.type} {l.medium})
                </span>
              </span>
              <select
                className={field}
                value={linkSetting(conditions, l.id)}
                onChange={(e) => set(withLink(conditions, l.id, e.target.value as LinkSetting))}
                data-cond-link={l.id}
                aria-label={`State of link ${l.id}`}
              >
                <option value="up">up</option>
                <option value="degraded">degraded</option>
                <option value="down">down</option>
              </select>
            </label>
          ))}
        </div>
        <p className="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
          Degraded means the link is up but misses its SLA; the numbers behind that are not modelled.
          {detail.ui_bases.degraded && <BasisBadge basis={detail.ui_bases.degraded} />}
          {detail.ui_bases.sla_thresholds && <BasisBadge basis={detail.ui_bases.sla_thresholds} />}
        </p>
      </fieldset>

      {hubs.length > 0 && (
        <fieldset className="mt-3 min-w-0">
          <legend className="text-sm font-medium">Hubs unreachable</legend>
          <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">
            {hubs.map((h) => (
              <label key={h.id} className="flex items-center gap-1.5 text-sm">
                <input
                  type="checkbox"
                  checked={!!conditions.hubs_unreachable?.includes(h.id)}
                  onChange={(e) => set(withToggled(conditions, 'hubs_unreachable', h.id, e.target.checked))}
                  data-cond-hub={h.id}
                />
                {h.name}
              </label>
            ))}
          </div>
        </fieldset>
      )}
      {t.gateways.length > 0 && (
        <fieldset className="mt-3 min-w-0">
          <legend className="text-sm font-medium">Gateways unreachable</legend>
          <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">
            {t.gateways.map((g) => (
              <label key={g.id} className="flex items-center gap-1.5 text-sm">
                <input
                  type="checkbox"
                  checked={!!conditions.gateways_unreachable?.includes(g.id)}
                  onChange={(e) => set(withToggled(conditions, 'gateways_unreachable', g.id, e.target.checked))}
                  data-cond-gateway={g.id}
                />
                {g.id}
              </label>
            ))}
          </div>
        </fieldset>
      )}
      {nsds.length > 0 && (
        <fieldset className="mt-3 min-w-0">
          <legend className="text-sm font-medium">Non SD-WAN destinations unreachable</legend>
          <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">
            {nsds.map((n) => (
              <label key={n.id} className="flex items-center gap-1.5 text-sm">
                <input
                  type="checkbox"
                  checked={!!conditions.nsds_unreachable?.includes(n.id)}
                  onChange={(e) => set(withToggled(conditions, 'nsds_unreachable', n.id, e.target.checked))}
                  data-cond-nsd={n.id}
                />
                {n.name ?? n.id}
              </label>
            ))}
          </div>
        </fieldset>
      )}

      <div className="mt-3">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={!!conditions.orchestrator_unreachable}
            onChange={(e) => set({ ...conditions, orchestrator_unreachable: e.target.checked })}
            data-cond-orchestrator
          />
          Orchestrator unreachable
        </label>
        {conditions.orchestrator_unreachable && (
          <p className="mt-1 text-xs text-rose-800 dark:text-rose-200" data-cond-orchestrator-note>
            Not modelled: what an Edge does when the Orchestrator is unreachable is not documented, so every tunnel
            shows as unknown.
          </p>
        )}
      </div>

      <fieldset className="mt-3 min-w-0">
        <legend className="text-sm font-medium">Dynamic tunnel build decision (stub)</legend>
        <div className="mt-1 flex flex-wrap items-center gap-2">
          <select
            className={field}
            value={conditions.dynamic_build ?? 'auto'}
            onChange={(e) => set({ ...conditions, dynamic_build: e.target.value as Conditions['dynamic_build'] })}
            data-cond-build
            aria-label="Build decision"
          >
            <option value="auto">auto</option>
            <option value="always">always</option>
            <option value="never">never</option>
          </select>
          {detail.ui_bases.build_stub && <BasisBadge basis={detail.ui_bases.build_stub} />}
          {detail.ui_bases.secure_tunnel && <BasisBadge basis={detail.ui_bases.secure_tunnel} />}
        </div>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Auto: blocked only by a NAT that blocks incoming UDP 2426 at the peer. Always and never are teaching switches.
          The real conditions for "a secure tunnel is possible" are not documented, so this is a visible stub.
        </p>
      </fieldset>

      <fieldset className="mt-3 min-w-0">
        <legend className="text-sm font-medium">Traffic between Edges (for dynamic tunnels)</legend>
        <div className="mt-1 flex flex-wrap items-center gap-2 text-sm">
          <select
            className={field}
            value={trafficA}
            onChange={(e) => setTrafficA(e.target.value)}
            aria-label="Traffic from"
            data-cond-traffic-a
          >
            {edges.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </select>
          <span aria-hidden>→</span>
          <select
            className={field}
            value={trafficB}
            onChange={(e) => setTrafficB(e.target.value)}
            aria-label="Traffic to"
            data-cond-traffic-b
          >
            {edges.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </select>
          <label className="flex items-center gap-1">
            last packet
            <input
              className={`${field} w-20`}
              inputMode="numeric"
              value={silent}
              onChange={(e) => setSilent(e.target.value)}
              aria-label="Seconds since the last packet"
              data-cond-traffic-silent
            />
            s ago
          </label>
          <button
            type="button"
            disabled={!trafficA || !trafficB || trafficA === trafficB}
            onClick={() => {
              const last = Math.max(0, Number(silent) || 0)
              const spec = { a: trafficA, b: trafficB, first_packet_ago_s: last + 60, last_packet_ago_s: last }
              set({
                ...conditions,
                traffic: [...(conditions.traffic ?? []).filter((x) => !(x.a === trafficA && x.b === trafficB)), spec],
              })
            }}
            className="rounded-md border border-slate-300 px-2 py-1 text-xs hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-50 dark:border-slate-700"
            data-cond-traffic-add
          >
            Add
          </button>
        </div>
        {conditions.traffic && conditions.traffic.length > 0 && (
          <ul className="mt-2 space-y-1 text-xs" data-cond-traffic-list>
            {conditions.traffic.map((x) => (
              <li key={`${x.a}-${x.b}`} className="flex items-center justify-between gap-2">
                <span>
                  {x.a} → {x.b}: last packet {x.last_packet_ago_s} s ago
                </span>
                <button
                  type="button"
                  className="underline"
                  onClick={() => set({ ...conditions, traffic: (conditions.traffic ?? []).filter((y) => y !== x) })}
                >
                  remove
                </button>
              </li>
            ))}
          </ul>
        )}
      </fieldset>
    </section>
  )
}
