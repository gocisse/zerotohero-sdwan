import { useMemo, useState } from 'react'
import { postPath, type Flow, type PathDecision, type PathOption } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import ConditionsPanel from './ConditionsPanel'
import { StateBadge } from './overlayUi'
import { stateLabel } from './overlayText'
import { useTopologyCtx } from './topologyContext'
import { Bases, NotModelledNote } from './ui'

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

function OptionCard({ o, index }: { o: PathOption; index: number }) {
  return (
    <li
      data-option={o.kind}
      data-chosen={o.chosen ? 'true' : 'false'}
      data-usable={o.usable ? 'true' : 'false'}
      className={`rounded-md border p-3 text-sm ${o.chosen ? 'border-emerald-500 bg-emerald-50/60 dark:border-emerald-700 dark:bg-emerald-950/20' : 'border-slate-200 dark:border-slate-800'}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-slate-500 dark:text-slate-400">{index + 1}.</span>
        <span className="font-medium">{o.label}</span>
        {o.chosen ? (
          <span className="rounded-full border border-emerald-500 px-2 py-0.5 text-xs font-semibold text-emerald-900 dark:text-emerald-100">
            chosen
          </span>
        ) : o.usable ? (
          <span className="rounded-full border border-slate-400 px-2 py-0.5 text-xs">usable, not chosen</span>
        ) : (
          <span className="rounded-full border border-red-400 px-2 py-0.5 text-xs text-red-900 dark:text-red-100">
            rejected
          </span>
        )}
      </div>
      <p className="mt-1 text-xs text-slate-700 dark:text-slate-200" data-option-reason>
        {o.reason}
      </p>
      <div className="mt-2 flex flex-wrap items-center gap-1">
        <Bases basis={o.basis} also={o.also} />
      </div>
    </li>
  )
}

/** The path explainer: pick a source, a destination and an application; see the chosen path, the rejected options and why. */
export default function PathExplainerPage() {
  const { detail, topologyId, conditions } = useTopologyCtx()
  const t = detail.topology
  const edges = t.sites.flatMap((s) => s.edges)
  const nsds = t.non_sdwan_destinations ?? []
  const apps = useMemo(() => {
    const set = new Set<string>()
    t.profiles.forEach((p) => p.business_rules?.forEach((r) => r.match.application && set.add(r.match.application)))
    t.sites.forEach((s) =>
      s.edges.forEach((e) =>
        e.overrides.business_rules?.forEach((r) => r.match.application && set.add(r.match.application)),
      ),
    )
    return [...set].sort()
  }, [t])

  const [source, setSource] = useState(edges[0]?.id ?? '')
  const [destKey, setDestKey] = useState(edges[1] ? `edge:${edges[1].id}` : '')
  const [app, setApp] = useState('')
  const [cls, setCls] = useState('')
  const [result, setResult] = useState<PathDecision | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const [busy, setBusy] = useState(false)

  const run = async () => {
    const [kind, id] = destKey.split(':') as ['edge' | 'nsd', string]
    const flow: Flow = {
      source,
      destination: { kind, id },
      application: app || undefined,
      traffic_class: cls || undefined,
    }
    setBusy(true)
    setError(null)
    try {
      setResult(await postPath(topologyId, flow, conditions))
    } catch (e) {
      setResult(null)
      setError(e as Error)
    } finally {
      setBusy(false)
    }
  }

  const s = result?.steering
  return (
    <div>
      <p className="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
        When one Edge talks to another (or to a Non SD-WAN destination), which path is used, which options were rejected
        and why, and what link steering does. The conditions below apply.
      </p>
      <div className="mt-4 grid gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="min-w-0">
          <form
            className="grid gap-4 sm:grid-cols-2"
            onSubmit={(e) => {
              e.preventDefault()
              void run()
            }}
            data-path-form
          >
            <label className="block text-sm">
              <span className="font-medium">Source Edge</span>
              <select className={field} value={source} onChange={(e) => setSource(e.target.value)} data-path-source>
                {edges.map((e) => (
                  <option key={e.id} value={e.id}>
                    {e.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Destination</span>
              <select className={field} value={destKey} onChange={(e) => setDestKey(e.target.value)} data-path-dest>
                {edges
                  .filter((e) => e.id !== source)
                  .map((e) => (
                    <option key={e.id} value={`edge:${e.id}`}>
                      {e.name}
                    </option>
                  ))}
                {nsds.map((n) => (
                  <option key={n.id} value={`nsd:${n.id}`}>
                    {n.name ?? n.id} (Non SD-WAN, {n.via})
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Application</span>
              <input
                className={field}
                list="apps"
                value={app}
                onChange={(e) => setApp(e.target.value)}
                placeholder="for example voice"
                data-path-app
              />
              <datalist id="apps">
                {apps.map((a) => (
                  <option key={a} value={a} />
                ))}
              </datalist>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Traffic class</span>
              <select className={field} value={cls} onChange={(e) => setCls(e.target.value)} data-path-class>
                <option value="">not set</option>
                <option value="real_time">Real-Time</option>
                <option value="transactional">Transactional</option>
                <option value="bulk">Bulk</option>
              </select>
              <span className="mt-1 block text-xs text-slate-500 dark:text-slate-400">
                Informational: it does not change the path here.
              </span>
            </label>
            <div className="sm:col-span-2">
              <button
                type="submit"
                disabled={busy || !destKey}
                className="rounded-md bg-sky-700 px-4 py-2 text-sm font-semibold text-white hover:bg-sky-800 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none disabled:opacity-50"
                data-path-run
              >
                {busy ? 'Working…' : 'Explain the path'}
              </button>
            </div>
          </form>

          <div aria-live="polite" className="mt-6">
            {error && <ErrorPanel title="That flow cannot be decided" error={error} />}
            {result && (
              <div data-path-result className="space-y-5">
                <div
                  role="note"
                  data-reachability-assumed
                  className="rounded-md border border-orange-300 bg-orange-50 px-3 py-2 text-sm text-orange-950 dark:border-orange-700 dark:bg-orange-950/40 dark:text-orange-100"
                >
                  {result.reachability.text}
                </div>
                <div>
                  <h2 className="text-lg font-semibold" data-path-why>
                    {result.chosen ? `Chosen: ${result.chosen.label}` : 'No path'}
                  </h2>
                  <p className="mt-1 text-sm text-slate-700 dark:text-slate-200">{result.why}</p>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Network service {result.network_service.replace('_', ' ')}
                    {result.rule_id
                      ? ` · rule ${result.rule_id} (${result.rule_owner})`
                      : ' · no business rule matched'}
                  </p>
                </div>

                {result.options.length > 0 && (
                  <section aria-labelledby="opts-h">
                    <h3 id="opts-h" className="text-base font-semibold">
                      Options, in the order they were tried
                    </h3>
                    <ol className="mt-2 space-y-2" data-option-list>
                      {result.options.map((o, i) => (
                        <OptionCard key={`${o.kind}-${o.via ?? ''}-${i}`} o={o} index={i} />
                      ))}
                    </ol>
                  </section>
                )}

                {s && (
                  <section aria-labelledby="steer-h" data-steering>
                    <h3 id="steer-h" className="text-base font-semibold">
                      Link steering on {result.flow.source}'s first leg
                    </h3>
                    <div className="mt-2 rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800">
                      <p>
                        Mode <b data-steer-mode>{s.mode.replace('_', ' ')}</b>
                        {s.strictness ? (
                          <>
                            , strictness <b>{s.strictness}</b>
                          </>
                        ) : null}
                        . Action <b data-steer-action>{s.action}</b>
                        {s.chosen ? (
                          <>
                            : link <b>{s.chosen.local_link}</b> <StateBadge state={s.chosen.state} />
                          </>
                        ) : null}
                      </p>
                      <p className="mt-1 text-xs text-slate-700 dark:text-slate-200">{s.reason}</p>
                      <div className="mt-2 flex flex-wrap items-center gap-1">
                        <Bases basis={s.basis} also={s.also} />
                      </div>
                    </div>
                  </section>
                )}

                {result.mtu && (
                  <section aria-labelledby="mtu-h" data-mtu>
                    <h3 id="mtu-h" className="text-base font-semibold">
                      Effective MTU for VPN (IPsec) traffic
                    </h3>
                    <div className="mt-2 rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800">
                      <p>
                        Path MTU {result.mtu.path_mtu} (limited by {result.mtu.limiting}) minus{' '}
                        {result.mtu.overhead_min} to {result.mtu.overhead_max} bytes of overhead:{' '}
                        <b data-mtu-range>
                          {result.mtu.effective_min} to {result.mtu.effective_max} bytes
                        </b>
                        .
                      </p>
                      <div className="mt-2 flex flex-wrap items-center gap-1">
                        <Bases basis={result.mtu.basis} also={result.mtu.also} />
                      </div>
                    </div>
                  </section>
                )}

                {result.notes.length > 0 && (
                  <ul className="space-y-1 text-xs text-slate-700 dark:text-slate-200">
                    {result.notes.map((n, i) => (
                      <li key={i} className="flex flex-wrap items-start gap-2">
                        <Bases basis={n.basis} />
                        <span className="min-w-0 flex-1">{n.text}</span>
                      </li>
                    ))}
                  </ul>
                )}
                {result.not_modelled.length > 0 && (
                  <div className="space-y-2">
                    {result.not_modelled.map((n) => (
                      <NotModelledNote key={n.rule} nm={n} />
                    ))}
                  </div>
                )}
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Tunnel states used:{' '}
                  {['up', 'degraded', 'down']
                    .map((st) => `${result.tunnels.filter((x) => x.state === st).length} ${stateLabel[st as 'up']}`)
                    .join(', ')}
                  .
                </p>
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
