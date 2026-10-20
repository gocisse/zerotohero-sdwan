import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  checkSelfCheckScreen,
  fetchSelfCheck,
  gradeSelfCheck,
  type AnswerValue,
  type SelfCheck,
  type SelfCheckGrade,
  type SelfCheckScreenResult,
} from '../api'
import ErrorPanel from '../components/ErrorPanel'
import Markdown from '../components/Markdown'
import { useCourseState } from '../course/context'
import { lessonPath } from '../lib/course'
import { GOALS, loadChecklist, saveChecklist, saveSelfCheck, type SelfCheckMemory } from '../progress/goals'
import QuestionInput from '../quiz/questions/QuestionInput'
import QuestionResultCard from '../quiz/QuestionResultCard'
import { withLab } from './labParams'
import { useAsync } from './useAsync'

const btn =
  'rounded-md border border-slate-300 px-3 py-1.5 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700'

const isAnswered = (v: AnswerValue | undefined) => v !== undefined && (!Array.isArray(v) || v.length > 0)

/** /sim/self-check: 12 mixed questions, 3 "read this screen" tasks and a personal checklist. Ungraded; nothing blocks. */
export default function SelfCheckPage() {
  const [run, setRun] = useState(0)
  const sc = useAsync(() => fetchSelfCheck(), `self-check:${run}`)
  return (
    <div data-self-check-page>
      <p className="text-sm text-slate-500 dark:text-slate-400">
        <Link to="/sim" className="underline">
          Simulator
        </Link>{' '}
        / Self-check
      </p>
      <h1 className="mt-1 text-2xl font-semibold">Self-check</h1>
      <p className="mt-1 max-w-3xl text-sm text-slate-600 dark:text-slate-300" data-self-check-note>
        An ungraded look at where you stand: 12 questions drawn from the quiz bank across every section, three tasks
        where you read the simulator's mock screens, and a checklist for yourself. Nothing is scored, nothing is stored
        on the server, nothing is locked. The checklist and your last result stay in this browser only.
      </p>
      {sc.status === 'loading' && <p className="mt-4 text-slate-500">Loading…</p>}
      {sc.status === 'error' && <ErrorPanel title="The self-check is unavailable" error={sc.error} />}
      {sc.status === 'ready' && <Body key={run} sc={sc.data} onAgain={() => setRun((r) => r + 1)} />}
      <Checklist />
    </div>
  )
}

/** Keeps the latest result in this browser only (never sent anywhere). */
function remember(
  g: SelfCheckGrade | null,
  screens: Record<string, SelfCheckScreenResult>,
  sectionOf: Map<string, string>,
) {
  const perSection: SelfCheckMemory['perSection'] = {}
  for (const r of g?.results ?? []) {
    const sid = sectionOf.get(r.question_id)
    if (!sid) continue
    const p = (perSection[sid] ??= { correct: 0, total: 0 })
    p.total += 1
    if (r.correct) p.correct += 1
  }
  const vals = Object.values(screens)
  saveSelfCheck({
    at: new Date().toISOString(),
    perSection,
    screens: { correct: vals.filter((v) => v.correct).length, total: vals.length },
  })
}

function Body({ sc, onAgain }: { sc: SelfCheck; onAgain: () => void }) {
  const state = useCourseState()
  const [answers, setAnswers] = useState<Record<string, AnswerValue>>({})
  const [grade, setGrade] = useState<SelfCheckGrade | null>(null)
  const [screenPick, setScreenPick] = useState<Record<string, string>>({})
  const [screenRes, setScreenRes] = useState<Record<string, SelfCheckScreenResult>>({})
  const [error, setError] = useState<Error | null>(null)
  const [busy, setBusy] = useState(false)

  const answered = sc.questions.filter((q) => isAnswered(answers[q.id])).length
  const byId = useMemo(() => new Map(sc.questions.map((q) => [q.id, q])), [sc.questions])
  const sectionOf = useMemo(() => new Map(sc.questions.map((q) => [q.id, q.section_id])), [sc.questions])
  const labSearch = `?lab=${encodeURIComponent(sc.lab)}`

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

  const check = () =>
    run(async () => {
      const subs = sc.questions
        .filter((q) => isAnswered(answers[q.id]))
        .map((q) => ({ question_id: q.id, answer: answers[q.id] as AnswerValue }))
      const g = await gradeSelfCheck(subs)
      setGrade(g)
      remember(g, screenRes, sectionOf)
    })

  const checkScreen = (taskId: string) =>
    run(async () => {
      const r = await checkSelfCheckScreen(taskId, screenPick[taskId] ?? '')
      const next = { ...screenRes, [taskId]: r }
      setScreenRes(next)
      remember(grade, next, sectionOf)
    })

  const sectionTitles = new Map(sc.questions.map((q) => [q.section_id, q.section_title]))
  const weakSections = new Map<string, number>()
  for (const r of grade?.results ?? []) {
    if (!r.correct) {
      const sid = byId.get(r.question_id)?.section_id
      if (sid) weakSections.set(sid, (weakSections.get(sid) ?? 0) + 1)
    }
  }

  return (
    <>
      <section aria-labelledby="sc-questions" className="mt-6" data-self-check-questions>
        <h2 id="sc-questions" className="text-lg font-semibold">
          Part 1: twelve questions
        </h2>
        <ol className="mt-3 space-y-5">
          {sc.questions.map((q, i) => {
            const r = grade?.results.find((x) => x.question_id === q.id)
            return (
              <li key={q.id} data-self-check-question={q.id}>
                <section
                  className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
                  aria-label={`Question ${i + 1}`}
                >
                  <div className="mb-1 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-slate-800">{q.section_title}</span>
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-slate-800">{q.topic}</span>
                  </div>
                  <div className="mb-3 text-base font-medium">
                    <Markdown>{`${i + 1}. ${q.prompt}`}</Markdown>
                  </div>
                  <QuestionInput
                    question={q}
                    value={answers[q.id]}
                    onChange={(v) => setAnswers((a) => ({ ...a, [q.id]: v }))}
                    disabled={!!r}
                  />
                  {r && (
                    <div className="mt-3">
                      <QuestionResultCard
                        index={i + 1}
                        question={q}
                        result={{
                          ...r,
                          given: r.given as AnswerValue | null,
                          correct_answer: r.correct_answer as AnswerValue,
                        }}
                        lessonTo={
                          state.status === 'ready' && r.lesson_ref ? lessonPath(state.course, r.lesson_ref) : undefined
                        }
                      />
                    </div>
                  )}
                </section>
              </li>
            )
          })}
        </ol>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button
            type="button"
            className={btn}
            onClick={() => void check()}
            disabled={busy || answered === 0 || !!grade}
            data-self-check-submit
          >
            Check my answers ({answered} of {sc.questions.length} answered)
          </button>
          <button type="button" className={btn} onClick={onAgain} data-self-check-again>
            New self-check
          </button>
        </div>
        {grade && (
          <div
            role="status"
            className="mt-3 rounded-lg border border-slate-300 p-3 text-sm dark:border-slate-700"
            data-self-check-summary
          >
            <p className="font-semibold">
              {grade.score} of {grade.total} answered questions correct. This is not a grade and is not stored on the
              server.
            </p>
            {weakSections.size > 0 ? (
              <p className="mt-1">
                Worth a second look:{' '}
                {[...weakSections.entries()].map(([sid, n], i) => (
                  <span key={sid}>
                    {i > 0 && ', '}
                    <Link className="underline" to={`/course/${sid}`}>
                      {sectionTitles.get(sid) ?? sid}
                    </Link>{' '}
                    ({n})
                  </span>
                ))}
              </p>
            ) : (
              <p className="mt-1">No missed questions in what you answered.</p>
            )}
          </div>
        )}
      </section>

      <section aria-labelledby="sc-screens" className="mt-8" data-self-check-screens>
        <h2 id="sc-screens" className="text-lg font-semibold">
          Part 2: read this screen
        </h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-600 dark:text-slate-300">{sc.symptom}</p>
        <ol className="mt-3 space-y-4">
          {sc.screens.map((s, i) => {
            const r = screenRes[s.id]
            return (
              <li
                key={s.id}
                className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
                data-self-check-screen={s.id}
              >
                <p className="font-medium">
                  {i + 1}. {s.prompt}
                </p>
                <p className="mt-1 text-sm">
                  <Link
                    to={withLab(`/sim/${s.topology}/${s.screen.path}`, labSearch)}
                    className="underline"
                    data-self-check-screen-link
                  >
                    Open: {s.screen.label}
                  </Link>
                </p>
                <fieldset className="mt-2">
                  <legend className="sr-only">Answer for screen task {i + 1}</legend>
                  <div className="space-y-1.5">
                    {s.choices.map((c) => (
                      <label key={c.id} className="flex cursor-pointer items-start gap-2 text-sm">
                        <input
                          type="radio"
                          name={`screen-${s.id}`}
                          value={c.id}
                          checked={screenPick[s.id] === c.id}
                          onChange={() => setScreenPick((p) => ({ ...p, [s.id]: c.id }))}
                          disabled={busy}
                          data-self-check-choice={c.id}
                        />
                        <span>{c.text}</span>
                      </label>
                    ))}
                  </div>
                </fieldset>
                <div className="mt-2">
                  <button
                    type="button"
                    className={btn}
                    onClick={() => void checkScreen(s.id)}
                    disabled={busy || !screenPick[s.id]}
                    data-self-check-screen-check={s.id}
                  >
                    Check
                  </button>
                </div>
                {r && (
                  <div
                    role="status"
                    className={`mt-2 rounded-md border p-2 text-sm ${r.correct ? 'border-emerald-400 bg-emerald-50 text-emerald-950 dark:border-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-100' : 'border-amber-400 bg-amber-50 text-amber-950 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-100'}`}
                    data-self-check-screen-result={r.correct ? 'correct' : 'incorrect'}
                  >
                    <p className="font-semibold">
                      {r.correct ? 'Correct.' : `Not quite. The simulator's answer: ${r.expected}`}
                    </p>
                    <p className="mt-1">{r.explanation}</p>
                    {r.assumptions.length > 0 && (
                      <p className="mt-1 text-xs">
                        Rests on {r.assumptions.map((a) => `assumption ${a.number} (${a.text})`).join('; ')}
                      </p>
                    )}
                  </div>
                )}
              </li>
            )
          })}
        </ol>
      </section>
      {error && (
        <div className="mt-4">
          <ErrorPanel title="That did not work" error={error} />
        </div>
      )}
    </>
  )
}

/** Personal checklist, ticked by hand and stored in this browser only. */
function Checklist() {
  const [ticks, setTicks] = useState<Record<string, boolean>>(() => loadChecklist())
  const toggle = (id: string) =>
    setTicks((t) => {
      const next = { ...t, [id]: !t[id] }
      saveChecklist(next)
      return next
    })
  const all = GOALS.flatMap((g) => [...g.explain, ...g.find])
  const done = all.filter((i) => ticks[i.id]).length
  return (
    <section aria-labelledby="sc-checklist" className="mt-8" data-self-check-checklist>
      <h2 id="sc-checklist" className="text-lg font-semibold">
        Part 3: your checklist
      </h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-300" data-checklist-count>
        {done} of {all.length} ticked. Be honest with yourself; only you see this, and it stays in this browser.
      </p>
      <div className="mt-3 grid gap-4 md:grid-cols-2">
        {GOALS.map((g) => (
          <fieldset
            key={g.id}
            className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
            data-checklist-goal={g.id}
          >
            <legend className="px-1 text-sm font-semibold">{g.title}</legend>
            <p className="text-xs font-medium uppercase tracking-wide text-slate-500">I can explain…</p>
            <ul className="mt-1 space-y-1.5">
              {g.explain.map((i) => (
                <li key={i.id}>
                  <label className="flex cursor-pointer items-start gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={!!ticks[i.id]}
                      onChange={() => toggle(i.id)}
                      data-checklist-item={i.id}
                    />
                    <span>{i.text}</span>
                  </label>
                </li>
              ))}
            </ul>
            <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-500">I can find in the VCO…</p>
            <ul className="mt-1 space-y-1.5">
              {g.find.map((i) => (
                <li key={i.id}>
                  <label className="flex cursor-pointer items-start gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={!!ticks[i.id]}
                      onChange={() => toggle(i.id)}
                      data-checklist-item={i.id}
                    />
                    <span>{i.text}</span>
                  </label>
                </li>
              ))}
            </ul>
          </fieldset>
        ))}
      </div>
      <p className="mt-3 text-sm">
        <Link to="/progress" className="underline">
          See your goal coverage on the dashboard
        </Link>
      </p>
    </section>
  )
}
