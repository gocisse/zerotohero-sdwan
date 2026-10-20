import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { postReachability, type ReachResult, type StageId } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { originatedPrefixes } from './routingQuery'
import { STAGE_TITLE, switchValueText } from './routingText'
import { withLab } from './labParams'
import { useTopologyCtx } from './topologyContext'
import { Bases } from './ui'

const STAGES: StageId[] = ['known', 'advertised', 'filtered', 'redistributed', 'winner', 'path']

function shuffle<T>(list: T[]): T[] {
  const a = [...list]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j]!, a[i]!]
  }
  return a
}

/**
 * Routing challenge: "Why can't Branch B reach this prefix?" The choices are the engine's own failing stages and the
 * answer is the engine's first failing stage. Ungraded, nothing is stored, never blocks anything.
 */
export default function RoutingChallengePage() {
  const { detail, topologyId, conditions, assumptions, labSearch } = useTopologyCtx()
  const t = detail.topology
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])
  const [pool, setPool] = useState<ReachResult[] | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const key = JSON.stringify([topologyId, conditions, assumptions])

  useEffect(() => {
    let cancelled = false
    const prefixes = [...new Set(originatedPrefixes(t).map((o) => o.prefix))].slice(0, 10)
    const qs = prefixes.flatMap((p) => edges.map((e) => ({ source: e.id, destination: p }))).slice(0, 40)
    Promise.all(qs.map((q) => postReachability(topologyId, q, conditions, assumptions)))
      .then((all) => !cancelled && setPool(all.filter((r) => r.outcome === 'unreachable' && r.first_failing)))
      .catch((e: Error) => !cancelled && setError(e))
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])

  const [index, setIndex] = useState(0)
  const [picked, setPicked] = useState<StageId | null>(null)
  const seed = useRef(Math.random())
  const current = pool && pool.length > 0 ? pool[(index + Math.floor(seed.current * pool.length)) % pool.length]! : null
  const choices = useMemo(() => {
    if (!current) return []
    const correct = current.first_failing as StageId
    return shuffle([correct, ...shuffle(STAGES.filter((s) => s !== correct)).slice(0, 3)])
  }, [current])

  if (error) return <ErrorPanel title="The routing challenge is unavailable" error={error} />
  if (!pool) return <p className="text-slate-500">Loading…</p>
  const name = (id: string) => edges.find((e) => e.id === id)?.name ?? id
  const correct = current?.first_failing as StageId | undefined
  const stage = current?.stages.find((s) => s.id === correct)

  return (
    <div className="max-w-2xl" data-routing-challenge>
      <p className="text-sm text-slate-600 dark:text-slate-300">
        A practice question answered from the simulator's own reasons. It is not graded, nothing is saved, and you can
        skip it at any time. It uses the conditions and assumption switches you set on the other routing screens.
      </p>
      {!current ? (
        <p className="mt-4 text-sm text-slate-500" data-rc-empty>
          Under the current conditions and assumptions every prefix of this topology is reachable (or the topology has
          no routing data). Turn an assumption switch off, take a link down or bring a neighbor down on the{' '}
          <Link to={withLab(`/sim/${topologyId}/routes`, labSearch)} className="underline">
            Route Table Dump
          </Link>{' '}
          screen to get a question.
        </p>
      ) : (
        <>
          <fieldset
            className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
            disabled={picked !== null}
          >
            <legend className="px-1 text-base font-semibold" data-rc-question>
              Why can't {name(current.query.source)} reach {current.prefix}?
            </legend>
            <div className="space-y-2">
              {choices.map((c) => (
                <label
                  key={c}
                  className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-200 p-2 text-sm has-[:checked]:border-sky-500 dark:border-slate-700"
                >
                  <input
                    type="radio"
                    name="rc"
                    value={c}
                    checked={picked === c}
                    onChange={() => setPicked(c)}
                    className="mt-1"
                    data-rc-choice={c}
                  />
                  <span>{STAGE_TITLE[c]}</span>
                </label>
              ))}
            </div>
          </fieldset>
          {picked !== null && correct && (
            <div
              role="status"
              data-rc-result={picked === correct ? 'correct' : 'incorrect'}
              className={`mt-4 rounded-lg border p-4 text-sm ${
                picked === correct
                  ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-100'
                  : 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100'
              }`}
            >
              <p className="font-semibold">{picked === correct ? 'Correct.' : 'Not quite.'}</p>
              <p className="mt-1" data-rc-explanation>
                The first failing stage is {correct}:{' '}
                {STAGE_TITLE[correct].charAt(0).toLowerCase() + STAGE_TITLE[correct].slice(1)}. {stage?.detail}
              </p>
              {stage && (
                <div className="mt-2 flex flex-wrap items-center gap-1">
                  <Bases basis={stage.basis} also={stage.also} />
                </div>
              )}
              {current.depends_on.length > 0 ? (
                <p className="mt-2 font-medium" data-rc-depends>
                  Note: this answer depends on an unverified assumption:{' '}
                  {current.depends_on
                    .map((d) => `${d.switch.label} is ${switchValueText(d.switch)} (assumption ${d.switch.assumption})`)
                    .join('; ')}
                  . Change it on the Route Table Dump screen and the answer changes.
                </p>
              ) : (
                <p className="mt-2 text-xs" data-rc-nodepends>
                  No assumption switch changes this answer.
                </p>
              )}
              <p className="mt-2 text-xs">
                <Link to={withLab(`/sim/${topologyId}/reachability`, labSearch)} className="underline">
                  Open the reachability explainer
                </Link>
              </p>
            </div>
          )}
          <div className="mt-4">
            <button
              type="button"
              onClick={() => {
                setIndex((i) => i + 1)
                setPicked(null)
              }}
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
              data-rc-next
            >
              {picked !== null ? 'Next question' : 'Skip'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}
