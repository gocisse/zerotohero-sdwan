import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { fetchLesson, putLessonProgress, type Lesson } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import CapstoneCard from '../components/CapstoneCard'
import SectionQuizCard from '../components/SectionQuizCard'
import { flattenLessons, useCourseState } from '../course/context'
import { nextSectionPath } from '../lib/course'
import { useProgress } from '../progress/context'
import NotFound from './NotFound'

// MDX compiler + highlighter are large; load them only when a lesson is opened.
const MdxRenderer = lazy(() => import('../mdx/MdxRenderer'))

type LessonState = { kind: 'loading' } | { kind: 'ready'; lesson: Lesson } | { kind: 'error'; error: Error }

export default function LessonPage() {
  const course = useCourseState()
  const { sectionId, lessonId } = useParams()

  const flat = useMemo(() => (course.status === 'ready' ? flattenLessons(course.course) : []), [course])

  if (course.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (course.status === 'error') return <ErrorPanel title="Course content is unavailable" error={course.error} />

  const index = flat.findIndex((l) => l.id === lessonId && l.sectionId === sectionId)
  if (index < 0) return <NotFound />
  const prev = flat[index - 1]
  const next = flat[index + 1]
  const section = course.course.sections.find((s) => s.id === sectionId)
  const isLastInSection = next?.sectionId !== sectionId
  const afterSection = nextSectionPath(course.course, sectionId ?? '')

  return (
    <article className="max-w-3xl">
      <LessonBody key={lessonId} lessonId={lessonId ?? ''} />

      {isLastInSection && section && section.question_count > 0 && (
        <SectionQuizCard sectionId={section.id} questionCount={section.question_count} nextPath={afterSection.to} />
      )}

      {!next && <CapstoneCard />}

      <nav
        aria-label="Lesson navigation"
        className="mt-10 flex justify-between gap-4 border-t border-slate-200 pt-4 dark:border-slate-800"
      >
        {prev ? (
          <Link to={`/course/${prev.sectionId}/${prev.id}`} className={navButton}>
            ← {prev.title}
          </Link>
        ) : (
          <span />
        )}
        {next ? (
          <Link to={`/course/${next.sectionId}/${next.id}`} className={navButton}>
            {next.title} →
          </Link>
        ) : (
          <Link to="/progress" className={navButton}>
            View your progress →
          </Link>
        )}
      </nav>
    </article>
  )
}

const navButton =
  'rounded-md border border-slate-300 px-4 py-2 text-sm hover:border-sky-400 hover:text-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:hover:border-sky-600 dark:hover:text-sky-300'

/**
 * Fetches and renders one lesson and records progress: 'visited' on open, 'completed' when the end
 * is scrolled into view or "Mark as complete" is clicked. Mounted with key={lessonId} so state resets.
 */
function LessonBody({ lessonId }: { lessonId: string }) {
  const [state, setState] = useState<LessonState>({ kind: 'loading' })
  const { progress, refresh } = useProgress()
  const endRef = useRef<HTMLDivElement>(null)
  const sentCompleted = useRef(false)
  const [rendered, setRendered] = useState(false)

  const status = progress?.lessons.find((l) => l.lesson_id === lessonId)?.status
  const completed = status === 'completed'

  useEffect(() => {
    let cancelled = false
    fetchLesson(lessonId)
      .then((lesson) => !cancelled && setState({ kind: 'ready', lesson }))
      .catch((error: Error) => !cancelled && setState({ kind: 'error', error }))
    // Progress is best-effort; failures must never get in the way of reading.
    putLessonProgress(lessonId, 'visited')
      .then(() => refresh())
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [lessonId, refresh])

  // Resolves true once the server has the lesson as completed (or a request is already on its way).
  const markComplete = useCallback((): Promise<boolean> => {
    if (sentCompleted.current) return Promise.resolve(true)
    sentCompleted.current = true
    return putLessonProgress(lessonId, 'completed')
      .then(() => {
        void refresh()
        return true
      })
      .catch(() => {
        sentCompleted.current = false
        return false
      })
  }, [lessonId, refresh])

  // Scrolling the end-of-lesson marker into view completes the lesson.
  useEffect(() => {
    const el = endRef.current
    if (!rendered || !el || typeof IntersectionObserver === 'undefined') return
    let retry: ReturnType<typeof setTimeout> | undefined
    const io = new IntersectionObserver((entries) => {
      if (!entries.some((e) => e.isIntersecting)) return
      void markComplete().then((ok) => {
        if (ok) {
          io.disconnect()
        } else {
          // The request failed: observe again shortly so that a marker that stays in view is reported once more.
          io.unobserve(el)
          retry = setTimeout(() => io.observe(el), 2000)
        }
      })
    })
    io.observe(el)
    return () => {
      clearTimeout(retry)
      io.disconnect()
    }
  }, [rendered, markComplete])

  return (
    <>
      {state.kind === 'loading' && <p className="text-slate-500">Loading lesson…</p>}
      {state.kind === 'error' && <ErrorPanel title="Could not load this lesson" error={state.error} />}
      {state.kind === 'ready' && (
        <>
          <header className="mb-6">
            <h1 className="text-3xl font-semibold">{state.lesson.title}</h1>
            <p className="mt-2 text-slate-600 dark:text-slate-400">{state.lesson.summary}</p>
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
              <span>{state.lesson.estimated_minutes} min</span>
              {state.lesson.topics.map((t) => (
                <span key={t} className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
                  {t}
                </span>
              ))}
            </div>
          </header>
          <div className="prose prose-slate max-w-none dark:prose-invert prose-pre:bg-[#0d1117] prose-pre:p-0 prose-code:before:content-none prose-code:after:content-none prose-blockquote:font-normal prose-blockquote:not-italic [&_blockquote_p]:before:content-none [&_blockquote_p]:after:content-none">
            <Suspense fallback={<p className="text-slate-500">Rendering…</p>}>
              <MdxRenderer
                key={state.lesson.source}
                source={state.lesson.source}
                onRendered={() => setRendered(true)}
              />
            </Suspense>
          </div>
          <div ref={endRef} data-lesson-end className="mt-8 flex items-center gap-3">
            {completed ? (
              <span className="text-sm font-medium text-emerald-700 dark:text-emerald-300" role="status">
                ✓ Lesson completed
              </span>
            ) : (
              <button
                type="button"
                onClick={markComplete}
                className="rounded-md border border-emerald-500 px-4 py-2 text-sm font-medium text-emerald-700 hover:bg-emerald-50 focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:outline-none dark:text-emerald-300 dark:hover:bg-emerald-950/40"
              >
                Mark as complete
              </button>
            )}
          </div>
        </>
      )}
    </>
  )
}
