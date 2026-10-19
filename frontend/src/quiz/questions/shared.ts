import type { AnswerValue, QuizQuestion } from '../../api'

export interface QuestionInputProps {
  question: QuizQuestion
  value: AnswerValue | undefined
  onChange: (value: AnswerValue) => void
  disabled?: boolean
}

/** Shared option row styling: highlights when checked, shows a focus ring for keyboard users. */
export const optionClass =
  'flex cursor-pointer items-start gap-3 rounded-lg border border-slate-300 px-4 py-3 text-sm transition-colors ' +
  'hover:border-sky-400 has-[:checked]:border-sky-500 has-[:checked]:bg-sky-50 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-sky-500 ' +
  'dark:border-slate-700 dark:hover:border-sky-600 dark:has-[:checked]:border-sky-500 dark:has-[:checked]:bg-sky-950/40'
