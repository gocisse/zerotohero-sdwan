import { optionClass, type QuestionInputProps } from './shared'

/** Radio group: one option id. Also used by TrueFalse. */
export default function SingleChoice({ question, value, onChange, disabled }: QuestionInputProps) {
  return (
    <fieldset className="space-y-2" disabled={disabled}>
      <legend className="sr-only">Choose one answer</legend>
      {question.options.map((o) => (
        <label key={o.id} className={optionClass}>
          <input
            type="radio"
            name={`q-${question.id}`}
            value={o.id}
            checked={value === o.id}
            onChange={() => onChange(o.id)}
            className="mt-0.5 h-4 w-4 accent-sky-600"
          />
          <span>{o.text}</span>
        </label>
      ))}
    </fieldset>
  )
}
