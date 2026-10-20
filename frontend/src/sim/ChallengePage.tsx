import { useMemo, useState } from 'react'
import { fetchEffectiveConfig, type EffectiveSetting } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useTopologyCtx } from './topologyContext'
import { useAsync } from './useAsync'
import { Bases, SourceBadge } from './ui'
import { valueText } from './format'

type Answer = 'profile' | 'edge-override' | 'default'

const OPTIONS: { id: Answer; label: string }[] = [
  { id: 'profile', label: 'It is inherited from the Profile' },
  { id: 'edge-override', label: 'It is overridden on this Edge' },
  { id: 'default', label: 'It is a default: nobody sets it' },
]

interface Question {
  edge: string
  edgeName: string
  profileName: string
  setting: EffectiveSetting
}

/** Challenge mode: "Why does Edge X have setting Y?" Ungraded, never blocks anything, nothing is stored. */
export default function ChallengePage() {
  const { detail, topologyId } = useTopologyCtx()
  const edges = useMemo(() => detail.topology.sites.flatMap((s) => s.edges), [detail])
  const state = useAsync(
    () => Promise.all(edges.map((e) => fetchEffectiveConfig(topologyId, e.id))),
    `challenge/${topologyId}`,
  )
  const pool = useMemo<Question[]>(() => {
    if (state.status !== 'ready') return []
    return state.data.flatMap((ec) =>
      ec.device
        .filter((s) => s.source.kind === 'profile' || s.source.kind === 'edge-override' || s.source.kind === 'default')
        .map((s) => ({ edge: ec.edge.id, edgeName: ec.edge.name, profileName: ec.edge.profile_name, setting: s })),
    )
  }, [state])

  // The first question is chosen once per pool; "Next question" replaces it. No effect or stored state is needed.
  const [seed] = useState(() => Math.random())
  const initial = pool.length > 0 ? (pool[Math.floor(seed * pool.length)] ?? null) : null
  const [chosen, setChosen] = useState<{ pool: Question[]; q: Question } | null>(null)
  const current = chosen && chosen.pool === pool ? chosen.q : initial
  const [picked, setPicked] = useState<Answer | null>(null)

  const next = () => {
    if (pool.length === 0) return
    // Prefer a different question from the one just shown.
    const choices = pool.length > 1 ? pool.filter((q) => q !== current) : pool
    const q = choices[Math.floor(Math.random() * choices.length)]
    if (q) setChosen({ pool, q })
    setPicked(null)
  }

  if (state.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (state.status === 'error') return <ErrorPanel title="Challenge mode is unavailable" error={state.error} />
  if (!current) return <p className="text-sm text-slate-500">No question can be made from this topology.</p>

  const correct = current.setting.source.kind as Answer
  const answered = picked !== null
  return (
    <div className="max-w-2xl" data-challenge>
      <p className="text-sm text-slate-600 dark:text-slate-300">
        A practice question. It is not graded, nothing is saved, and you can skip it at any time.
      </p>
      <fieldset className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800" disabled={answered}>
        <legend className="px-1 text-base font-semibold" data-challenge-question>
          Why does {current.edgeName} have {current.setting.label} = {valueText(current.setting.value)}?
        </legend>
        <p className="mb-3 text-xs text-slate-500 dark:text-slate-400">
          {current.edgeName} uses Profile {current.profileName}.
        </p>
        <div className="space-y-2">
          {OPTIONS.map((o) => (
            <label
              key={o.id}
              className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-200 p-2 text-sm has-[:checked]:border-sky-500 dark:border-slate-700"
            >
              <input
                type="radio"
                name="why"
                value={o.id}
                checked={picked === o.id}
                onChange={() => setPicked(o.id)}
                className="mt-1"
                data-choice={o.id}
              />
              <span>{o.label}</span>
            </label>
          ))}
        </div>
      </fieldset>
      {answered && (
        <div
          role="status"
          data-challenge-result={picked === correct ? 'correct' : 'incorrect'}
          className={`mt-4 rounded-lg border p-4 text-sm ${
            picked === correct
              ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-100'
              : 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100'
          }`}
        >
          <p className="font-semibold">{picked === correct ? 'Correct.' : 'Not quite.'}</p>
          <p className="mt-1">{current.setting.why}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <SourceBadge source={current.setting.source} />
            <Bases basis={current.setting.basis} also={current.setting.also} />
          </div>
        </div>
      )}
      <div className="mt-4 flex gap-2">
        <button
          type="button"
          onClick={next}
          className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
          data-challenge-next
        >
          {answered ? 'Next question' : 'Skip'}
        </button>
      </div>
    </div>
  )
}
