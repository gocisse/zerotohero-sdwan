import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchReviewQueue, submitReviewAnswers, type AnswerValue, type QuestionResult, type ReviewQueue } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import Markdown from '../components/Markdown'
import { useCourseState } from '../course/context'
import { formatDateTime, lessonPath } from '../lib/course'
import { useProgress } from '../progress/context'
import QuestionInput from '../quiz/questions/QuestionInput'
import QuestionResultCard from '../quiz/QuestionResultCard'

const primaryBtn =
  'rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none disabled:opacity-50 dark:focus-visible:ring-offset-slate-950'
const secondaryBtn =
  'rounded-md border border-slate-300 px-4 py-2 text-sm hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:hover:border-sky-600'

const isAnswered = (v: AnswerValue | undefined) => v !== undefined && (!Array.isArray(v) || v.length > 0)

type Load = { kind: 'loading' } | { kind: 'ready'; queue: ReviewQueue } | { kind: 'error'; error: Error }

/** Review sessions: one question at a time with instant feedback, then a summary. */
export default function ReviewPage() {
  const [session, setSession] = useState(0)
  return <ReviewSession key={session} onRestart={() => setSession((s) => s + 1)} />
}

function ReviewSession({ onRestart }: { onRestart: () => void }) {
  const course = useCourseState()
  const { refresh } = useProgress()
  const [load, setLoad] = useState<Load>({ kind: 'loading' })
  const [index, setIndex] = useState(0)
  const [value, setValue] = useState<AnswerValue | undefined>(undefined)
  const [feedback, setFeedback] = useState<QuestionResult | null>(null)
  const [results, setResults] = useState<QuestionResult[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    let cancelled = false
    fetchReviewQueue()
      .then((queue) => !cancelled && setLoad({ kind: 'ready', queue }))
      .catch((e: Error) => !cancelled && setLoad({ kind: 'error', error: e }))
    return () => {
      cancelled = true
    }
  }, [])

  if (load.kind === 'loading' || course.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (load.kind === 'error') return <ErrorPanel title="Could not load your review queue" error={load.error} />
  if (course.status === 'error') return <ErrorPanel title="Course content is unavailable" error={course.error} />

  const { queue } = load
  const questions = queue.questions

  if (questions.length === 0) {
    return (
      <div className="max-w-2xl">
        <h1 className="text-2xl font-semibold">Review</h1>
        <div className="mt-4 rounded-lg border border-slate-200 p-6 dark:border-slate-800" data-testid="review-empty">
          <p className="text-lg font-medium">Nothing to review right now 🎉</p>
          <p className="mt-1 text-slate-600 dark:text-slate-400">
            {queue.next_due_at
              ? `Your next review is due ${formatDateTime(queue.next_due_at)} (${queue.upcoming_count} question${queue.upcoming_count === 1 ? '' : 's'} scheduled).`
              : 'Questions you answer in section quizzes will appear here on a spaced schedule.'}
          </p>
          <Link to="/" className="mt-4 inline-block text-sky-700 underline dark:text-sky-300">
            Back to home
          </Link>
        </div>
      </div>
    )
  }

  // Finished: summary.
  if (index >= questions.length) {
    const right = results.filter((r) => r.correct).length
    const missed = results.filter((r) => !r.correct)
    return (
      <div className="max-w-2xl">
        <h1 className="text-2xl font-semibold">Review complete</h1>
        <div className="mt-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800" role="status">
          <div className="text-3xl font-semibold" data-testid="review-score">
            {right} / {results.length} correct
          </div>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
            {missed.length === 0
              ? 'All correct. Those questions move to a longer interval.'
              : `${missed.length} question${missed.length === 1 ? '' : 's'} will come back for another try.`}
          </p>
        </div>
        {missed.length > 0 && (
          <ul className="mt-4 space-y-2 text-sm">
            {missed.map((r) => (
              <li key={r.question_id}>
                <span className="font-medium">{r.topic}</span>
                {r.lesson_ref && lessonPath(course.course, r.lesson_ref) && (
                  <>
                    {' · '}
                    <Link
                      className="text-sky-700 underline dark:text-sky-300"
                      to={lessonPath(course.course, r.lesson_ref) as string}
                    >
                      Review the lesson
                    </Link>
                  </>
                )}
              </li>
            ))}
          </ul>
        )}
        <div className="mt-6 flex gap-3">
          <button type="button" className={primaryBtn} onClick={onRestart}>
            Check for more
          </button>
          <Link to="/progress" className={secondaryBtn}>
            View progress
          </Link>
        </div>
      </div>
    )
  }

  const q = questions[index]!

  const check = async () => {
    if (!isAnswered(value)) return
    setBusy(true)
    setError(null)
    try {
      const res = await submitReviewAnswers([{ question_id: q.id, answer: value as AnswerValue }])
      const r = res.results[0]!
      setFeedback(r)
      setResults((prev) => [...prev, r])
      void refresh()
    } catch (e) {
      setError(e as Error)
    } finally {
      setBusy(false)
    }
  }

  const next = () => {
    setFeedback(null)
    setValue(undefined)
    setIndex(index + 1)
  }

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-semibold">Review</h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400" aria-live="polite">
        Question {index + 1} of {questions.length}
        {queue.due_count > questions.length && ` (${queue.due_count} due in total)`}
      </p>

      {feedback ? (
        <div className="mt-4 space-y-4">
          <QuestionResultCard
            question={q}
            result={feedback}
            lessonTo={feedback.lesson_ref ? lessonPath(course.course, feedback.lesson_ref) : undefined}
          />
          <button type="button" className={primaryBtn} onClick={next} autoFocus>
            {index + 1 >= questions.length ? 'Finish' : 'Next question →'}
          </button>
        </div>
      ) : (
        <section
          className="mt-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
          aria-labelledby="review-prompt"
        >
          <div className="mb-1 text-xs text-slate-500 dark:text-slate-400">
            <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-slate-800">{q.topic}</span>
          </div>
          <div id="review-prompt" className="mb-4 font-medium">
            <Markdown>{q.prompt}</Markdown>
          </div>
          <QuestionInput key={q.id} question={q} value={value} onChange={setValue} />
          <div className="mt-4">
            <button
              type="button"
              className={primaryBtn}
              onClick={() => void check()}
              disabled={busy || !isAnswered(value)}
            >
              {busy ? 'Checking…' : 'Check answer'}
            </button>
          </div>
        </section>
      )}
      {error && (
        <div className="mt-4">
          <ErrorPanel title="Could not check that answer" error={error} />
        </div>
      )}
    </div>
  )
}
