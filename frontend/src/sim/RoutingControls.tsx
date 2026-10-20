import type { AssumptionSwitch, Conditions, RoutingAssumptions } from '../api'
import { isEmptyConditions, linkSetting, withLink, type LinkSetting } from './conditions'
import { switchValueText, withNeighbor, withProbe } from './routingText'
import { useTopologyCtx } from './topologyContext'
import { BasisBadge } from './ui'

const field =
  'max-w-full rounded-md border border-slate-300 bg-white px-2 py-1 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

/**
 * Conditions for the routing screens: WAN links, LAN neighbors and static-route probes. A what-if tool: it is sent with
 * each request, nothing is stored and the topology never changes.
 */
export function RoutingConditions() {
  const { detail, conditions, setConditions } = useTopologyCtx()
  const t = detail.topology
  const edges = t.sites.flatMap((s) => s.edges)
  const neighbors: { id: string; label: string }[] = []
  const probes: { id: string; label: string }[] = []
  for (const e of edges) {
    for (const i of e.routing?.ospf?.interfaces ?? [])
      neighbors.push({ id: `${e.id}/${i.id}`, label: `${e.name}: OSPF neighbor on ${i.id} (${i.mode})` })
    for (const n of e.routing?.bgp?.neighbors ?? [])
      neighbors.push({ id: `${e.id}/${n.id}`, label: `${e.name}: BGP neighbor ${n.ip} (AS ${n.asn})` })
    for (const r of e.routing?.static_routes ?? [])
      if (r.icmp_probe) probes.push({ id: `${e.id}/${r.id}`, label: `${e.name}: probe of static route ${r.subnet}` })
  }
  const set = (c: Conditions) => setConditions(c)
  return (
    <section
      aria-labelledby="rc-h"
      className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
      data-routing-conditions
    >
      <div className="flex items-center justify-between gap-2">
        <h2 id="rc-h" className="text-base font-semibold">
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
        What-if only: nothing is stored and the topology never changes.
      </p>
      <fieldset className="mt-3 min-w-0">
        <legend className="text-sm font-medium">WAN links</legend>
        <div className="mt-1 grid grid-cols-1 gap-x-6 gap-y-1 sm:grid-cols-2">
          {t.links.map((l) => (
            <label key={l.id} className="flex items-center justify-between gap-2 text-sm">
              <span className="min-w-0 flex-1">{l.id}</span>
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
      </fieldset>
      {neighbors.length > 0 && (
        <fieldset className="mt-3 min-w-0">
          <legend className="text-sm font-medium">LAN neighbors down</legend>
          <div className="mt-1 space-y-1">
            {neighbors.map((n) => (
              <label key={n.id} className="flex items-start gap-1.5 text-sm">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={!!conditions.neighbors_down?.includes(n.id)}
                  onChange={(e) => set(withNeighbor(conditions, n.id, e.target.checked))}
                  data-cond-neighbor={n.id}
                />
                <span className="min-w-0">{n.label}</span>
              </label>
            ))}
          </div>
        </fieldset>
      )}
      {probes.length > 0 && (
        <fieldset className="mt-3 min-w-0">
          <legend className="text-sm font-medium">ICMP probes failing</legend>
          <div className="mt-1 space-y-1">
            {probes.map((n) => (
              <label key={n.id} className="flex items-start gap-1.5 text-sm">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={!!conditions.probes_down?.includes(n.id)}
                  onChange={(e) => set(withProbe(conditions, n.id, e.target.checked))}
                  data-cond-probe={n.id}
                />
                <span className="min-w-0">{n.label}</span>
              </label>
            ))}
          </div>
        </fieldset>
      )}
    </section>
  )
}

/**
 * Every unverified routing default the results depend on, as switches. Changing one recomputes every result. A what-if
 * tool: nothing is stored, and these values are NOT documented.
 */
export function AssumptionsPanel({ switches }: { switches: AssumptionSwitch[] }) {
  const { assumptions, setAssumptions } = useTopologyCtx()
  const set = (key: keyof RoutingAssumptions, value: boolean | string) =>
    setAssumptions({ ...assumptions, [key]: value })
  const anyChanged = switches.some((s) => s.changed)
  return (
    <section
      aria-labelledby="as-h"
      className="rounded-lg border border-orange-300 p-4 dark:border-orange-800"
      data-assumptions-panel
    >
      <div className="flex items-center justify-between gap-2">
        <h2 id="as-h" className="text-base font-semibold">
          Assumptions in effect
        </h2>
        <button
          type="button"
          onClick={() => setAssumptions({})}
          disabled={!anyChanged}
          className="rounded-md border border-slate-300 px-2 py-1 text-xs hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-50 dark:border-slate-700"
          data-assumptions-reset
        >
          Back to the defaults
        </button>
      </div>
      <p className="mt-1 text-xs text-slate-700 dark:text-slate-200">
        These values are <strong>not verified</strong>: the Arista documentation read does not state them. They are
        switches so you can see how much a result depends on them. A what-if tool: nothing is stored, and changing a
        switch recomputes every result on this page. A value written in the topology file wins over its switch.
      </p>
      <ul className="mt-3 space-y-2">
        {switches.map((s) => (
          <li
            key={s.key}
            className="text-sm"
            data-assumption={s.key}
            data-assumption-changed={s.changed ? 'true' : 'false'}
          >
            <div className="flex flex-wrap items-center gap-2">
              {s.kind === 'bool' ? (
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={s.value === true}
                    onChange={(e) => set(s.key, e.target.checked)}
                    data-assumption-input={s.key}
                  />
                  <span>{s.label}</span>
                </label>
              ) : (
                <label className="flex flex-wrap items-center gap-2">
                  <span>{s.label}</span>
                  <select
                    className={field}
                    value={String(s.value)}
                    onChange={(e) => set(s.key, e.target.value)}
                    data-assumption-input={s.key}
                  >
                    {(s.choices ?? []).map((c) => (
                      <option key={c} value={c}>
                        {c.replace(/_/g, ' ')}
                      </option>
                    ))}
                  </select>
                </label>
              )}
              <BasisBadge basis={s.basis} />
              {s.used && (
                <span
                  className="rounded-full border border-orange-400 px-2 py-0.5 text-xs text-orange-900 dark:text-orange-100"
                  data-assumption-used
                >
                  used by this result
                </span>
              )}
              {s.changed && (
                <span className="text-xs text-slate-500 dark:text-slate-400">
                  (default: {String(s.default).replace(/_/g, ' ')})
                </span>
              )}
            </div>
            <p className="ml-6 text-xs text-slate-600 dark:text-slate-300">
              Now {switchValueText(s)}. {s.text}
            </p>
          </li>
        ))}
      </ul>
    </section>
  )
}
