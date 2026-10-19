import { Link } from 'react-router-dom'
import type { AnswerValue, QuestionResult, QuizQuestion } from '../api'
import Markdown from '../components/Markdown'

const text = (q: QuizQuestion, id: string) => q.options.find((o) => o.id === id)?.text ?? id

/** Human-readable answer: option text(s); ordering is shown as a numbered sequence. */
function describe(q: QuizQuestion, v: AnswerValue | null): string {
  if (v === null || v === undefined || (Array.isArray(v) && v.length === 0)) return '— (no answer)'
  if (typeof v === 'string') return text(q, v)
  const names = v.map((id) => text(q, id))
  return q.type === 'ordering' ? names.map((n, i) => `${i + 1}. ${n}`).join('  ') : names.join(', ')
}

interface Props {
  index?: number
  question: QuizQuestion
  result: QuestionResult
  /** Path of the lesson to review (from lesson_ref), if it exists in the course. */
  lessonTo?: string
}

/** Feedback for one graded question: verdict, answers, explanations, and a lesson link. No lock language. */
export default function QuestionResultCard({ index, question, result, lessonTo }: Props) {
  const partial = !result.correct && result.partial_score > 0
  const verdict = result.correct
    ? 'Correct'
    : partial
      ? `Partly correct (${Math.round(result.partial_score * 100)}%)`
      : result.answered
        ? 'Incorrect'
        : 'Not answered'
  const tone = result.correct
    ? 'border-emerald-300 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/30'
    : partial
      ? 'border-amber-300 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/30'
      : 'border-rose-300 bg-rose-50 dark:border-rose-800 dark:bg-rose-950/30'
  const label = result.correct
    ? 'text-emerald-800 dark:text-emerald-200'
    : partial
      ? 'text-amber-800 dark:text-amber-200'
      : 'text-rose-800 dark:text-rose-200'
  const wrongExplanations = Object.entries(result.option_explanations)

  return (
    <article
      className={`rounded-lg border p-4 ${tone}`}
      data-result={result.correct ? 'correct' : 'incorrect'}
      aria-label={`Question ${index ?? ''} ${verdict}`}
    >
      <div className={`text-sm font-semibold ${label}`}>
        <span aria-hidden>{result.correct ? '✓' : '✗'}</span> {index !== undefined && `Question ${index} · `}
        {verdict}
      </div>
      <Markdown className="mt-1 font-medium">{question.prompt}</Markdown>
      <dl className="mt-2 space-y-1 text-sm">
        {!result.correct && (
          <div>
            <dt className="inline font-medium">Your answer: </dt>
            <dd className="inline">{describe(question, result.given)}</dd>
          </div>
        )}
        <div>
          <dt className="inline font-medium">Correct answer: </dt>
          <dd className="inline">{describe(question, result.correct_answer)}</dd>
        </div>
      </dl>
      {wrongExplanations.map(([id, why]) => (
        <p key={id} className="mt-2 text-sm text-slate-700 dark:text-slate-300">
          <span className="font-medium">About “{text(question, id)}”:</span> {why}
        </p>
      ))}
      <Markdown className="mt-2 text-sm">{result.explanation}</Markdown>
      {lessonTo && !result.correct && (
        <Link
          to={lessonTo}
          className="mt-2 inline-block text-sm font-medium text-sky-700 underline hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-100"
        >
          Review this lesson →
        </Link>
      )}
    </article>
  )
}
