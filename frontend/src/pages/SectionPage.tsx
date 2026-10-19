import { Link, useParams } from 'react-router-dom'
import ErrorPanel from '../components/ErrorPanel'
import { useCourseState } from '../course/context'
import { formatDateTime, pct } from '../lib/course'
import { useProgress } from '../progress/context'
import NotFound from './NotFound'

export default function SectionPage() {
  const state = useCourseState()
  const { sectionId } = useParams()
  const { progress } = useProgress()

  if (state.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (state.status === 'error') return <ErrorPanel title="Course content is unavailable" error={state.error} />
  const section = state.course.sections.find((s) => s.id === sectionId)
  if (!section) return <NotFound />
  const sp = progress?.sections.find((x) => x.section_id === section.id)

  return (
    <div className="max-w-3xl">
      {section.placeholder && (
        <p className="mb-3 inline-block rounded bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-900 dark:bg-amber-900/40 dark:text-amber-100">
          PLACEHOLDER SECTION
        </p>
      )}
      <h1 className="text-2xl font-semibold">{section.title}</h1>
      <p className="mt-2 text-slate-600 dark:text-slate-400">{section.summary}</p>
      <p className="mt-1 text-sm text-slate-500">
        About {section.estimated_minutes} min · {section.lessons.length} lessons
      </p>

      <ol className="mt-6 space-y-3">
        {section.lessons.map((l, i) => (
          <li key={l.id}>
            <Link
              to={`/course/${section.id}/${l.id}`}
              className="block rounded-lg border border-slate-200 p-4 hover:border-sky-400 dark:border-slate-800 dark:hover:border-sky-600"
            >
              <div className="font-medium">
                {i + 1}. {l.title}
              </div>
              <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{l.summary}</div>
              <div className="mt-1 text-xs text-slate-500">{l.estimated_minutes} min</div>
            </Link>
          </li>
        ))}
      </ol>

      {section.question_count > 0 && (
        <section
          aria-labelledby="section-quiz"
          className="mt-8 rounded-lg border border-sky-200 bg-sky-50 p-5 dark:border-sky-900 dark:bg-sky-950/30"
        >
          <h2 id="section-quiz" className="font-semibold">
            Section quiz
          </h2>
          <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
            {section.question_count} question{section.question_count === 1 ? '' : 's'}.
            {sp && sp.quiz_attempts > 0
              ? ` Best score ${pct(sp.best_score_pct)}, latest ${pct(sp.latest_score_pct)} over ${sp.quiz_attempts} attempt${sp.quiz_attempts === 1 ? '' : 's'}.`
              : sp?.quiz_skipped && sp.quiz_skipped_at
                ? ` Skipped on ${formatDateTime(sp.quiz_skipped_at)}; you can take it any time.`
                : ' Not taken yet.'}
          </p>
          <Link
            to={`/course/${section.id}/quiz`}
            className="mt-3 inline-block rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950"
          >
            {sp && sp.quiz_attempts > 0 ? 'Retake the quiz' : 'Take the quiz'}
          </Link>
        </section>
      )}
    </div>
  )
}
