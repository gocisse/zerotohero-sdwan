import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchTopics, type TopicMastery } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useCourseState } from '../course/context'
import { formatDateTime, pct } from '../lib/course'
import { useProgress } from '../progress/context'
import GoalCoverageView from '../progress/GoalCoverageView'

export default function ProgressPage() {
  const course = useCourseState()
  const { progress, review, refresh } = useProgress()
  const [topics, setTopics] = useState<TopicMastery[] | null>(null)
  const [topicsError, setTopicsError] = useState<Error | null>(null)

  useEffect(() => {
    let cancelled = false
    void refresh()
    fetchTopics()
      .then((t) => !cancelled && setTopics(t.topics))
      .catch((e: Error) => !cancelled && setTopicsError(e))
    return () => {
      cancelled = true
    }
  }, [refresh])

  if (course.status === 'error') return <ErrorPanel title="Course content is unavailable" error={course.error} />
  if (course.status === 'loading' || !progress) return <p className="text-slate-500">Loading…</p>

  const titleOf = (id: string) => course.course.sections.find((s) => s.id === id)?.title ?? id
  const weak = (topics ?? []).filter((t) => t.weak)

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-semibold">Your progress</h1>

      <section
        aria-label="Overall progress"
        className="mt-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
      >
        <div className="flex items-baseline justify-between">
          <span className="font-medium">Overall</span>
          <span className="text-2xl font-semibold" data-testid="overall-pct">
            {pct(progress.overall_percentage)}
          </span>
        </div>
        <div
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(progress.overall_percentage)}
          aria-label="Lessons completed"
          className="mt-2 h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
        >
          <div className="h-full bg-emerald-500" style={{ width: `${progress.overall_percentage}%` }} />
        </div>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          {progress.lessons_completed} of {progress.lessons_total} lessons completed
        </p>
      </section>

      <GoalCoverageView progress={progress} topics={topics} />

      <section
        aria-label="Labs"
        className="mt-6 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
        data-labs-block
      >
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h2 className="font-medium">Labs</h2>
          <Link to="/sim/labs" className="text-sm underline">
            All labs
          </Link>
        </div>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400" data-testid="labs-summary">
          {progress.labs.completed} of {progress.labs.total} completed · {progress.labs.started} in progress ·{' '}
          {progress.labs.skipped} skipped. Labs are optional.
        </p>
        <ul className="mt-2 space-y-1 text-sm">
          {progress.labs.items.map((l) => (
            <li key={l.lab_id} data-lab-progress={l.lab_id} data-lab-status={l.status}>
              <Link to={`/sim/labs/${l.lab_id}`} className="underline">
                {l.title}
              </Link>
              : {l.status.replace('_', ' ')}
              {l.attempts > 0 ? ` · ${l.attempts} checks` : ''}
              {l.hints_used > 0 ? ` · ${l.hints_used} hints` : ''}
            </li>
          ))}
        </ul>
      </section>

      <section
        aria-label="Review questions due"
        className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-300 bg-amber-50 p-5 dark:border-amber-800 dark:bg-amber-950/30"
      >
        <div>
          <div className="font-medium">Review questions</div>
          <p className="text-sm text-slate-700 dark:text-slate-300" data-testid="due-count">
            {review
              ? review.due_count > 0
                ? `${review.due_count} due now`
                : review.next_due_at
                  ? `Nothing due. Next review ${formatDateTime(review.next_due_at)}.`
                  : 'Nothing due yet. Questions you answer in quizzes show up here.'
              : '…'}
          </p>
        </div>
        <Link
          to="/review"
          className="rounded-md bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950"
        >
          {review && review.due_count > 0 ? 'Start review' : 'Open review'}
        </Link>
      </section>

      <section aria-labelledby="sections-h" className="mt-6">
        <h2 id="sections-h" className="text-lg font-semibold">
          Sections
        </h2>
        <div className="mt-2 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-800">
          <table className="w-full border-collapse text-left text-sm">
            <caption className="sr-only">Progress per section</caption>
            <thead className="bg-slate-100 dark:bg-slate-900">
              <tr>
                {['Section', 'Lessons', 'Best', 'Latest', 'Attempts', 'Quiz'].map((h) => (
                  <th key={h} scope="col" className="px-3 py-2 font-semibold">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {progress.sections.map((s) => (
                <tr
                  key={s.section_id}
                  className="border-t border-slate-200 dark:border-slate-800"
                  data-section={s.section_id}
                >
                  <th scope="row" className="px-3 py-2 font-medium">
                    <Link className="text-sky-700 hover:underline dark:text-sky-300" to={`/course/${s.section_id}`}>
                      {titleOf(s.section_id)}
                    </Link>
                  </th>
                  <td className="px-3 py-2">
                    {s.lessons_completed}/{s.lessons_total}
                  </td>
                  <td className="px-3 py-2" data-col="best">
                    {pct(s.best_score_pct)}
                  </td>
                  <td className="px-3 py-2" data-col="latest">
                    {pct(s.latest_score_pct)}
                  </td>
                  <td className="px-3 py-2" data-col="attempts">
                    {s.quiz_attempts}
                  </td>
                  <td className="px-3 py-2" data-col="quiz">
                    {s.quiz_attempts > 0 ? 'Taken' : s.quiz_skipped ? 'Skipped' : 'Not taken'}
                    {' · '}
                    <Link
                      className="text-sky-700 hover:underline dark:text-sky-300"
                      to={`/course/${s.section_id}/quiz`}
                    >
                      {s.quiz_attempts > 0 ? 'retake' : 'take now'}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section aria-labelledby="weak-h" className="mt-6">
        <h2 id="weak-h" className="text-lg font-semibold">
          Topics worth revisiting
        </h2>
        {topicsError && <ErrorPanel title="Could not load topics" error={topicsError} />}
        {topics && weak.length === 0 && (
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
            {topics.length === 0
              ? 'Take a quiz and your topic strengths will appear here.'
              : 'No weak topics right now. Nice!'}
          </p>
        )}
        <ul className="mt-2 space-y-3">
          {weak.map((t) => (
            <li
              key={t.topic}
              data-weak-topic={t.topic}
              className="rounded-lg border border-rose-200 bg-rose-50 p-4 dark:border-rose-900 dark:bg-rose-950/30"
            >
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium">{t.topic}</span>
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {pct(t.accuracy_pct)} accuracy over {t.answers} answers
                </span>
              </div>
              {t.lessons.length > 0 && (
                <p className="mt-1 text-sm">
                  Review:{' '}
                  {t.lessons.map((l, i) => (
                    <span key={l.id}>
                      {i > 0 && ', '}
                      <Link className="text-sky-700 underline dark:text-sky-300" to={`/course/${l.section_id}/${l.id}`}>
                        {l.title}
                      </Link>
                    </span>
                  ))}
                </p>
              )}
            </li>
          ))}
        </ul>
        {topics && topics.length > weak.length && (
          <details className="mt-3 text-sm">
            <summary className="cursor-pointer text-slate-600 dark:text-slate-400">All topics</summary>
            <ul className="mt-2 space-y-1">
              {topics.map((t) => (
                <li key={t.topic}>
                  {t.topic}: {pct(t.accuracy_pct)} ({t.answers} answers)
                </li>
              ))}
            </ul>
          </details>
        )}
      </section>
    </div>
  )
}
