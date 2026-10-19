import { useEffect, useRef } from 'react'
import type { QuestionInputProps } from './shared'

const btn =
  'rounded-md border border-slate-300 px-2.5 py-1 text-sm hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ' +
  'disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-700 dark:hover:border-sky-600'

/**
 * Ordering with up/down buttons (keyboard operable; focus follows the moved item).
 * The list is shown in the order the server sent (already shuffled); the answer counts as given once the
 * user moves something or confirms "Keep this order".
 */
export default function Ordering({ question, value, onChange, disabled }: QuestionInputProps) {
  const order = Array.isArray(value) ? value : question.options.map((o) => o.id)
  const byId = new Map(question.options.map((o) => [o.id, o.text]))
  const refocus = useRef<string | null>(null)

  useEffect(() => {
    if (refocus.current) {
      document.getElementById(refocus.current)?.focus()
      refocus.current = null
    }
  })

  const move = (index: number, delta: -1 | 1) => {
    const next = [...order]
    const target = index + delta
    const a = next[index]
    const b = next[target]
    if (a === undefined || b === undefined) return
    next[index] = b
    next[target] = a
    // Keep focus on the same button; if the item hit an end, move focus to its other button.
    const atEnd = delta === -1 ? target === 0 : target === next.length - 1
    refocus.current = `ord-${question.id}-${a}-${atEnd ? (delta === -1 ? 'down' : 'up') : delta === -1 ? 'up' : 'down'}`
    onChange(next)
  }

  return (
    <fieldset disabled={disabled}>
      <legend className="sr-only">Put the items in the correct order</legend>
      <p className="mb-2 text-xs text-slate-500 dark:text-slate-400">
        Use the arrow buttons to put the items in order, first to last.
      </p>
      <ol className="space-y-2">
        {order.map((id, i) => (
          <li
            key={id}
            className="flex items-center gap-3 rounded-lg border border-slate-300 px-4 py-2 text-sm dark:border-slate-700"
          >
            <span className="w-5 shrink-0 text-slate-400" aria-hidden>
              {i + 1}.
            </span>
            <span className="flex-1">{byId.get(id) ?? id}</span>
            <button
              type="button"
              id={`ord-${question.id}-${id}-up`}
              className={btn}
              disabled={i === 0}
              onClick={() => move(i, -1)}
              aria-label={`Move ${byId.get(id) ?? id} up`}
            >
              ↑
            </button>
            <button
              type="button"
              id={`ord-${question.id}-${id}-down`}
              className={btn}
              disabled={i === order.length - 1}
              onClick={() => move(i, 1)}
              aria-label={`Move ${byId.get(id) ?? id} down`}
            >
              ↓
            </button>
          </li>
        ))}
      </ol>
      <div className="mt-2 flex items-center gap-3">
        <button type="button" className={btn} onClick={() => onChange(order)} aria-pressed={Array.isArray(value)}>
          {Array.isArray(value) ? '✓ Order confirmed' : 'Keep this order'}
        </button>
      </div>
    </fieldset>
  )
}
