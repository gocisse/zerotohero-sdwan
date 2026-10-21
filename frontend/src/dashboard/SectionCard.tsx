import type { GuideSection } from '../api'
import AddedBadge from '../studyguide/AddedBadge'
import { pct } from '../lib/course'
import { sectionStats } from '../studyguide/derive'
import TopicRow from './TopicRow'

/** A study guide section: topics covered by level, lessons, quiz average and labs. Clicking expands its topics. */
export default function SectionCard({
  section: s,
  open,
  onToggle,
  topology,
}: {
  section: GuideSection
  open: boolean
  onToggle: () => void
  topology: string | null
}) {
  const st = sectionStats(s)
  const lessonFraction = st.lessonsTotal === 0 ? 0 : st.lessonsCompleted / st.lessonsTotal
  return (
    <section
      id={`section-${s.id}`}
      data-guide-section={s.id}
      className="scroll-mt-4 rounded-lg border border-slate-200 dark:border-slate-800"
    >
      <h3>
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={open}
          aria-controls={`topics-${s.id}`}
          className="flex w-full flex-col gap-2 rounded-lg p-4 text-left hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:hover:bg-slate-900"
        >
          <span className="flex items-center gap-2 font-semibold">
            <span aria-hidden className={`text-xs transition-transform ${open ? 'rotate-90' : ''}`}>
              ▶
            </span>
            <span className="rounded bg-slate-200 px-1.5 py-0.5 text-xs dark:bg-slate-700">{s.id}</span>
            <span className="flex-1">{s.title}</span>
            {s.added_by_course && <AddedBadge />}
          </span>
          <span className="text-sm text-slate-700 dark:text-slate-300" data-section-summary>
            {st.covered === 0
              ? `Not covered in this course yet (${st.topics} topics)`
              : `${st.done} of ${st.covered} covered topics done`}
            {st.byCoverage.none > 0 && st.covered > 0 && ` · ${st.byCoverage.none} not yet in the course`}
          </span>
          <span className="text-xs text-slate-600 dark:text-slate-400" data-section-mix>
            {st.byCoverage.full} full · {st.byCoverage.partial} partial · {st.byCoverage.none} not covered
          </span>
          <span
            role="progressbar"
            aria-label={`${s.id} lessons completed`}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(lessonFraction * 100)}
            className="block h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
          >
            <span className="block h-full bg-emerald-500" style={{ width: `${lessonFraction * 100}%` }} />
          </span>
          <span className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-600 dark:text-slate-400">
            <span data-section-lessons>
              Lessons {st.lessonsCompleted}/{st.lessonsTotal}
            </span>
            <span data-section-quiz>Avg best quiz {pct(st.avgBestQuizPct)}</span>
            <span data-section-labs>
              Labs {st.labsCompleted}/{st.labsTotal}
            </span>
          </span>
        </button>
      </h3>
      {open && (
        <ul id={`topics-${s.id}`} className="space-y-2 px-4 pb-4">
          {s.topics.map((t) => (
            <TopicRow key={t.id} topic={t} topology={topology} />
          ))}
        </ul>
      )}
    </section>
  )
}
