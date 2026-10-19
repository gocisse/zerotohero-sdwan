import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  fetchQuiz,
  submitQuizAttempt,
  type AnswerValue,
  type AttemptResult,
  type Course,
  type QuizQuestion,
} from '../api'
import ErrorPanel from '../components/ErrorPanel'
import Markdown from '../components/Markdown'
import { flattenLessons, useCourseState } from '../course/context'
import { lessonPath, nextSectionPath } from '../lib/course'
import { useProgress } from '../progress/context'
import QuestionInput from '../quiz/questions/QuestionInput'
import QuestionResultCard from '../quiz/QuestionResultCard'
import NotFound from './NotFound'

const primaryBtn =
  'rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none disabled:opacity-50 dark:focus-visible:ring-offset-slate-950'
const secondaryBtn =
  'rounded-md border border-slate-300 px-4 py-2 text-sm hover:border-sky-400 hover:text-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-40 dark:border-slate-700 dark:hover:border-sky-600 dark:hover:text-sky-300'

const isAnswered = (v: AnswerValue | undefined) => v !== undefined && (!Array.isArray(v) || v.length > 0)

function shuffled<T>(items: T[]): T[] {
  const a = [...items]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j] as T, a[i] as T]
  }
  return a
}

/** Quiz page. Retrying remounts the run (new fetch, new option shuffle where the type allows it). */
export default function QuizPage() {
  const { sectionId = '' } = useParams()
  const [run, setRun] = useState(0)
  return (
    <QuizRun
      key={`${sectionId}:${run}`}
      sectionId={sectionId}
      shuffleChoices={run > 0}
      onRetry={() => setRun((r) => r + 1)}
    />
  )
}

type Load = { kind: 'loading' } | { kind: 'ready'; questions: QuizQuestion[] } | { kind: 'error'; error: Error }

function QuizRun({
  sectionId,
  shuffleChoices,
  onRetry,
}: {
  sectionId: string
  shuffleChoices: boolean
  onRetry: () => void
}) {
  const course = useCourseState()
  const { refresh } = useProgress()
  const [load, setLoad] = useState<Load>({ kind: 'loading' })
  const [answers, setAnswers] = useState<Record<string, AnswerValue>>({})
  const [index, setIndex] = useState(0)
  const [confirming, setConfirming] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<Error | null>(null)
  const [result, setResult] = useState<AttemptResult | null>(null)

  useEffect(() => {
    let cancelled = false
    fetchQuiz(sectionId)
      .then((quiz) => {
        if (cancelled) return
        // Only single/multi choice are shuffled. True/false keeps its order and ordering questions
        // arrive already shuffled from the server (their authored order is the answer).
        const questions = quiz.questions.map((q) =>
          shuffleChoices && (q.type === 'single_choice' || q.type === 'multi_choice')
            ? { ...q, options: shuffled(q.options) }
            : q,
        )
        setLoad({ kind: 'ready', questions })
      })
      .catch((error: Error) => !cancelled && setLoad({ kind: 'error', error }))
    return () => {
      cancelled = true
    }
  }, [sectionId, shuffleChoices])

  if (course.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (course.status === 'error') return <ErrorPanel title="Course content is unavailable" error={course.error} />
  const section = course.course.sections.find((s) => s.id === sectionId)
  if (!section) return <NotFound />
  if (load.kind === 'loading') return <p className="text-slate-500">Loading quiz…</p>
  if (load.kind === 'error') return <ErrorPanel title="Could not load this quiz" error={load.error} />

  const { questions } = load
  const answeredCount = questions.filter((q) => isAnswered(answers[q.id])).length

  const submit = async () => {
    setConfirming(false)
    setSubmitting(true)
    setSubmitError(null)
    try {
      const res = await submitQuizAttempt(
        sectionId,
        questions.map((q) => ({
          question_id: q.id,
          answer: isAnswered(answers[q.id]) ? (answers[q.id] as AnswerValue) : null,
        })),
      )
      setResult(res)
      void refresh()
      window.scrollTo?.({ top: 0 })
    } catch (e) {
      setSubmitError(e as Error)
    } finally {
      setSubmitting(false)
    }
  }

  if (result) {
    return (
      <QuizResults
        course={course.course}
        sectionId={sectionId}
        sectionTitle={section.title}
        questions={questions}
        result={result}
        onRetry={onRetry}
      />
    )
  }

  if (questions.length === 0) {
    return (
      <div className="max-w-3xl">
        <p>This section has no quiz questions yet.</p>
        <Link className="mt-3 inline-block text-sky-700 underline dark:text-sky-300" to={`/course/${sectionId}`}>
          Back to section
        </Link>
      </div>
    )
  }

  const q = questions[Math.min(index, questions.length - 1)] as QuizQuestion
  const unanswered = questions.length - answeredCount

  return (
    <div className="max-w-3xl">
      <Link to={`/course/${sectionId}`} className="text-sm text-slate-500 hover:underline dark:text-slate-400">
        ← Back to section
      </Link>
      <h1 className="mt-2 text-2xl font-semibold">Quiz: {section.title}</h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400" aria-live="polite">
        Question {index + 1} of {questions.length} · {answeredCount} answered
      </p>
      {section.placeholder && (
        <p className="mt-1 text-xs font-semibold text-amber-700 dark:text-amber-300">PLACEHOLDER QUIZ</p>
      )}

      <nav aria-label="Questions" className="mt-4 flex flex-wrap gap-2">
        {questions.map((qq, i) => {
          const done = isAnswered(answers[qq.id])
          return (
            <button
              key={qq.id}
              type="button"
              onClick={() => setIndex(i)}
              aria-label={`Question ${i + 1}, ${done ? 'answered' : 'not answered'}`}
              aria-current={i === index ? 'step' : undefined}
              className={`h-9 w-9 rounded-full border text-sm font-medium focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
                i === index ? 'border-sky-600 ring-2 ring-sky-300 dark:ring-sky-800' : ''
              } ${
                done
                  ? 'border-sky-600 bg-sky-600 text-white'
                  : 'border-slate-300 text-slate-600 hover:border-sky-400 dark:border-slate-700 dark:text-slate-300'
              }`}
            >
              {i + 1}
            </button>
          )
        })}
      </nav>

      <section
        aria-labelledby="question-prompt"
        className="mt-5 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
      >
        <div className="mb-1 flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
          <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-slate-800">{q.topic}</span>
          <span>difficulty {q.difficulty}/3</span>
        </div>
        <div id="question-prompt" className="mb-4 text-base font-medium">
          <Markdown>{q.prompt}</Markdown>
        </div>
        <QuestionInput question={q} value={answers[q.id]} onChange={(v) => setAnswers((a) => ({ ...a, [q.id]: v }))} />
      </section>

      <div className="mt-5 flex flex-wrap items-center gap-3">
        <button type="button" className={secondaryBtn} disabled={index === 0} onClick={() => setIndex(index - 1)}>
          ← Previous
        </button>
        <button
          type="button"
          className={secondaryBtn}
          disabled={index >= questions.length - 1}
          onClick={() => setIndex(index + 1)}
        >
          Next →
        </button>
        <span className="flex-1" />
        <button
          type="button"
          className={primaryBtn}
          disabled={submitting}
          onClick={() => (unanswered > 0 ? setConfirming(true) : void submit())}
        >
          {submitting ? 'Submitting…' : 'Submit quiz'}
        </button>
      </div>

      {confirming && (
        <div
          role="alertdialog"
          aria-labelledby="confirm-title"
          className="mt-4 rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950/30"
        >
          <p id="confirm-title" className="font-medium">
            You have {unanswered} unanswered question{unanswered === 1 ? '' : 's'}. Submit anyway?
          </p>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            Unanswered questions count as incorrect, and you can retry the quiz whenever you like.
          </p>
          <div className="mt-3 flex gap-3">
            <button type="button" className={primaryBtn} onClick={() => void submit()} autoFocus>
              Submit anyway
            </button>
            <button type="button" className={secondaryBtn} onClick={() => setConfirming(false)}>
              Keep working
            </button>
          </div>
        </div>
      )}
      {submitError && (
        <div className="mt-4">
          <ErrorPanel title="Could not submit the quiz" error={submitError} />
        </div>
      )}
    </div>
  )
}

interface ResultsProps {
  course: Course
  sectionId: string
  sectionTitle: string
  questions: QuizQuestion[]
  result: AttemptResult
  onRetry: () => void
}

function QuizResults({ course, sectionId, sectionTitle, questions, result, onRetry }: ResultsProps) {
  const heading = useRef<HTMLHeadingElement>(null)
  useEffect(() => heading.current?.focus(), [])

  const byId = useMemo(() => new Map(questions.map((q) => [q.id, q])), [questions])
  const lessons = useMemo(() => new Map(flattenLessons(course).map((l) => [l.id, l])), [course])
  const next = nextSectionPath(course, sectionId)

  // Topics with at least one miss, each with the lessons to revisit (from lesson_ref).
  const toRevisit = useMemo(() => {
    const topics = new Map<string, Set<string>>()
    for (const r of result.results) {
      if (r.correct) continue
      const set = topics.get(r.topic) ?? new Set<string>()
      if (r.lesson_ref && lessons.has(r.lesson_ref)) set.add(r.lesson_ref)
      topics.set(r.topic, set)
    }
    return [...topics.entries()]
  }, [result, lessons])

  return (
    <div className="max-w-3xl">
      <h1 ref={heading} tabIndex={-1} className="text-2xl font-semibold outline-none">
        Quiz results: {sectionTitle}
      </h1>

      <div className="mt-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800" role="status">
        <div className="text-4xl font-semibold" data-testid="score">
          {result.score} / {result.total}
        </div>
        <div className="mt-1 text-slate-600 dark:text-slate-400">{Math.round(result.percentage)}% correct</div>
        <p className="mt-3 text-sm">
          {result.passed
            ? 'Nice work! You’ve got a solid grip on this section.'
            : 'Good practice round. A few things are worth another look. Retry whenever you like, or carry on; it’s entirely up to you.'}
        </p>
      </div>

      {!result.passed && toRevisit.length > 0 && (
        <section
          aria-labelledby="revisit"
          className="mt-4 rounded-lg border border-sky-200 bg-sky-50 p-4 dark:border-sky-900 dark:bg-sky-950/30"
        >
          <h2 id="revisit" className="font-semibold">
            Topics worth another look
          </h2>
          <ul className="mt-2 space-y-1 text-sm">
            {toRevisit.map(([topic, refs]) => (
              <li key={topic}>
                <span className="font-medium">{topic}</span>
                {[...refs].map((id) => (
                  <span key={id}>
                    {' · '}
                    <Link className="text-sky-700 underline dark:text-sky-300" to={lessonPath(course, id) ?? '#'}>
                      {lessons.get(id)?.title}
                    </Link>
                  </span>
                ))}
              </li>
            ))}
          </ul>
        </section>
      )}

      <div className="mt-6 space-y-4">
        {result.results.map((r, i) => {
          const q = byId.get(r.question_id)
          return q ? (
            <QuestionResultCard
              key={r.question_id}
              index={i + 1}
              question={q}
              result={r}
              lessonTo={r.lesson_ref ? lessonPath(course, r.lesson_ref) : undefined}
            />
          ) : null
        })}
      </div>

      <div className="mt-8 flex flex-wrap gap-3 border-t border-slate-200 pt-5 dark:border-slate-800">
        <button type="button" className={secondaryBtn} onClick={onRetry}>
          Retry quiz
        </button>
        <Link to={`/course/${sectionId}`} className={secondaryBtn}>
          Back to section
        </Link>
        <Link to={next.to} className={primaryBtn}>
          {next.isLast ? 'Continue to your progress' : 'Continue to next section'} →
        </Link>
      </div>
    </div>
  )
}
