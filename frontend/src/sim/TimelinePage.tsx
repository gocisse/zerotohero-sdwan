import { useEffect, useMemo, useState } from 'react'
import { postTimeline, type TimelineEvent, type TimelineResult, type Tunnel, type TunnelType } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { StateBadge } from './overlayUi'
import { stateLabel, typeLabel } from './overlayText'
import TunnelMap, { MapLegend } from './TunnelMap'
import { useTopologyCtx } from './topologyContext'
import { useReducedMotion } from './useReducedMotion'
import { BasisBadge, Bases, NotModelledNote } from './ui'

type Scenario = 'dynamic' | 'failover' | 'link'

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

const num = (s: string, d: number) => (Number.isFinite(Number(s)) && s.trim() !== '' ? Math.max(0, Number(s)) : d)

// The map under the player shows Hub and destination tunnels and only the dynamic tunnels that change: Gateway tunnels
// and the many unused dynamic pairs would bury the story.
const MAP_TYPES = new Set<TunnelType>(['hub', 'dynamic', 'nsd-gateway', 'nsd-edge'])

/** The timeline player: tunnel states over a sequence of timed events (dynamic tunnel lifecycle, Hub failover, link failure). */
export default function TimelinePage() {
  const { detail, topologyId, conditions } = useTopologyCtx()
  const t = detail.topology
  const reduced = useReducedMotion()
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])

  // Defaults: a pair of Edges whose Profile has Branch to Branch on; a Spoke with two Hubs; the first link.
  const pairDefault = useMemo(() => {
    for (const p of t.profiles) {
      if (!p.cloud_vpn?.branch_to_branch?.enabled) continue
      const e = edges.filter((x) => x.profile === p.id)
      if (e.length >= 2) return [e[0]!.id, e[1]!.id] as const
    }
    return [edges[0]?.id ?? '', edges[1]?.id ?? ''] as const
  }, [t, edges])
  const spokes = useMemo(
    () =>
      edges.filter(
        (e) => (t.profiles.find((p) => p.id === e.profile)?.cloud_vpn?.branch_to_hub?.hubs?.length ?? 0) >= 2,
      ),
    [t, edges],
  )
  const available: Scenario[] = ['dynamic', ...(spokes.length > 0 ? (['failover'] as const) : []), 'link']

  const [scenario, setScenario] = useState<Scenario>('dynamic')
  const [a, setA] = useState<string>(pairDefault[0])
  const [b, setB] = useState<string>(pairDefault[1])
  const [startAt, setStartAt] = useState('10')
  const [stopAt, setStopAt] = useState('40')
  const [spoke, setSpoke] = useState(spokes[0]?.id ?? '')
  const [downAt, setDownAt] = useState('5')
  const [upAt, setUpAt] = useState('20')
  const [flushAt, setFlushAt] = useState('30')
  const [link, setLink] = useState(t.links[0]?.id ?? '')
  const [linkDown, setLinkDown] = useState('5')
  const [linkUp, setLinkUp] = useState('25')

  const primaryHub = useMemo(() => {
    const e = edges.find((x) => x.id === spoke)
    const hubs = t.profiles.find((p) => p.id === e?.profile)?.cloud_vpn?.branch_to_hub?.hubs ?? []
    const first = hubs[0]
    if (!first) return ''
    const c = t.clusters?.find((x) => x.id === first)
    return c ? (c.members[(e?.logical_id ?? 0) % c.members.length] ?? '') : first
  }, [t, edges, spoke])

  const events: TimelineEvent[] = useMemo(() => {
    if (scenario === 'dynamic')
      return [
        { at_s: num(startAt, 10), type: 'traffic_start', a, b },
        { at_s: num(stopAt, 40), type: 'traffic_stop', a, b },
      ]
    if (scenario === 'failover')
      return [
        { at_s: num(downAt, 5), type: 'hub_down', target: primaryHub },
        { at_s: num(upAt, 20), type: 'hub_up', target: primaryHub },
        { at_s: num(flushAt, 30), type: 'flush_flows', target: spoke },
      ]
    return [
      { at_s: num(linkDown, 5), type: 'link_down', target: link },
      { at_s: num(linkUp, 25), type: 'link_up', target: link },
    ]
  }, [scenario, a, b, startAt, stopAt, downAt, upAt, flushAt, spoke, primaryHub, link, linkDown, linkUp])

  const key = JSON.stringify([topologyId, conditions, events])
  const [res, setRes] = useState<{ key: string; data: TimelineResult | null; error: Error | null }>({
    key: '',
    data: null,
    error: null,
  })
  useEffect(() => {
    let cancelled = false
    postTimeline(topologyId, conditions, events)
      .then((data) => !cancelled && setRes({ key, data, error: null }))
      .catch((error: Error) => !cancelled && setRes({ key, data: null, error }))
    return () => {
      cancelled = true
    }
    // The key encodes the topology, the conditions and the events.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])

  const data = res.key === key ? res.data : null
  const [step, setStep] = useState<{ key: string; i: number }>({ key: '', i: 0 })
  const [playing, setPlaying] = useState(false)
  const frames = useMemo(() => data?.frames ?? [], [data])
  const index = step.key === key ? Math.min(step.i, Math.max(frames.length - 1, 0)) : 0
  const go = (i: number) => setStep({ key, i: Math.max(0, Math.min(i, frames.length - 1)) })

  useEffect(() => {
    if (!playing) return
    if (index >= frames.length - 1) {
      const id = setTimeout(() => setPlaying(false), 0)
      return () => clearTimeout(id)
    }
    const id = setTimeout(() => setStep({ key, i: index + 1 }), 1400)
    return () => clearTimeout(id)
  }, [playing, index, frames.length, key])

  const frame = frames[index]
  const states = useMemo(() => new Map((frame?.states ?? []).map((s) => [s.id, s.state])), [frame])
  const tunnels: Tunnel[] = useMemo(
    () => (data?.tunnels ?? []).map((tn) => ({ ...tn, state: states.get(tn.id) ?? tn.state })),
    [data, states],
  )
  // The tunnels that change somewhere in the timeline are the interesting ones.
  const changing = useMemo(() => {
    const ids = new Set<string>()
    for (const f of frames) for (const c of f.changes) ids.add(c.id)
    return tunnels.filter((x) => ids.has(x.id))
  }, [frames, tunnels])
  const changingIds = useMemo(() => new Set(changing.map((x) => x.id)), [changing])
  const mapTunnels = useMemo(
    () => tunnels.filter((x) => x.type !== 'dynamic' || changingIds.has(x.id)),
    [tunnels, changingIds],
  )
  const maxT = Math.max(1, ...frames.map((f) => f.at_s)) * 1.05
  const unreachable = {
    edges: conditions.hubs_unreachable ?? [],
    gateways: conditions.gateways_unreachable ?? [],
    nsds: [] as string[],
  }
  const scenarioName: Record<Scenario, string> = {
    dynamic: 'Dynamic branch-to-branch tunnel',
    failover: 'Hub failover',
    link: 'Link failure',
  }

  return (
    <div>
      <p className="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
        Watch tunnels change over time. Pick a scenario, then step through or play it. Nothing is stored.
      </p>
      <div className="mt-4 grid max-w-3xl gap-4 sm:grid-cols-2" data-tl-form>
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium">Scenario</span>
          <select
            className={field}
            value={scenario}
            onChange={(e) => setScenario(e.target.value as Scenario)}
            data-tl-scenario
          >
            {available.map((s) => (
              <option key={s} value={s}>
                {scenarioName[s]}
              </option>
            ))}
          </select>
        </label>
        {scenario === 'dynamic' && (
          <>
            <label className="block text-sm">
              <span className="font-medium">Edge that starts sending</span>
              <select className={field} value={a} onChange={(e) => setA(e.target.value)} data-tl-a>
                {edges.map((e) => (
                  <option key={e.id} value={e.id}>
                    {e.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Peer</span>
              <select className={field} value={b} onChange={(e) => setB(e.target.value)} data-tl-b>
                {edges.map((e) => (
                  <option key={e.id} value={e.id}>
                    {e.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Traffic starts at (s)</span>
              <input
                className={field}
                inputMode="numeric"
                value={startAt}
                onChange={(e) => setStartAt(e.target.value)}
                data-tl-start
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">Traffic stops at (s)</span>
              <input
                className={field}
                inputMode="numeric"
                value={stopAt}
                onChange={(e) => setStopAt(e.target.value)}
                data-tl-stop
              />
            </label>
          </>
        )}
        {scenario === 'failover' && (
          <>
            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">Spoke</span>
              <select className={field} value={spoke} onChange={(e) => setSpoke(e.target.value)} data-tl-spoke>
                {spokes.map((e) => (
                  <option key={e.id} value={e.id}>
                    {e.name}
                  </option>
                ))}
              </select>
              <span className="mt-1 block text-xs text-slate-500 dark:text-slate-400">
                The primary Hub of this Spoke is {primaryHub || 'not set'}.
              </span>
            </label>
            {[
              ['Primary Hub goes down at (s)', downAt, setDownAt],
              ['Primary Hub returns at (s)', upAt, setUpAt],
              ['Affected flows are flushed at (s)', flushAt, setFlushAt],
            ].map(([label, value, set]) => (
              <label key={label as string} className="block text-sm">
                <span className="font-medium">{label as string}</span>
                <input
                  className={field}
                  inputMode="numeric"
                  value={value as string}
                  onChange={(e) => (set as (v: string) => void)(e.target.value)}
                />
              </label>
            ))}
          </>
        )}
        {scenario === 'link' && (
          <>
            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">WAN link</span>
              <select className={field} value={link} onChange={(e) => setLink(e.target.value)} data-tl-link>
                {t.links.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.id}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium">Goes down at (s)</span>
              <input
                className={field}
                inputMode="numeric"
                value={linkDown}
                onChange={(e) => setLinkDown(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">Comes back at (s)</span>
              <input className={field} inputMode="numeric" value={linkUp} onChange={(e) => setLinkUp(e.target.value)} />
            </label>
          </>
        )}
      </div>

      {res.key === key && res.error && (
        <div className="mt-4">
          <ErrorPanel title="That timeline cannot be built" error={res.error} />
        </div>
      )}
      {!data && !res.error && <p className="mt-4 text-slate-500">Loading…</p>}

      {data && frame && (
        <div className="mt-6 space-y-5" data-timeline>
          <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Timeline controls">
            <button type="button" onClick={() => go(index - 1)} disabled={index === 0} className={btn} data-tl-prev>
              ← Previous
            </button>
            <button
              type="button"
              onClick={() => go(index + 1)}
              disabled={index >= frames.length - 1}
              className={btn}
              data-tl-next
            >
              Next →
            </button>
            <button
              type="button"
              onClick={() => {
                if (!playing && index >= frames.length - 1) go(0)
                setPlaying((p) => !p)
              }}
              className={btn}
              data-tl-play
            >
              {playing ? '❚❚ Pause' : '▶ Play'}
            </button>
            <button
              type="button"
              onClick={() => {
                setPlaying(false)
                go(0)
              }}
              className={btn}
              data-tl-reset
            >
              ↺ Reset
            </button>
            <span className="ml-auto text-sm text-slate-600 dark:text-slate-300" data-tl-step aria-live="polite">
              Frame {index + 1} of {frames.length} · t = {frame.at_s} s
            </span>
          </div>

          <svg
            viewBox="0 0 1000 70"
            role="img"
            aria-label={`Time axis from 0 to ${Math.round(maxT)} seconds, ${frames.length} frames, at ${frame.at_s} seconds`}
            className="w-full"
            style={{ height: 'auto', maxHeight: 90 }}
          >
            <line x1={20} y1={35} x2={980} y2={35} className="stroke-slate-400 dark:stroke-slate-500" strokeWidth={2} />
            {frames.map((f, i) => {
              const x = 20 + (f.at_s / maxT) * 960
              return (
                <g key={i}>
                  <line
                    x1={x}
                    y1={26}
                    x2={x}
                    y2={44}
                    className={f.cause === 'derived' ? 'stroke-amber-500' : 'stroke-sky-600 dark:stroke-sky-400'}
                    strokeWidth={3}
                  />
                  <text x={x} y={62} textAnchor="middle" fontSize={14} className="fill-slate-600 dark:fill-slate-300">
                    {f.at_s}s
                  </text>
                </g>
              )
            })}
            <circle
              cx={20 + (frame.at_s / maxT) * 960}
              cy={35}
              r={9}
              className="fill-emerald-600 stroke-white dark:stroke-slate-900"
              strokeWidth={3}
              style={reduced ? undefined : { transition: 'cx 0.6s ease' }}
              data-tl-marker
            />
          </svg>

          <div className="rounded-lg border border-slate-200 p-4 dark:border-slate-800" data-tl-frame>
            <h2 className="text-lg font-semibold">{frame.label}</h2>
            <div className="mt-2 flex flex-wrap items-center gap-1">
              {frame.basis?.map((b) => (
                <BasisBadge key={b.rule} basis={b} />
              ))}
            </div>
            {frame.first_packet && (
              <p className="mt-3 text-sm" data-tl-first-packet>
                <b>First packets</b> take the standing path: {frame.first_packet.label}. {frame.first_packet.reason}.{' '}
                <BasisBadge basis={frame.first_packet.basis} />
              </p>
            )}
            {frame.changes.length > 0 && (
              <ul className="mt-3 space-y-1 text-sm" data-tl-changes>
                {frame.changes.map((c) => (
                  <li key={c.id}>
                    <code className="text-xs">{c.id}</code>: {stateLabel[c.from]} → <b>{stateLabel[c.to]}</b>.{' '}
                    <span className="text-slate-600 dark:text-slate-300">{c.reason}</span>
                  </li>
                ))}
              </ul>
            )}
            {frame.failovers.length > 0 && scenario === 'failover' && (
              <ul className="mt-3 space-y-2 text-sm" data-tl-failovers>
                {frame.failovers
                  .filter((f) => f.spoke === spoke)
                  .map((f) => (
                    <li key={f.spoke} className="rounded-md border border-slate-200 p-2 dark:border-slate-800">
                      <p>
                        <b>{f.spoke}</b>: local flows use <b data-tl-local>{f.local_hub}</b>
                        {f.local_note ? ` (${f.local_note})` : ''}; peer flows use <b data-tl-peer>{f.peer_hub}</b>
                        {f.peer_note ? ` (${f.peer_note})` : ''}.
                      </p>
                      <div className="mt-1 flex flex-wrap items-center gap-1">
                        <Bases basis={f.basis} />
                        {f.not_modelled?.map((n) => (
                          <Bases key={n.rule} basis={n.basis} />
                        ))}
                      </div>
                    </li>
                  ))}
              </ul>
            )}
          </div>

          <section aria-labelledby="tl-tun-h">
            <h3 id="tl-tun-h" className="text-base font-semibold">
              Tunnels that change in this timeline
            </h3>
            <ul className="mt-2 space-y-2" data-tl-tunnels>
              {changing.map((tn) => (
                <li
                  key={tn.id}
                  data-tl-tunnel={tn.id}
                  data-state={tn.state}
                  className="flex flex-wrap items-center gap-2 rounded-md border border-slate-200 p-2 text-sm dark:border-slate-800"
                >
                  <span className="font-medium">{typeLabel[tn.type]}</span>
                  <span>
                    {tn.a.name} <span aria-hidden>↔</span> {tn.b.name}
                  </span>
                  <StateBadge state={tn.state} />
                </li>
              ))}
              {changing.length === 0 && (
                <li className="text-sm text-slate-500">No tunnel changes state in this timeline.</li>
              )}
            </ul>
          </section>

          <TunnelMap
            topology={t}
            tunnels={mapTunnels}
            types={MAP_TYPES}
            secondaryGateway={false}
            showPotential={true}
            unreachable={unreachable}
          />
          <MapLegend />

          <section aria-labelledby="tl-steps-h">
            <h3 id="tl-steps-h" className="text-base font-semibold">
              All frames
            </h3>
            <ol className="mt-2 space-y-1 text-sm" data-tl-frames>
              {frames.map((f, i) => (
                <li key={i}>
                  <button
                    type="button"
                    onClick={() => go(i)}
                    aria-current={i === index ? 'step' : undefined}
                    className={`w-full rounded-md border px-3 py-1.5 text-left hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${i === index ? 'border-sky-500 bg-sky-50 dark:bg-sky-950/30' : 'border-slate-200 dark:border-slate-800'}`}
                  >
                    t = {f.at_s} s: {f.label}
                  </button>
                </li>
              ))}
            </ol>
          </section>
          {data.not_modelled.length > 0 && (
            <div className="space-y-2">
              {data.not_modelled.map((n) => (
                <NotModelledNote key={n.rule} nm={n} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

const btn =
  'rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-50 dark:border-slate-700'
