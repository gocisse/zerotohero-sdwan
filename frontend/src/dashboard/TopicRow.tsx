import { Link } from 'react-router-dom'
import type { GuideTopic, LessonStatus } from '../api'
import { pct } from '../lib/course'
import CoverageBadge from './CoverageBadge'

const icon: Record<LessonStatus, { glyph: string; label: string; cls: string }> = {
  not_started: { glyph: '○', label: 'not started', cls: 'text-slate-400 dark:text-slate-500' },
  visited: { glyph: '◐', label: 'in progress', cls: 'text-sky-600 dark:text-sky-400' },
  completed: { glyph: '✓', label: 'completed', cls: 'text-emerald-600 dark:text-emerald-400' },
}

const labStatus = (s: string) => (s === 'not_started' ? 'not started' : s.replace('_', ' '))
const linkCls = 'text-sky-700 underline dark:text-sky-300'

/** One study guide topic: coverage, lesson status icons, quiz score, lab status and links into the course. */
export default function TopicRow({ topic: t, topology }: { topic: GuideTopic; topology: string | null }) {
  const p = t.progress
  const uncovered = t.coverage === 'none'
  const sim = (path: string) => path.replace('{topology}', topology ?? '')
  return (
    <li
      id={`topic-${t.id}`}
      data-guide-topic={t.id}
      data-coverage={t.coverage}
      className="scroll-mt-4 rounded-md border border-slate-200 p-3 dark:border-slate-800"
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="min-w-0 flex-1 font-medium">{t.title}</span>
        <span className="text-xs text-slate-500 dark:text-slate-400">{t.kind}</span>
        <CoverageBadge coverage={t.coverage} note={t.note} />
      </div>

      {uncovered ? (
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400" data-not-covered>
          Not covered in this course yet. <span className="text-xs">{t.note}</span>
        </p>
      ) : (
        <div className="mt-2 space-y-1.5 text-sm">
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span className="text-slate-600 dark:text-slate-400">
              Lessons {p.lessons_completed}/{p.lessons_total}
            </span>
            {t.lessons.map((l) => (
              <Link
                key={l.id}
                to={`/course/${l.section_id}/${l.id}`}
                title={`${l.title} (${icon[l.status].label})`}
                data-lesson-status={l.status}
                className="inline-flex items-center gap-1 rounded border border-slate-200 px-1.5 py-0.5 text-xs hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
              >
                <span aria-hidden className={icon[l.status].cls}>
                  {icon[l.status].glyph}
                </span>
                <span>{l.title}</span>
                <span className="sr-only">: {icon[l.status].label}</span>
              </Link>
            ))}
          </p>
          <p className="text-slate-600 dark:text-slate-400" data-topic-quiz>
            {p.best_section_score_pct == null && p.topic_accuracy_pct == null
              ? 'Quiz: not taken yet'
              : `Quiz: ${p.best_section_score_pct == null ? '' : `best section score ${pct(p.best_section_score_pct)}`}${p.best_section_score_pct != null && p.topic_accuracy_pct != null ? ' · ' : ''}${p.topic_accuracy_pct == null ? '' : `topic accuracy ${pct(p.topic_accuracy_pct)} over ${p.quiz_answers} answers`}`}
            {p.review_due > 0 && (
              <>
                {' · '}
                <Link to="/review" className={linkCls}>
                  {p.review_due} review due
                </Link>
              </>
            )}
          </p>
          {t.labs.length > 0 && (
            <p className="text-slate-600 dark:text-slate-400" data-topic-labs>
              Labs:{' '}
              {t.labs.map((l, i) => (
                <span key={l.id}>
                  {i > 0 && ', '}
                  <Link to={`/sim/labs/${l.id}`} className={linkCls}>
                    {l.title}
                  </Link>{' '}
                  ({labStatus(l.status)})
                </span>
              ))}
            </p>
          )}
          {(t.diagrams.length > 0 || t.sim_screens.length > 0) && (
            <p className="flex flex-wrap gap-x-3 gap-y-1 text-slate-600 dark:text-slate-400">
              {t.diagrams.length > 0 && t.lessons[0] && (
                <span>
                  Diagrams:{' '}
                  {t.diagrams.map((d, i) => (
                    <span key={d.id}>
                      {i > 0 && ', '}
                      <Link
                        to={`/course/${t.lessons[0]!.section_id}/${t.lessons[0]!.id}#diagram-${d.id}`}
                        className={linkCls}
                      >
                        {d.title}
                      </Link>
                    </span>
                  ))}
                </span>
              )}
              {t.sim_screens.length > 0 && (
                <span>
                  Simulator:{' '}
                  {t.sim_screens.map((s, i) => (
                    <span key={s.id}>
                      {i > 0 && ', '}
                      {s.path.includes('{topology}') && !topology ? (
                        s.label
                      ) : (
                        <Link to={sim(s.path)} className={linkCls}>
                          {s.label}
                        </Link>
                      )}
                    </span>
                  ))}
                </span>
              )}
            </p>
          )}
        </div>
      )}
    </li>
  )
}
