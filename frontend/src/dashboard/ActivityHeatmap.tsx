import type { Activity } from '../api'
import { HEATMAP_WEEKS, heatmapWeeks, intensity, sessionsThisWeek } from '../studyguide/derive'

const level = [
  'fill-slate-200 dark:fill-slate-800',
  'fill-emerald-200 dark:fill-emerald-900',
  'fill-emerald-400 dark:fill-emerald-700',
  'fill-emerald-600 dark:fill-emerald-500',
  'fill-emerald-800 dark:fill-emerald-300',
]
const words: Record<string, [string, string]> = {
  lesson_completed: ['lesson completed', 'lessons completed'],
  lesson_visited: ['lesson opened', 'lessons opened'],
  quiz_attempt: ['quiz attempt', 'quiz attempts'],
  review_answer: ['review answer', 'review answers'],
  lab_attempt: ['lab attempt', 'lab attempts'],
  lab_completed: ['lab completed', 'labs completed'],
}
const describe = (counts: Record<string, number>) =>
  Object.entries(counts)
    .map(([k, n]) => `${n} ${words[k]?.[n === 1 ? 0 : 1] ?? k}`)
    .join(', ')

const CELL = 18
const GAP = 4
const LEFT = 30

/** A 12-week activity heatmap and a sessions count, both derived from stored timestamps. */
export default function ActivityHeatmap({ activity, now }: { activity: Activity; now: Date }) {
  const weeks = heatmapWeeks(activity.events, now)
  const activeDays = weeks.flat().filter((d) => d.total > 0).length
  const sessions = sessionsThisWeek(activity.events, now)
  const width = LEFT + HEATMAP_WEEKS * (CELL + GAP)
  const height = 7 * (CELL + GAP)
  const dayNames = ['Mon', '', 'Wed', '', 'Fri', '', '']
  const first = weeks[0]![0]!.date
  const last = weeks[HEATMAP_WEEKS - 1]![6]!.date
  return (
    <section
      aria-labelledby="activity-h"
      className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
      data-activity
    >
      <h2 id="activity-h" className="text-lg font-semibold">
        Activity
      </h2>
      <p className="mt-1 text-sm text-slate-700 dark:text-slate-300" data-sessions>
        {sessions} session{sessions === 1 ? '' : 's'} this week · active on {activeDays} of the last {HEATMAP_WEEKS * 7}{' '}
        days
      </p>
      <div className="mt-3 overflow-x-auto">
        <svg
          width={width}
          height={height}
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label={`Activity per day from ${first} to ${last}: ${activeDays} active days`}
          className="max-w-full"
        >
          {dayNames.map((n, i) => (
            <text
              key={i}
              x={0}
              y={i * (CELL + GAP) + CELL - 4}
              className="fill-slate-500 text-[10px] dark:fill-slate-400"
            >
              {n}
            </text>
          ))}
          {weeks.map((w, wi) =>
            w.map((d, di) => (
              <rect
                key={d.date}
                x={LEFT + wi * (CELL + GAP)}
                y={di * (CELL + GAP)}
                width={CELL}
                height={CELL}
                rx={3}
                data-heatmap-day={d.date}
                data-count={d.total}
                className={
                  d.future ? 'fill-transparent stroke-slate-200 dark:stroke-slate-800' : level[intensity(d.total)]
                }
              >
                <title>
                  {d.future ? `${d.date}: later` : `${d.date}: ${d.total === 0 ? 'no activity' : describe(d.counts)}`}
                </title>
              </rect>
            )),
          )}
        </svg>
      </div>
      <p className="mt-2 flex items-center gap-1 text-xs text-slate-600 dark:text-slate-400" aria-hidden>
        Less
        <svg width={5 * 14} height={12}>
          {level.map((c, i) => (
            <rect key={i} x={i * 14} width={12} height={12} rx={2} className={c} />
          ))}
        </svg>
        More
      </p>
      {activeDays === 0 && (
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400" data-activity-empty>
          No activity yet. This fills in as you open lessons, take quizzes and try labs.
        </p>
      )}
      <ul className="mt-2 list-disc space-y-0.5 pl-5 text-xs text-slate-600 dark:text-slate-400" data-activity-limits>
        <li>A session is activity with no gap longer than 30 minutes.</li>
        {activity.limits.map((l) => (
          <li key={l}>{l}</li>
        ))}
      </ul>
    </section>
  )
}
