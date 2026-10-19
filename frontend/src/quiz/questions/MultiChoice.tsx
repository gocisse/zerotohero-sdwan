import { optionClass, type QuestionInputProps } from './shared'

/** Checkbox group: any number of option ids. */
export default function MultiChoice({ question, value, onChange, disabled }: QuestionInputProps) {
  const picked = Array.isArray(value) ? value : []
  const toggle = (id: string) => onChange(picked.includes(id) ? picked.filter((x) => x !== id) : [...picked, id])
  return (
    <fieldset className="space-y-2" disabled={disabled}>
      <legend className="sr-only">Choose all answers that apply</legend>
      <p className="text-xs text-slate-500 dark:text-slate-400">Select all that apply.</p>
      {question.options.map((o) => (
        <label key={o.id} className={optionClass}>
          <input
            type="checkbox"
            value={o.id}
            checked={picked.includes(o.id)}
            onChange={() => toggle(o.id)}
            className="mt-0.5 h-4 w-4 accent-sky-600"
          />
          <span>{o.text}</span>
        </label>
      ))}
    </fieldset>
  )
}
