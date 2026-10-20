import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  applyLabFix,
  checkLabTask,
  fetchLab,
  fetchLabDebrief,
  putLabStatus,
  revealLabHint,
  type LabCheck,
  type LabDebriefItem,
  type LabDetail,
  type TraceResult,
} from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { LessonLink } from './LessonLink'
import { withLab } from './labParams'
import { LAB_STATUS_STYLE, LAB_STATUS_TEXT, difficultyText } from './labText'
import { VERDICT_TEXT } from './traceText'
import { useAsync } from './useAsync'

const btn =
  'rounded-md border border-slate-300 px-3 py-1.5 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700'

function CheckResult({ r }: { r: LabCheck }) {
  return (
    <div
      role="status"
      data-lab-result={r.correct ? 'correct' : 'incorrect'}
      className={`mt-3 rounded-lg border p-3 text-sm ${
        r.correct
          ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-100'
          : 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100'
      }`}
    >
      <p className="font-semibold">{r.correct ? 'Correct.' : 'Not quite. You can try again, or ask for a hint.'}</p>
      {!r.correct && r.expected && (
        <p className="mt-1" data-lab-expected>
          The simulator's answer: {r.expected}
        </p>
      )}
      {r.correct && r.explanation && (
        <p className="mt-1" data-lab-explanation>
          {r.explanation}
        </p>
      )}
      {!r.correct && r.explanation && (
        <p className="mt-1" data-lab-explanation>
          {r.explanation}
        </p>
      )}
      {r.detail && <p className="mt-1 text-xs">What the simulator reports: {r.detail}</p>}
      {r.assumptions.length > 0 && (
        <p className="mt-2 text-xs" data-lab-assumptions>
          This answer rests on {r.assumptions.length === 1 ? 'an assumption' : 'assumptions'} that the guides do not
          state: {r.assumptions.map((a) => `assumption ${a.number} (${a.text})`).join('; ')}
        </p>
      )}
    </div>
  )
}

function TraceSummary({ t }: { t: TraceResult }) {
  return (
    <div
      className={`mt-3 rounded-lg border p-3 text-sm ${t.verdict === 'delivered' ? 'border-emerald-300 bg-emerald-50/70 dark:border-emerald-800 dark:bg-emerald-950/30' : 'border-red-300 bg-red-50/70 dark:border-red-800 dark:bg-red-950/30'}`}
      data-lab-trace={t.verdict}
    >
      <p className="font-semibold">
        After this change: {VERDICT_TEXT[t.verdict]}
        {t.blocked_at > 0 ? ` at stage ${t.blocked_at}` : ''}
      </p>
      <p className="mt-1">{t.summary}</p>
    </div>
  )
}

/** /sim/labs/:labId: the symptom, tasks, progressive hints, screens to inspect, fixes and a completion card. */
export default function LabRunnerPage() {
  const { labId = '' } = useParams()
  const lab = useAsync(() => fetchLab(labId), labId)
  if (lab.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (lab.status === 'error') {
    return (
      <ErrorPanel title={`Lab "${labId}" is unavailable`} error={lab.error}>
        <p className="mt-3 text-sm">
          <Link to="/sim/labs" className="underline">
            Back to the labs
          </Link>
        </p>
      </ErrorPanel>
    )
  }
  return <Runner key={lab.data.id} lab={lab.data} />
}

/** Where each finding would be found in the real VCO, using the guides' own menu names; unverified locations are marked. */
function Debrief({ labId, status }: { labId: string; status: string }) {
  const [items, setItems] = useState<LabDebriefItem[]>([])
  useEffect(() => {
    if (status !== 'completed' && status !== 'skipped') return
    let alive = true
    fetchLabDebrief(labId)
      .then((d) => alive && setItems(d.items))
      .catch(() => alive && setItems([]))
    return () => {
      alive = false
    }
  }, [labId, status])
  if (items.length === 0) return null
  return (
    <section className="mt-4 rounded-lg border border-slate-300 p-4 text-sm dark:border-slate-700" data-lab-debrief>
      <h2 className="text-base font-semibold">Debrief: where you would find each finding in the real VCO</h2>
      <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
        Menu names are the ones the Arista guides use. A line marked unverified names a location the guides read do not
        confirm.
      </p>
      <ul className="mt-2 space-y-2">
        {items.map((d, i) => (
          <li key={i} data-lab-debrief-item={d.unverified ? 'unverified' : 'documented'}>
            <p>{d.finding}</p>
            <p className="text-xs">
              <span className="font-medium">Where:</span> {d.where}
              {d.unverified && (
                <span className="ml-2 rounded-full border border-orange-600 px-2 py-0.5 text-orange-900 dark:border-orange-400 dark:text-orange-200">
                  unverified path
                </span>
              )}
            </p>
            {d.unverified && d.note && <p className="text-xs text-slate-600 dark:text-slate-300">{d.note}</p>}
          </li>
        ))}
      </ul>
    </section>
  )
}

function Runner({ lab }: { lab: LabDetail }) {
  const navigate = useNavigate()
  const [picked, setPicked] = useState<Record<string, string>>({})
  const [results, setResults] = useState<Record<string, LabCheck>>({})
  const [hints, setHints] = useState<{ index: number; text: string; task: string }[]>([])
  const [applied, setApplied] = useState<string[]>([])
  const [fixTrace, setFixTrace] = useState<TraceResult | null>(null)
  const [status, setStatus] = useState(lab.progress.status)
  const [checks, setChecks] = useState(0)
  const [error, setError] = useState<Error | null>(null)
  const [busy, setBusy] = useState(false)
  const done = useRef(false)

  const allCorrect = lab.task_list.every((t) => results[t.id]?.correct)
  useEffect(() => {
    if (allCorrect && !done.current) {
      done.current = true
      putLabStatus(lab.id, 'completed')
        .then(() => setStatus('completed'))
        .catch((e: Error) => setError(e))
    }
  }, [allCorrect, lab.id])

  const labSearch = useMemo(() => {
    const q = new URLSearchParams({ lab: lab.id })
    if (applied.length) q.set('fixes', applied.join(','))
    return `?${q.toString()}`
  }, [lab.id, applied])

  const run = async (fn: () => Promise<void>) => {
    setBusy(true)
    setError(null)
    try {
      await fn()
    } catch (e) {
      setError(e as Error)
    } finally {
      setBusy(false)
    }
  }

  const check = (taskId: string, answer: string) =>
    run(async () => {
      const r = await checkLabTask(lab.id, taskId, answer, applied)
      setResults((x) => ({ ...x, [taskId]: r }))
      setChecks((n) => n + 1)
      setStatus((s) => (s === 'completed' ? s : 'started'))
    })

  const applyFix = (fixId: string) =>
    run(async () => {
      const r = await applyLabFix(lab.id, fixId, applied)
      setApplied(r.applied)
      setFixTrace(r.trace ?? null)
      setResults((x) => {
        const copy = { ...x }
        for (const t of lab.task_list) if (t.type === 'fix') delete copy[t.id]
        return copy
      })
    })

  const nextHint = () =>
    run(async () => {
      const n = hints.length + 1
      const h = await revealLabHint(lab.id, n)
      setHints((x) => [...x, { index: h.index, text: h.text, task: h.task }])
    })

  const leave = (s: 'started' | 'skipped') =>
    run(async () => {
      await putLabStatus(lab.id, s)
      navigate('/sim/labs')
    })

  const flowPath = lab.flow ? withLab(`/sim/${lab.topology}/trace`, labSearch) : ''
  return (
    <div className="max-w-3xl" data-lab-runner={lab.id}>
      <p className="text-sm text-slate-500 dark:text-slate-400">
        <Link to="/sim" className="underline">
          Simulator
        </Link>{' '}
        /{' '}
        <Link to="/sim/labs" className="underline">
          Labs
        </Link>{' '}
        / {lab.title}
      </p>
      <h1 className="mt-1 text-2xl font-semibold">{lab.title}</h1>
      <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
        <span
          className={`rounded-full border px-2 py-0.5 text-xs ${LAB_STATUS_STYLE[status]}`}
          data-lab-status={status}
        >
          {LAB_STATUS_TEXT[status]}
        </span>
        Difficulty: {difficultyText(lab.difficulty)} · about {lab.estimated_minutes} min
      </p>
      <p className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs">
        {lab.goals.map((g) => (
          <LessonLink key={g.id} id={g.id} prefix="Lesson" />
        ))}
      </p>

      <section
        className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
        aria-labelledby="lab-symptom"
      >
        <h2 id="lab-symptom" className="text-base font-semibold">
          The symptom
        </h2>
        <p className="mt-1 text-sm" data-lab-symptom>
          {lab.symptom}
        </p>
        {lab.relies_on_assumptions.length > 0 && (
          <p className="mt-2 text-xs text-orange-900 dark:text-orange-200" data-lab-relies>
            Assumptions this lab relies on: {lab.relies_on_assumptions.join(', ')}. They are not verified; the simulator
            marks each one with a badge.
          </p>
        )}
      </section>

      <section
        className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
        aria-labelledby="lab-screens"
      >
        <h2 id="lab-screens" className="text-base font-semibold">
          Inspect the network
        </h2>
        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
          These open the mock VCO screens on the lab's faulty network
          {applied.length > 0 ? ' with your fixes applied' : ''}. Read the answers from them.
        </p>
        <ul className="mt-2 list-disc space-y-1 pl-5 text-sm">
          {lab.screens.map((s) => (
            <li key={s.path}>
              <Link to={withLab(`/sim/${lab.topology}/${s.path}`, labSearch)} className="underline" data-lab-screen>
                {s.label}
              </Link>
            </li>
          ))}
          {flowPath && (
            <li>
              <Link to={flowPath} className="underline" data-lab-screen>
                Flow tracer for the flow in question
              </Link>
            </li>
          )}
        </ul>
      </section>

      <ol className="mt-4 space-y-4" data-lab-tasks>
        {lab.task_list.map((t, i) => {
          const r = results[t.id]
          return (
            <li
              key={t.id}
              className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
              data-lab-task={t.id}
              data-task-type={t.type}
              data-task-done={r?.correct ? 'true' : 'false'}
            >
              <fieldset disabled={busy}>
                <legend className="px-1 text-base font-semibold">
                  {i + 1}. <span className="text-xs font-normal uppercase tracking-wide text-slate-500">{t.type}</span>{' '}
                  {t.prompt}
                </legend>
                {t.type !== 'fix' ? (
                  <div className="mt-2 space-y-2">
                    {t.choices.map((c) => (
                      <label
                        key={c.id}
                        className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-200 p-2 text-sm has-[:checked]:border-sky-500 dark:border-slate-700"
                      >
                        <input
                          type="radio"
                          name={`task-${t.id}`}
                          value={c.id}
                          checked={picked[t.id] === c.id}
                          onChange={() => setPicked((p) => ({ ...p, [t.id]: c.id }))}
                          className="mt-1"
                          data-lab-choice={c.id}
                        />
                        <span>{c.text}</span>
                      </label>
                    ))}
                    <button
                      type="button"
                      className={btn}
                      disabled={!picked[t.id]}
                      onClick={() => check(t.id, picked[t.id]!)}
                      data-lab-check={t.id}
                    >
                      Check my answer
                    </button>
                  </div>
                ) : (
                  <div className="mt-2 space-y-2">
                    <p className="text-xs text-slate-600 dark:text-slate-300">
                      Pick a change and apply it: the simulator recomputes the network at once. Applied so far:{' '}
                      {applied.length === 0 ? 'nothing' : applied.length}.
                    </p>
                    {lab.fixes.map((f) => (
                      <div
                        key={f.id}
                        className="rounded-md border border-slate-200 p-2 text-sm dark:border-slate-700"
                        data-lab-fix={f.id}
                        data-applied={applied.includes(f.id) ? 'true' : 'false'}
                      >
                        <p className="font-medium">{f.title}</p>
                        <p className="text-xs text-slate-600 dark:text-slate-300">{f.description}</p>
                        <button
                          type="button"
                          className={`${btn} mt-1`}
                          disabled={applied.includes(f.id)}
                          onClick={() => applyFix(f.id)}
                          data-lab-apply={f.id}
                        >
                          {applied.includes(f.id) ? 'Applied' : 'Apply this change'}
                        </button>
                      </div>
                    ))}
                    {fixTrace && <TraceSummary t={fixTrace} />}
                    <button type="button" className={btn} onClick={() => check(t.id, '')} data-lab-check={t.id}>
                      Check the network now
                    </button>
                  </div>
                )}
              </fieldset>
              {r && <CheckResult r={r} />}
            </li>
          )
        })}
      </ol>

      <section
        className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
        aria-labelledby="lab-hints"
      >
        <h2 id="lab-hints" className="text-base font-semibold">
          Hints
        </h2>
        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
          Hints come one at a time, from gentle to specific. Using them is fine.
        </p>
        {hints.length > 0 && (
          <ol className="mt-2 list-decimal space-y-1 pl-5 text-sm" data-lab-hints>
            {hints.map((h) => (
              <li key={h.index} data-lab-hint={h.index}>
                {h.text}
              </li>
            ))}
          </ol>
        )}
        <button
          type="button"
          className={`${btn} mt-2`}
          disabled={busy || hints.length >= lab.hint_count}
          onClick={nextHint}
          data-lab-next-hint
        >
          {hints.length >= lab.hint_count ? 'No more hints' : `Show hint ${hints.length + 1} of ${lab.hint_count}`}
        </button>
      </section>

      {error && (
        <div className="mt-4" data-lab-error>
          <ErrorPanel title="Something went wrong" error={error} />
        </div>
      )}

      {allCorrect && (
        <section
          role="status"
          className="mt-4 rounded-lg border border-emerald-400 bg-emerald-50 p-4 text-sm text-emerald-950 dark:border-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-100"
          data-lab-complete
        >
          <h2 className="text-base font-semibold">Lab complete</h2>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            <li>
              Checks made: {checks} · hints used: {hints.length}
            </li>
            <li>
              Changes applied:{' '}
              {applied.length === 0
                ? 'none'
                : applied.map((id) => lab.fixes.find((f) => f.id === id)?.title ?? id).join('; ')}
            </li>
            <li>
              Reinforced lessons:{' '}
              {lab.goals.map((g, i) => (
                <span key={g.id}>
                  {i > 0 ? ', ' : ''}
                  <LessonLink id={g.id} prefix="Lesson" />
                </span>
              ))}
            </li>
          </ul>
          <p className="mt-2">
            <Link to="/sim/labs" className="font-medium underline" data-lab-back>
              Back to the labs
            </Link>
          </p>
        </section>
      )}

      <Debrief labId={lab.id} status={status} />

      <div className="mt-5 flex flex-wrap gap-2">
        <button type="button" className={btn} onClick={() => leave('skipped')} data-lab-skip>
          Skip this lab
        </button>
        <button type="button" className={btn} onClick={() => leave('started')} data-lab-later>
          I will finish it later
        </button>
      </div>
    </div>
  )
}
