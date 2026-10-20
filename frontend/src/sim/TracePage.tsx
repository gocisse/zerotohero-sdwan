import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { postTrace, type FlowSpec, type TraceResult, type TraceStage } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import ConditionsPanel from './ConditionsPanel'
import { LessonLink } from './LessonLink'
import { withLab } from './labParams'
import { defaultQuery, originatedPrefixes } from './routingQuery'
import { AssumptionsPanel, RoutingConditions } from './RoutingControls'
import { sourceText, switchValueText } from './routingText'
import { useTopologyCtx } from './topologyContext'
import { TRACE_STATUS_MARK, TRACE_STATUS_STYLE, TRACE_STATUS_TEXT, VERDICT_TEXT } from './traceText'
import { Bases, NotModelledNote } from './ui'

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

function Evidence({ s }: { s: TraceStage }) {
  return (
    <>
      {s.rules && s.rules.length > 0 && (
        <ul className="mt-2 space-y-1 text-xs" data-evidence="rules">
          {s.rules.map((r) => (
            <li
              key={r.rule_id}
              data-rule={r.rule_id}
              data-chosen={r.chosen ? 'true' : 'false'}
              className={r.chosen ? 'font-medium' : ''}
            >
              <span aria-hidden>{r.chosen ? '▶ ' : '· '}</span>
              {r.rule_id} ({r.owner}): {r.chosen ? 'MATCHED' : r.matched ? 'matches' : 'does not match'} - {r.reason}
            </li>
          ))}
        </ul>
      )}
      {s.firewall && s.firewall.length > 0 && (
        <ul className="mt-2 space-y-1 text-xs" data-evidence="firewall">
          {s.firewall.map((r) => (
            <li
              key={r.rule_id}
              data-fw-rule={r.rule_id}
              data-decisive={r.decisive ? 'true' : 'false'}
              className={r.decisive ? 'font-medium' : ''}
            >
              <span aria-hidden>{r.decisive ? '▶ ' : '· '}</span>
              {r.rule_id} ({r.group} group, {r.owner}, action {r.action}):{' '}
              {r.decisive ? 'DECIDES' : r.matched ? 'matches' : 'does not match'} - {r.reason}
            </li>
          ))}
        </ul>
      )}
      {s.candidates && s.candidates.length > 0 && (
        <ul className="mt-2 space-y-1 text-xs" data-evidence="routes">
          {s.candidates.map((c) => (
            <li key={c.id} data-selected={c.selected ? 'true' : 'false'}>
              #{c.rank} <code>{c.prefix}</code> {sourceText(c)}
              {c.origin_name ? ` from ${c.origin_name}` : ''}: {c.reason}
            </li>
          ))}
        </ul>
      )}
      {s.path && s.path.options.length > 0 && (
        <ul className="mt-2 space-y-1 text-xs" data-evidence="options">
          {s.path.options.map((o, i) => (
            <li
              key={i}
              data-option={o.kind}
              data-chosen={o.chosen ? 'true' : 'false'}
              data-usable={o.usable ? 'true' : 'false'}
            >
              <span className="font-medium">{o.label}</span>:{' '}
              {o.chosen ? 'CHOSEN' : o.usable ? 'usable, not chosen' : 'rejected'} - {o.reason}{' '}
              <Bases basis={o.basis} also={o.also} />
            </li>
          ))}
        </ul>
      )}
      {s.remediation && (
        <ul className="mt-2 space-y-1 text-xs" data-evidence="remediation">
          {s.remediation.map((m) => (
            <li key={m.name} data-mech={m.name} data-candidate={m.candidate ? 'true' : 'false'}>
              <span className="font-medium">{m.name}</span> (
              {m.candidate ? 'a documented candidate for this flow' : 'not a candidate for this flow'}): {m.conditions}{' '}
              <Bases basis={m.basis} also={m.also} />
            </li>
          ))}
        </ul>
      )}
    </>
  )
}

function StageCard({
  s,
  blocked,
  topologyId,
  labSearch,
}: {
  s: TraceStage
  blocked: boolean
  topologyId: string
  labSearch: string
}) {
  return (
    <li
      data-trace-stage={s.id}
      data-status={s.status}
      data-blocked-here={blocked ? 'true' : 'false'}
      className={`rounded-md border p-3 text-sm ${TRACE_STATUS_STYLE[s.status]} ${blocked ? 'ring-2 ring-red-500' : ''}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span aria-hidden className="font-mono font-bold">
          {TRACE_STATUS_MARK[s.status]}
        </span>
        <span className="font-medium">
          {s.number}. {s.title}
        </span>
        <span className="rounded-full border border-slate-400 px-2 py-0.5 text-xs">{TRACE_STATUS_TEXT[s.status]}</span>
        {blocked && (
          <span className="rounded-full border border-red-500 px-2 py-0.5 text-xs font-semibold text-red-900 dark:text-red-100">
            the trace ends here
          </span>
        )}
      </div>
      {s.input.length > 0 && (
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400" data-trace-input>
          Input: {s.input.join(' · ')}
        </p>
      )}
      <p className="mt-1" data-trace-decision>
        {s.decision}
      </p>
      {s.reason && (
        <p className="mt-1 text-xs text-slate-700 dark:text-slate-200" data-trace-reason>
          {s.reason}
        </p>
      )}
      {s.details && s.details.length > 0 && !s.rules?.length && !s.firewall?.length && !s.path && (
        <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-slate-600 dark:text-slate-300">
          {s.details.map((d, i) => (
            <li key={i}>{d}</li>
          ))}
        </ul>
      )}
      <Evidence s={s} />
      {s.mtu && (
        <p className="mt-2 text-xs" data-evidence="mtu">
          Path MTU {s.mtu.path_mtu} ({s.mtu.limiting}); overhead {s.mtu.overhead_min} to {s.mtu.overhead_max} bytes;
          effective packet MTU {s.mtu.effective_min} to {s.mtu.effective_max}.
        </p>
      )}
      {s.mtu && s.details && s.details.length > 0 && (
        <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-slate-600 dark:text-slate-300">
          {s.details.map((d, i) => (
            <li key={i}>{d}</li>
          ))}
        </ul>
      )}
      {s.not_modelled && s.not_modelled.length > 0 && (
        <div className="mt-2 grid gap-2 md:grid-cols-2" data-stage-not-modelled>
          {s.not_modelled.map((n) => (
            <NotModelledNote key={n.rule} nm={n} />
          ))}
        </div>
      )}
      {s.heuristic && (
        <p
          className="mt-2 rounded border border-amber-600 bg-amber-50 p-2 text-xs text-amber-950 dark:border-amber-400 dark:bg-amber-950 dark:text-amber-100"
          data-heuristic
        >
          <strong>Heuristic, may be wrong.</strong> {s.heuristic.replace(/^Heuristic, may be wrong: /, '')}
        </p>
      )}
      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
        <Bases basis={s.basis} also={s.also} />
        <Link to={withLab(`/sim/${topologyId}/${s.screen}`, labSearch)} className="underline" data-stage-screen>
          Inspect the screen
        </Link>
        <LessonLink id={s.lesson_ref} />
      </div>
    </li>
  )
}

/** The flow tracer: follow ONE flow end to end and see, at every stage, what was decided and why. */
export default function TracePage() {
  const { detail, topologyId, conditions, assumptions, labSearch } = useTopologyCtx()
  const t = detail.topology
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])
  const origins = useMemo(() => originatedPrefixes(t), [t])
  const apps = useMemo(() => {
    const set = new Set<string>()
    t.profiles.forEach((p) => p.business_rules?.forEach((r) => r.match.application && set.add(r.match.application)))
    edges.forEach((e) =>
      e.overrides.business_rules?.forEach((r) => r.match.application && set.add(r.match.application)),
    )
    return [...set].sort()
  }, [t, edges])
  const init = useMemo<FlowSpec>(() => {
    if (detail.lab?.flow) return detail.lab.flow
    const hasRouting = edges.some((e) => e.routing)
    if (hasRouting) {
      const q = defaultQuery(t)
      return { source: q.source, destination: q.destination }
    }
    // Two Edges whose Profiles enable Cloud VPN, so the default flow has an overlay path to follow.
    const vpn = edges.filter((e) => t.profiles.find((p) => p.id === e.profile)?.cloud_vpn?.segments?.length)
    const [a, b] = vpn.length >= 2 ? [vpn[0]!, vpn[1]!] : [edges[0], edges[1]]
    return { source: a?.id ?? '', destination: b ? `edge:${b.id}` : '' }
  }, [t, edges, detail.lab])
  const [form, setForm] = useState<FlowSpec>(init)
  const [asked, setAsked] = useState<FlowSpec>(init)
  const [state, setState] = useState<{ result: TraceResult | null; error: Error | null; busy: boolean }>({
    result: null,
    error: null,
    busy: true,
  })
  const key = JSON.stringify([topologyId, asked, conditions, assumptions, labSearch])

  useEffect(() => {
    let cancelled = false
    setState((s) => ({ ...s, busy: true }))
    postTrace(topologyId, asked, conditions, assumptions)
      .then((result) => !cancelled && setState({ result, error: null, busy: false }))
      .catch((error: Error) => !cancelled && setState((s) => ({ ...s, error, busy: false })))
    return () => {
      cancelled = true
    }
    // `key` stands for the five inputs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])

  const r = state.result
  const set = (patch: Partial<FlowSpec>) => setForm((f) => ({ ...f, ...patch }))
  const num = (v: string): number | undefined => (v.trim() === '' ? undefined : Number(v))
  return (
    <div data-trace-page>
      <section className="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
        <h2 className="text-base font-semibold">Follow one flow end to end</h2>
        <p className="mt-1 text-sm text-slate-700 dark:text-slate-200">
          Segment, business policy, firewall, route, path, remediation, overhead and return path, in the simulator's
          presentation order (the real order of the policy, firewall and routing decisions is not documented).
        </p>
        <form
          className="mt-3 grid gap-3 sm:grid-cols-4"
          onSubmit={(e) => {
            e.preventDefault()
            setAsked({ ...form, destination: form.destination.trim() })
          }}
        >
          <label className="text-sm">
            <span className="font-medium">Source Edge</span>
            <select
              className={field}
              value={form.source}
              onChange={(e) => set({ source: e.target.value })}
              data-trace-source
            >
              {edges.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">LAN host IP (optional)</span>
            <input
              className={field}
              value={form.source_ip ?? ''}
              onChange={(e) => set({ source_ip: e.target.value || undefined })}
              data-trace-source-ip
              placeholder="10.20.255.2"
            />
          </label>
          <label className="text-sm sm:col-span-2">
            <span className="font-medium">Destination: IP, prefix, edge:&lt;id&gt; or nsd:&lt;id&gt;</span>
            <input
              className={field}
              list="trace-dests"
              value={form.destination}
              onChange={(e) => set({ destination: e.target.value })}
              data-trace-dest
            />
            <datalist id="trace-dests">
              {origins.map((o, i) => (
                <option key={`o${i}`} value={o.prefix}>
                  {o.label}
                </option>
              ))}
              {edges.map((e) => (
                <option key={e.id} value={`edge:${e.id}`}>
                  {e.name}
                </option>
              ))}
              {(t.non_sdwan_destinations ?? []).map((n) => (
                <option key={n.id} value={`nsd:${n.id}`}>
                  Non SD-WAN destination {n.id}
                </option>
              ))}
            </datalist>
          </label>
          <label className="text-sm">
            <span className="font-medium">Segment</span>
            <select
              className={field}
              value={form.segment ?? 'global'}
              onChange={(e) => set({ segment: e.target.value })}
              data-trace-segment
            >
              {t.segments.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Application</span>
            <input
              className={field}
              list="trace-apps"
              value={form.application ?? ''}
              onChange={(e) => set({ application: e.target.value || undefined })}
              data-trace-app
            />
            <datalist id="trace-apps">
              {apps.map((a) => (
                <option key={a} value={a} />
              ))}
            </datalist>
          </label>
          <label className="text-sm">
            <span className="font-medium">Protocol</span>
            <select
              className={field}
              value={form.protocol ?? ''}
              onChange={(e) => set({ protocol: e.target.value || undefined })}
              data-trace-protocol
            >
              <option value="">any</option>
              <option value="tcp">tcp</option>
              <option value="udp">udp</option>
              <option value="icmp">icmp</option>
              <option value="gre">gre</option>
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Port</span>
            <input
              className={field}
              inputMode="numeric"
              value={form.port ?? ''}
              onChange={(e) => set({ port: num(e.target.value) })}
              data-trace-port
            />
          </label>
          <label className="text-sm">
            <span className="font-medium">DSCP (optional)</span>
            <input
              className={field}
              value={form.dscp ?? ''}
              onChange={(e) => set({ dscp: e.target.value || undefined })}
              data-trace-dscp
            />
          </label>
          <label className="text-sm">
            <span className="font-medium">Traffic class</span>
            <select
              className={field}
              value={form.traffic_class ?? ''}
              onChange={(e) => set({ traffic_class: e.target.value || undefined })}
              data-trace-class
            >
              <option value="">not stated</option>
              <option value="real_time">real time</option>
              <option value="transactional">transactional</option>
              <option value="bulk">bulk</option>
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Packet size, bytes</span>
            <input
              className={field}
              inputMode="numeric"
              value={form.packet_size ?? ''}
              onChange={(e) => set({ packet_size: num(e.target.value) })}
              data-trace-size
            />
          </label>
          <div className="sm:col-span-4">
            <button
              type="submit"
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
              data-trace-run
            >
              Trace
            </button>
          </div>
        </form>
      </section>

      {state.error && (
        <div className="mt-4" data-trace-error>
          <ErrorPanel title="The flow could not be traced" error={state.error} />
        </div>
      )}
      {r && (
        <div
          className="mt-5"
          aria-live="polite"
          data-trace-result
          data-verdict={r.verdict}
          data-blocked-stage={r.blocked_stage}
        >
          <div
            className={`rounded-lg border p-4 text-sm ${
              r.verdict === 'delivered'
                ? 'border-emerald-400 bg-emerald-50/70 dark:border-emerald-700 dark:bg-emerald-950/30'
                : r.verdict === 'not-modelled'
                  ? 'border-dashed border-rose-300 bg-rose-50/60 dark:border-rose-800 dark:bg-rose-950/20'
                  : 'border-red-400 bg-red-50/70 dark:border-red-700 dark:bg-red-950/30'
            }`}
          >
            <p className="font-semibold" data-trace-verdict>
              {VERDICT_TEXT[r.verdict]}
              {r.blocked_at > 0 ? ` at stage ${r.blocked_at}` : ''}
              {state.busy ? ' (updating…)' : ''}
            </p>
            <p className="mt-1" data-trace-summary>
              {r.summary}
            </p>
          </div>
          {r.depends_on.length > 0 && (
            <div
              role="note"
              data-trace-depends
              className="mt-3 rounded-lg border border-orange-300 bg-orange-50/70 p-3 text-sm dark:border-orange-800 dark:bg-orange-950/30"
            >
              <p className="font-semibold">This verdict depends on unverified assumptions.</p>
              <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs">
                {r.depends_on.map((d) => (
                  <li key={d.switch.key} data-depends={d.switch.key}>
                    {d.switch.label} is {switchValueText(d.switch)} (assumption {d.switch.assumption}). Set to{' '}
                    {String(d.flipped_value).replace(/_/g, ' ')}, the verdict becomes {d.new_outcome}
                    {d.new_first_failing ? ` at the ${d.new_first_failing} stage` : ''}.
                  </li>
                ))}
              </ul>
            </div>
          )}
          <ol className="mt-4 space-y-2" data-trace-stages>
            {r.stages.map((s) => (
              <StageCard
                key={s.id}
                s={s}
                blocked={s.number === r.blocked_at}
                topologyId={topologyId}
                labSearch={labSearch}
              />
            ))}
          </ol>
          {r.not_modelled.length > 0 && (
            <section className="mt-5" aria-labelledby="tr-nm">
              <h3 id="tr-nm" className="text-base font-semibold">
                Not modelled in this trace
              </h3>
              <div className="mt-2 grid gap-2 md:grid-cols-2" data-trace-not-modelled>
                {r.not_modelled.map((n) => (
                  <NotModelledNote key={n.rule} nm={n} />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
      <div className="mt-5 grid gap-5 lg:grid-cols-2">
        <div className="space-y-5">
          <ConditionsPanel wide />
          <RoutingConditions links={false} />
        </div>
        <AssumptionsPanel switches={r?.assumptions ?? []} />
      </div>
    </div>
  )
}
