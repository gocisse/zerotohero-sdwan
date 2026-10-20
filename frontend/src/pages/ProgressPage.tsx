import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { fetchActivity, fetchTopics, fetchTopologies, type Activity, type StudyGuide, type TopicMastery } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useCourseState } from '../course/context'
import ActivityHeatmap from '../dashboard/ActivityHeatmap'
import ProgressDetails from '../dashboard/ProgressDetails'
import QuizPerformance from '../dashboard/QuizPerformance'
import Ring from '../dashboard/Ring'
import SectionCard from '../dashboard/SectionCard'
import WeakTopics from '../dashboard/WeakTopics'
import { useProgress } from '../progress/context'
import GoalCoverageView from '../progress/GoalCoverageView'
import { useStudyGuide } from '../studyguide/context'
import { guideLessonOrder, nextRecommended, overallStats } from '../studyguide/derive'

const card = 'rounded-lg border border-slate-200 p-4 dark:border-slate-800'
const linkBtn =
  'inline-block rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950'

export default function ProgressPage() {
  const course = useCourseState()
  const { progress, review, refresh } = useProgress()
  const sg = useStudyGuide()
  const { hash } = useLocation()
  const [topics, setTopics] = useState<TopicMastery[] | null>(null)
  const [topicsError, setTopicsError] = useState<Error | null>(null)
  const [activity, setActivity] = useState<Activity | null>(null)
  const [activityError, setActivityError] = useState<Error | null>(null)
  const [topology, setTopology] = useState<string | null>(null)
  const [openSections, setOpenSections] = useState<Set<string>>(new Set())
  const [now] = useState(() => new Date())

  const guide: StudyGuide | null = sg.status === 'ready' ? sg.guide : null
  const refreshGuide = sg.status === 'loading' ? null : sg.refresh

  useEffect(() => {
    let cancelled = false
    void refresh()
    void refreshGuide?.()
    fetchTopics()
      .then((t) => !cancelled && setTopics(t.topics))
      .catch((e: Error) => !cancelled && setTopicsError(e))
    fetchActivity()
      .then((a) => !cancelled && setActivity(a))
      .catch((e: Error) => !cancelled && setActivityError(e))
    fetchTopologies()
      .then((t) => !cancelled && setTopology(t[0]?.id ?? null))
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
    // refreshGuide is stable per provider state; the data is re-fetched each time the page opens.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [refresh])

  // #topic-<id> and #section-<id> open the guide section that holds the target.
  const [seenHash, setSeenHash] = useState('')
  if (guide && hash !== seenHash) {
    setSeenHash(hash)
    const id = hash.replace(/^#(topic|section)-/, '')
    const owner = guide.sections.find((s) => s.id === id || s.topics.some((t) => t.id === id))
    if (owner && /^#(topic|section)-/.test(hash) && !openSections.has(owner.id))
      setOpenSections(new Set(openSections).add(owner.id))
  }
  useEffect(() => {
    if (!guide || !hash) return
    const raf = requestAnimationFrame(() => document.getElementById(hash.slice(1))?.scrollIntoView({ block: 'start' }))
    return () => cancelAnimationFrame(raf)
  }, [guide, hash, seenHash])

  const stats = useMemo(() => (guide ? overallStats(guide) : null), [guide])
  const first = useMemo(
    () => (guide ? (guideLessonOrder(guide).find((x) => x.guideSection !== null)?.lesson ?? null) : null),
    [guide],
  )
  const next = useMemo(() => (guide ? nextRecommended(guide) : null), [guide])

  if (course.status === 'error') return <ErrorPanel title="Course content is unavailable" error={course.error} />
  if (course.status === 'loading' || !progress) return <p className="text-slate-500">Loading…</p>

  const fresh = progress.lessons_completed === 0 && !activity?.last_lesson
  const last = activity?.last_lesson ?? null

  return (
    <div className="max-w-5xl space-y-6" data-dashboard>
      <h1 className="text-2xl font-semibold">Your study dashboard</h1>

      <section aria-label="Where to go next" className="grid gap-3 md:grid-cols-3" data-dashboard-header>
        <div className={card} data-continue>
          <h2 className="text-sm font-medium text-slate-600 dark:text-slate-400">
            {fresh ? 'Begin here' : 'Continue where you left off'}
          </h2>
          {last ? (
            <>
              <p className="mt-1 font-medium">{last.title}</p>
              <p className="mb-3 text-sm text-slate-600 dark:text-slate-400">
                {last.status === 'completed' ? 'Completed' : 'In progress'}
              </p>
              <Link to={`/course/${last.section_id}/${last.lesson_id}`} className={linkBtn}>
                Open lesson
              </Link>
            </>
          ) : first ? (
            <>
              <p className="mt-1 font-medium">{first.title}</p>
              <p className="mb-3 text-sm text-slate-600 dark:text-slate-400">
                The first lesson of the study guide. Lessons mark themselves as you read.
              </p>
              <Link to={`/course/${first.section_id}/${first.id}`} className={linkBtn}>
                Start the first lesson
              </Link>
            </>
          ) : (
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">Open any lesson from the menu to begin.</p>
          )}
        </div>

        <div className={card} data-review-card>
          <h2 className="text-sm font-medium text-slate-600 dark:text-slate-400">Review questions</h2>
          <p className="mt-1 font-medium" data-dashboard-due>
            {review ? (review.due_count > 0 ? `${review.due_count} due now` : 'Nothing due') : '…'}
          </p>
          <p className="mb-3 text-sm text-slate-600 dark:text-slate-400">
            {review && review.due_count === 0 && !review.next_due_at
              ? 'Questions you answer in quizzes show up here.'
              : 'A short spaced-repetition session.'}
          </p>
          <Link to="/review" className={linkBtn}>
            {review && review.due_count > 0 ? 'Start review' : 'Open review'}
          </Link>
        </div>

        <div className={card} data-next-step>
          <h2 className="text-sm font-medium text-slate-600 dark:text-slate-400">Next in the study guide</h2>
          {next ? (
            <>
              <p className="mt-1 font-medium">{next.title}</p>
              <p className="mb-3 text-sm text-slate-600 dark:text-slate-400">
                The first lesson in guide order that is not completed. A suggestion only.
              </p>
              <Link to={`/course/${next.section_id}/${next.id}`} className={linkBtn}>
                Open lesson
              </Link>
            </>
          ) : guide ? (
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              Every lesson in the study guide is completed. Try a lab or the self-check.
            </p>
          ) : (
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">The study guide is unavailable right now.</p>
          )}
        </div>
      </section>

      <section aria-labelledby="guide-h" data-guide-progress>
        <h2 id="guide-h" className="text-lg font-semibold">
          Study guide progress
        </h2>
        {!guide || !stats ? (
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
            {sg.status === 'loading'
              ? 'Loading the study guide…'
              : 'The study guide could not be loaded. The rest of this page still works.'}
          </p>
        ) : (
          <>
            <div className={`${card} mt-2 flex flex-wrap items-center gap-5`}>
              <Ring
                fraction={stats.fraction}
                center={`${stats.done}/${stats.covered}`}
                label={`${stats.done} of ${stats.covered} covered topics done`}
              />
              <div className="min-w-0 flex-1" data-overall>
                <p className="font-medium" data-overall-line>
                  {stats.done} of {stats.covered} covered topics done
                </p>
                <p className="text-sm text-slate-600 dark:text-slate-400" data-overall-uncovered>
                  {stats.notCovered} topic{stats.notCovered === 1 ? '' : 's'} not yet in the course
                </p>
                <p className="mt-1 text-xs text-slate-600 dark:text-slate-400">
                  A topic is done when all its lessons are completed. Topics the course does not cover are listed but
                  never counted against you.
                </p>
              </div>
            </div>
            <div className="mt-3 space-y-3">
              {guide.sections.map((s) => (
                <SectionCard
                  key={s.id}
                  section={s}
                  topology={topology}
                  open={openSections.has(s.id)}
                  onToggle={() =>
                    setOpenSections((prev) => {
                      const n = new Set(prev)
                      if (!n.delete(s.id)) n.add(s.id)
                      return n
                    })
                  }
                />
              ))}
            </div>
          </>
        )}
      </section>

      <div className={card}>
        {topicsError ? (
          <ErrorPanel title="Could not load topics" error={topicsError} />
        ) : (
          <WeakTopics topics={topics} guide={guide} />
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {activity ? (
          <ActivityHeatmap activity={activity} now={now} />
        ) : (
          <section className={card} aria-labelledby="activity-h">
            <h2 id="activity-h" className="text-lg font-semibold">
              Activity
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {activityError ? 'Activity could not be loaded.' : 'Loading…'}
            </p>
          </section>
        )}
        <QuizPerformance course={course.course} progress={progress.sections} quiz={activity?.quiz ?? []} />
      </div>

      <GoalCoverageView progress={progress} topics={topics} />

      <details className={card} data-progress-details>
        <summary className="cursor-pointer text-lg font-semibold">Details</summary>
        <ProgressDetails course={course.course} progress={progress} review={review} />
      </details>
    </div>
  )
}
