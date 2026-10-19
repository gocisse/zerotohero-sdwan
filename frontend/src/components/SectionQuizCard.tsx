import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { skipQuiz } from '../api'
import { useProgress } from '../progress/context'

interface Props {
  sectionId: string
  questionCount: number
  /** Where both "Skip for now" and the lesson navigation lead: the next section (or progress page). */
  nextPath: string
}

/**
 * Shown after the last lesson of a section. The quiz is a prominent step, never a gate:
 * both buttons lead forward.
 */
export default function SectionQuizCard({ sectionId, questionCount, nextPath }: Props) {
  const navigate = useNavigate()
  const { refresh } = useProgress()
  const [busy, setBusy] = useState(false)

  const skip = async () => {
    setBusy(true)
    try {
      await skipQuiz(sectionId)
      await refresh()
    } catch {
      // Recording the skip is best-effort: moving on must never depend on it.
    }
    navigate(nextPath)
  }

  return (
    <section
      aria-labelledby="check-understanding"
      className="mt-10 rounded-xl border-2 border-sky-300 bg-sky-50 p-6 dark:border-sky-800 dark:bg-sky-950/30"
    >
      <h2 id="check-understanding" className="text-xl font-semibold">
        Check your understanding
      </h2>
      <p className="mt-1 text-slate-700 dark:text-slate-300">
        A short quiz ({questionCount} question{questionCount === 1 ? '' : 's'}) to lock in what you just learned. Take
        it now, or skip it and come back whenever you like.
      </p>
      <div className="mt-4 flex flex-wrap gap-3">
        <Link
          to={`/course/${sectionId}/quiz`}
          className="rounded-md bg-sky-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-sky-700 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950"
        >
          Take the quiz
        </Link>
        <button
          type="button"
          onClick={() => void skip()}
          disabled={busy}
          className="rounded-md border border-slate-400 px-5 py-2.5 text-sm hover:border-sky-500 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:opacity-60 dark:border-slate-600"
        >
          Skip for now
        </button>
      </div>
    </section>
  )
}
