import { useId, useState, type ReactNode } from 'react'
import Markdown from '../components/Markdown'

interface Props {
  prompt: string
  choices: string[]
  /** Zero-based index into `choices`. */
  answer: number
  children?: ReactNode
}

type Pick = number | 'unsure' | null

const btn =
  'rounded-md border px-3 py-2 text-left text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none disabled:cursor-default'

/**
 * "Predict before you reveal": think first, then see the answer and the explanation.
 * Deliberately NOT graded and NOT stored: nothing is recorded, nothing is blocked, and the reader can reset it.
 */
export default function Predict({ prompt, choices, answer, children }: Props) {
  const [picked, setPicked] = useState<Pick>(null)
  const promptId = useId()

  if (
    typeof prompt !== 'string' ||
    !Array.isArray(choices) ||
    choices.length < 2 ||
    !Number.isInteger(answer) ||
    answer < 0 ||
    answer >= choices.length
  ) {
    return (
      <div
        role="alert"
        className="not-prose my-5 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-900 dark:border-red-800 dark:bg-red-950/40 dark:text-red-100"
      >
        <b>&lt;Predict&gt; is misconfigured:</b> it needs a prompt, at least two choices, and a zero-based{' '}
        <code>answer</code> index inside <code>choices</code>.
      </div>
    )
  }

  const revealed = picked !== null
  const gotIt = picked === answer

  return (
    <section
      aria-labelledby={promptId}
      data-predict
      className="not-prose my-6 rounded-xl border-2 border-violet-300 bg-violet-50 p-5 dark:border-violet-800 dark:bg-violet-950/30"
    >
      <div className="text-xs font-semibold tracking-wide text-violet-700 uppercase dark:text-violet-300">
        Predict before you reveal
      </div>
      <div id={promptId} className="mt-1 font-medium">
        <Markdown>{prompt}</Markdown>
      </div>

      <div role="group" aria-labelledby={promptId} className="mt-3 grid gap-2">
        {choices.map((c, i) => {
          const isAnswer = revealed && i === answer
          const isWrongPick = revealed && picked === i && i !== answer
          return (
            <button
              key={i}
              type="button"
              disabled={revealed}
              aria-pressed={picked === i}
              data-choice={i}
              data-state={isAnswer ? 'correct' : isWrongPick ? 'wrong' : undefined}
              onClick={() => setPicked(i)}
              className={`${btn} ${
                isAnswer
                  ? 'border-emerald-500 bg-emerald-50 dark:bg-emerald-950/40'
                  : isWrongPick
                    ? 'border-rose-500 bg-rose-50 dark:bg-rose-950/40'
                    : 'border-slate-300 bg-white hover:border-violet-400 dark:border-slate-700 dark:bg-slate-900'
              }`}
            >
              <span aria-hidden className="mr-2">
                {isAnswer ? '✓' : isWrongPick ? '✗' : '○'}
              </span>
              <Markdown inline>{c}</Markdown>
              {isAnswer && <span className="sr-only"> (correct answer)</span>}
              {isWrongPick && <span className="sr-only"> (your choice, not correct)</span>}
            </button>
          )
        })}
        {!revealed && (
          <button
            type="button"
            data-unsure
            onClick={() => setPicked('unsure')}
            className={`${btn} border-dashed border-slate-400 bg-transparent text-slate-600 hover:border-violet-400 dark:border-slate-600 dark:text-slate-300`}
          >
            I’m not sure
          </button>
        )}
      </div>

      <div aria-live="polite" data-predict-result>
        {revealed && (
          <div className="mt-4 rounded-lg border border-slate-200 bg-white p-4 text-sm dark:border-slate-700 dark:bg-slate-900">
            <p className="font-semibold">
              {picked === 'unsure'
                ? 'No problem. Here is the answer.'
                : gotIt
                  ? '✓ Nice prediction!'
                  : '✗ Not quite. Predicting wrong is part of learning.'}
            </p>
            <p className="mt-1">
              The answer:{' '}
              <b>
                <Markdown inline>{choices[answer] as string}</Markdown>
              </b>
            </p>
            <div className="mt-2 prose prose-sm prose-slate max-w-none dark:prose-invert">{children}</div>
            <button
              type="button"
              onClick={() => setPicked(null)}
              className="mt-3 rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-violet-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-600"
            >
              ↺ Try again
            </button>
          </div>
        )}
      </div>

      <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">
        Not graded or saved. The answer is visible in the page source, so this is for thinking, not scoring.
      </p>
    </section>
  )
}
