import { Link } from 'react-router-dom'
import type { Progress, TopicMastery } from '../api'
import { GOALS, goalCoverage, loadChecklist, loadSelfCheck } from './goals'

const pct = (n: number | null) => (n == null ? '—' : `${Math.round(n)}%`)

/**
 * Goal coverage: how far each of the goals of the course is covered, from lesson progress, quiz scores, weak topics,
 * labs and the last self-check (kept in this browser only). Informational: nothing reads it to restrict anything, and it is
 * not a grade.
 */
export default function GoalCoverageView({ progress, topics }: { progress: Progress; topics: TopicMastery[] | null }) {
  const memory = loadSelfCheck()
  const checklist = loadChecklist()
  const rows = GOALS.map((g) => goalCoverage(g, progress, topics, memory, checklist))
  return (
    <section aria-labelledby="goals-h" className="mt-6" data-goal-coverage>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="goals-h" className="text-lg font-semibold">
          Goal coverage
        </h2>
        <Link to="/sim/self-check" className="text-sm underline" data-goal-self-check>
          Take the self-check
        </Link>
      </div>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        How far the course's goals are covered, from your lesson progress, quiz results, labs and last self-check.
        Informational only: it is not a grade and nothing is locked by it.
        {memory
          ? ` Last self-check: ${new Date(memory.at).toLocaleDateString()} (kept in this browser only).`
          : ' No self-check yet.'}
      </p>
      <ul className="mt-3 grid gap-3 md:grid-cols-2">
        {rows.map((r) => (
          <li
            key={r.goal.id}
            className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
            data-goal={r.goal.id}
          >
            <div className="flex items-baseline justify-between gap-2">
              <h3 className="font-medium">{r.goal.title}</h3>
              <span className="text-xl font-semibold" data-goal-overall>
                {pct(r.overall)}
              </span>
            </div>
            <div
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(r.overall ?? 0)}
              aria-label={`${r.goal.title}: goal coverage`}
              className="mt-2 h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
            >
              <div className="h-full bg-violet-500" style={{ width: `${r.overall ?? 0}%` }} />
            </div>
            <p className="mt-2 text-xs text-slate-600 dark:text-slate-400">{r.goal.blurb}</p>
            <dl className="mt-2 grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs">
              <dt>Lessons completed</dt>
              <dd data-goal-part="lessons">{pct(r.lessons)}</dd>
              <dt>Best quiz scores</dt>
              <dd data-goal-part="quiz">{pct(r.quiz)}</dd>
              <dt>Labs completed</dt>
              <dd data-goal-part="labs">{pct(r.labs)}</dd>
              <dt>Last self-check</dt>
              <dd data-goal-part="self-check">{pct(r.selfCheck)}</dd>
              <dt>Checklist ticked</dt>
              <dd data-goal-part="checklist">
                {r.checklist.ticked} of {r.checklist.total}
              </dd>
            </dl>
            {r.weakTopics.length > 0 && (
              <p className="mt-2 text-xs text-rose-800 dark:text-rose-300" data-goal-weak>
                Weak topics: {r.weakTopics.join(', ')}
              </p>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}
