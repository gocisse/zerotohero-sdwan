import { useEffect, useMemo, useState } from 'react'
import { postReachability, type ReachResult, type ReachStage } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { AssumptionsPanel, RoutingConditions } from './RoutingControls'
import { sourceText, STATUS_TEXT, switchValueText } from './routingText'
import { useTopologyCtx } from './topologyContext'
import { defaultQuery, originatedPrefixes } from './routingQuery'
import { LessonLink } from './LessonLink'
import { Bases, BasisBadge, NotModelledNote } from './ui'

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

const STATUS_STYLE: Record<string, string> = {
  ok: 'border-emerald-400 bg-emerald-50/60 dark:border-emerald-700 dark:bg-emerald-950/20',
  failed: 'border-red-400 bg-red-50/70 dark:border-red-700 dark:bg-red-950/30',
  skipped: 'border-slate-200 dark:border-slate-800',
  'not-reached': 'border-slate-200 opacity-70 dark:border-slate-800',
  'not-modelled': 'border-dashed border-rose-300 bg-rose-50/60 dark:border-rose-800 dark:bg-rose-950/20',
}
const STATUS_MARK: Record<string, string> = {
  ok: '✓',
  failed: '✗',
  skipped: '–',
  'not-reached': '·',
  'not-modelled': '?',
}

function StageCard({ s, first }: { s: ReachStage; first: boolean }) {
  return (
    <li
      data-stage={s.id}
      data-status={s.status}
      data-first-failing={first ? 'true' : 'false'}
      className={`rounded-md border p-3 text-sm ${STATUS_STYLE[s.status]} ${first ? 'ring-2 ring-red-500' : ''}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span aria-hidden className="font-mono font-bold">
          {STATUS_MARK[s.status]}
        </span>
        <span className="font-medium">
          {s.number}. {s.question}
        </span>
        <span className="rounded-full border border-slate-400 px-2 py-0.5 text-xs">{STATUS_TEXT[s.status]}</span>
        {first && (
          <span className="rounded-full border border-red-500 px-2 py-0.5 text-xs font-semibold text-red-900 dark:text-red-100">
            first failing stage
          </span>
        )}
      </div>
      {s.detail && (
        <p className="mt-1 text-xs text-slate-700 dark:text-slate-200" data-stage-detail>
          {s.detail}
        </p>
      )}
      {s.evidence && s.evidence.length > 0 && (
        <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-slate-600 dark:text-slate-300">
          {s.evidence.map((e, i) => (
            <li key={i}>{e}</li>
          ))}
        </ul>
      )}
      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
        <Bases basis={s.basis} also={s.also} />
        <span className="text-slate-500 dark:text-slate-400">Change it in: {s.where}.</span>
        <LessonLink id={s.lesson_ref} />
      </div>
    </li>
  )
}

/** The reachability explainer: follow one prefix from its owner to a source Edge and see where it stops and why. */
export default function ReachabilityPage() {
  const { detail, topologyId, conditions, assumptions } = useTopologyCtx()
  const t = detail.topology
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])
  const origins = useMemo(() => originatedPrefixes(t), [t])
  const init = useMemo(() => defaultQuery(t), [t])
  const [source, setSource] = useState(init.source)
  const [dest, setDest] = useState(init.destination)
  const [segment, setSegment] = useState('global')
  const [asked, setAsked] = useState({ source: init.source, destination: init.destination, segment: 'global' })
  const [state, setState] = useState<{ result: ReachResult | null; error: Error | null; busy: boolean }>({
    result: null,
    error: null,
    busy: true,
  })
  const key = JSON.stringify([topologyId, asked, conditions, assumptions])

  useEffect(() => {
    let cancelled = false
    setState((s) => ({ ...s, busy: true }))
    postReachability(topologyId, asked, conditions, assumptions)
      .then((result) => !cancelled && setState({ result, error: null, busy: false }))
      .catch((error: Error) => !cancelled && setState((s) => ({ ...s, error, busy: false })))
    return () => {
      cancelled = true
    }
    // `key` stands for the four inputs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])

  const r = state.result
  const srcName = edges.find((e) => e.id === asked.source)?.name ?? asked.source
  return (
    <div data-reachability-page>
      <section className="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
        <h2 className="text-base font-semibold">Why does Edge X have (or not have) a route to this prefix?</h2>
        <p className="mt-1 text-sm text-slate-700 dark:text-slate-200">
          Follow the prefix through the six checks of the Section 6 diagnostic. This order is our teaching method, not
          Arista's procedure.
        </p>
        <form
          className="mt-3 grid gap-3 sm:grid-cols-4"
          onSubmit={(e) => {
            e.preventDefault()
            setAsked({ source, destination: dest.trim(), segment })
          }}
        >
          <label className="text-sm">
            <span className="font-medium">Source Edge</span>
            <select className={field} value={source} onChange={(e) => setSource(e.target.value)} data-reach-source>
              {edges.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm sm:col-span-2">
            <span className="font-medium">Destination prefix or IP</span>
            <input
              className={field}
              list="reach-prefixes"
              value={dest}
              onChange={(e) => setDest(e.target.value)}
              data-reach-dest
              placeholder="10.20.0.0/24"
            />
            <datalist id="reach-prefixes">
              {origins.map((o, i) => (
                <option key={i} value={o.prefix}>
                  {o.label}
                </option>
              ))}
            </datalist>
          </label>
          <label className="text-sm">
            <span className="font-medium">Segment</span>
            <select className={field} value={segment} onChange={(e) => setSegment(e.target.value)} data-reach-segment>
              {t.segments.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <div className="sm:col-span-4">
            <button
              type="submit"
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
              data-reach-run
            >
              Explain
            </button>
            <span className="ml-3 text-xs text-slate-500 dark:text-slate-400">
              Known prefixes:{' '}
              {origins
                .slice(0, 6)
                .map((o) => o.prefix)
                .join(', ')}
              {origins.length > 6 ? ', …' : ''}
            </span>
          </div>
        </form>
      </section>

      {state.error && (
        <div className="mt-4" data-reach-error>
          <ErrorPanel title="The question could not be answered" error={state.error} />
        </div>
      )}
      {r && (
        <div
          className="mt-5"
          aria-live="polite"
          data-reach-result
          data-outcome={r.outcome}
          data-first-failing={r.first_failing}
        >
          <div
            className={`rounded-lg border p-4 text-sm ${
              r.outcome === 'reachable'
                ? 'border-emerald-400 bg-emerald-50/70 dark:border-emerald-700 dark:bg-emerald-950/30'
                : r.outcome === 'unreachable'
                  ? 'border-red-400 bg-red-50/70 dark:border-red-700 dark:bg-red-950/30'
                  : 'border-dashed border-rose-300 bg-rose-50/60 dark:border-rose-800 dark:bg-rose-950/20'
            }`}
          >
            <p className="font-semibold" data-reach-verdict>
              {srcName} to {r.prefix} in segment {r.query.segment}:{' '}
              {r.outcome === 'reachable'
                ? 'reachable'
                : r.outcome === 'unreachable'
                  ? 'not reachable'
                  : 'cannot be answered (not modelled)'}
              {state.busy ? ' (updating…)' : ''}
            </p>
            <p className="mt-1" data-reach-reason>
              {r.reason}
            </p>
            <p className="mt-2 text-xs">
              <LessonLink id={r.lesson_ref} prefix="Matching lesson" />
              {r.path && (
                <>
                  {' '}
                  · <LessonLink id="ov-diagnosis" prefix="Paths" />
                </>
              )}
            </p>
          </div>
          {r.depends_on.length > 0 && (
            <div
              role="note"
              data-reach-depends
              className="mt-3 rounded-lg border border-orange-300 bg-orange-50/70 p-3 text-sm dark:border-orange-800 dark:bg-orange-950/30"
            >
              <p className="font-semibold">This answer depends on unverified assumptions.</p>
              <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs">
                {r.depends_on.map((d) => (
                  <li key={d.switch.key} data-depends={d.switch.key}>
                    {d.switch.label} is {switchValueText(d.switch)} (assumption {d.switch.assumption}). Set to{' '}
                    {String(d.flipped_value).replace(/_/g, ' ')}, the answer becomes {d.new_outcome}
                    {d.new_first_failing ? ` (first failing stage: ${d.new_first_failing})` : ''}.
                  </li>
                ))}
              </ul>
            </div>
          )}
          {r.not_modelled
            .filter((n) => n.rule === 'nm-rt-gateway-reflector')
            .map((n) => (
              <p key={n.rule} className="mt-3 text-xs text-slate-600 dark:text-slate-300" data-gateway-note>
                Not modelled: Gateway as route reflector. {n.reason} <BasisBadge basis={n.basis} />
              </p>
            ))}
          <ol className="mt-4 space-y-2" data-reach-stages>
            {r.stages.map((s) => (
              <StageCard key={s.id} s={s} first={s.id === r.first_failing} />
            ))}
          </ol>

          {r.origins.length > 0 && (
            <section className="mt-5" aria-labelledby="ro-h" data-reach-origins>
              <h3 id="ro-h" className="text-base font-semibold">
                Exits for this prefix
              </h3>
              <ul className="mt-2 space-y-1 text-sm">
                {r.origins.map((o) => (
                  <li
                    key={o.edge + o.via + o.prefix}
                    data-origin={o.edge}
                    data-origin-status={o.status}
                    className="rounded-md border border-slate-200 p-2 dark:border-slate-800"
                  >
                    <span className="font-medium">{o.name}</span> offers {o.prefix} ({o.source}):{' '}
                    {o.status === 'ok'
                      ? o.selected
                        ? 'its route is the winner.'
                        : 'it reaches the source Edge.'
                      : `stops at stage "${o.stage}".`}{' '}
                    {o.reason} <Bases basis={o.basis} also={o.also} />
                  </li>
                ))}
              </ul>
            </section>
          )}
          {r.candidates.length > 0 && (
            <section className="mt-5" aria-labelledby="rcand-h">
              <h3 id="rcand-h" className="text-base font-semibold">
                Candidates in {srcName}'s table
              </h3>
              <ul className="mt-2 space-y-1 text-xs">
                {r.candidates.map((c) => (
                  <li key={c.id} data-candidate data-selected={c.selected ? 'true' : 'false'}>
                    #{c.rank} <code>{c.prefix}</code> {sourceText(c)}
                    {c.origin_name ? ` from ${c.origin_name}` : ''}: {c.reason} <Bases basis={c.basis} also={c.also} />
                  </li>
                ))}
              </ul>
            </section>
          )}
          <p className="mt-4 text-xs text-slate-500 dark:text-slate-400">{r.return_traffic}</p>
          {r.not_modelled.length > 0 && (
            <div className="mt-4 grid gap-2 md:grid-cols-2" data-reach-not-modelled>
              {r.not_modelled.map((n) => (
                <NotModelledNote key={n.rule} nm={n} />
              ))}
            </div>
          )}
        </div>
      )}
      <div className="mt-5 grid gap-5 lg:grid-cols-2">
        <RoutingConditions />
        <AssumptionsPanel switches={r?.assumptions ?? []} />
      </div>
    </div>
  )
}
