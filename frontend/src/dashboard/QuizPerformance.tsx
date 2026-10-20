import type { Activity, Course, SectionProgress } from '../api'
import { pct } from '../lib/course'
import { sparkPoints } from '../studyguide/derive'

/** Best quiz score per section; the last 10 attempts show as a sparkline on hover or keyboard focus. */
export default function QuizPerformance({
  course,
  progress,
  quiz,
}: {
  course: Course
  progress: SectionProgress[]
  quiz: Activity['quiz']
}) {
  return (
    <section
      aria-labelledby="quiz-h"
      className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
      data-quiz-performance
    >
      <h2 id="quiz-h" className="text-lg font-semibold">
        Quiz performance
      </h2>
      <ul className="mt-2 space-y-1">
        {course.sections
          .filter((s) => s.question_count > 0)
          .map((s) => {
            const best = progress.find((p) => p.section_id === s.id)?.best_score_pct ?? null
            const attempts = quiz.find((q) => q.section_id === s.id)?.attempts ?? []
            const scores = attempts.map((a) => (a.total > 0 ? (100 * a.score) / a.total : 0))
            return (
              <li key={s.id} data-quiz-section={s.id} className="group relative">
                <div
                  tabIndex={0}
                  aria-label={`${s.title}: best ${pct(best)}${scores.length ? `, last ${scores.length} attempts ${scores.map((x) => Math.round(x) + '%').join(', ')}` : ', no attempts yet'}`}
                  className="flex items-center gap-3 rounded-md px-2 py-1.5 text-sm hover:bg-slate-100 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:hover:bg-slate-800"
                >
                  <span className="min-w-0 flex-1 truncate">{s.title}</span>
                  <span
                    className="h-1.5 w-20 shrink-0 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
                    aria-hidden
                  >
                    <span className="block h-full bg-sky-500" style={{ width: `${best ?? 0}%` }} />
                  </span>
                  <span className="w-10 shrink-0 text-right font-medium" data-quiz-best>
                    {pct(best)}
                  </span>
                </div>
                <div
                  aria-hidden
                  className="pointer-events-none invisible absolute right-2 bottom-full z-20 mb-1 w-44 rounded-md border border-slate-300 bg-white p-2 text-xs shadow-lg group-focus-within:visible group-hover:visible dark:border-slate-600 dark:bg-slate-800"
                >
                  {scores.length === 0 ? (
                    <span>No attempts yet</span>
                  ) : (
                    <>
                      <span>
                        Last {scores.length} attempt{scores.length === 1 ? '' : 's'}
                      </span>
                      <svg
                        width={160}
                        height={44}
                        viewBox="0 0 160 44"
                        className="mt-1 block"
                        data-sparkline={scores.length}
                      >
                        <polyline
                          points={sparkPoints(scores, 160, 44)}
                          fill="none"
                          strokeWidth={2}
                          className="stroke-sky-600 dark:stroke-sky-400"
                        />
                        {sparkPoints(scores, 160, 44)
                          .split(' ')
                          .map((pt) => {
                            const [x, y] = pt.split(',')
                            return <circle key={pt} cx={x} cy={y} r={2.5} className="fill-sky-600 dark:fill-sky-400" />
                          })}
                      </svg>
                    </>
                  )}
                </div>
              </li>
            )
          })}
      </ul>
    </section>
  )
}
