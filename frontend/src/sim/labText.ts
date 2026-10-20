import type { LabStatus } from '../api'

export const LAB_STATUS_TEXT: Record<LabStatus, string> = {
  not_started: 'Not started',
  started: 'In progress',
  completed: 'Completed',
  skipped: 'Skipped',
}

export const LAB_STATUS_STYLE: Record<LabStatus, string> = {
  not_started: 'border-slate-300 text-slate-700 dark:border-slate-600 dark:text-slate-200',
  started: 'border-sky-400 bg-sky-50 text-sky-900 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-100',
  completed:
    'border-emerald-400 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-100',
  skipped: 'border-amber-400 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100',
}

export function difficultyText(d: number): string {
  return ['', 'Easy', 'Medium', 'Hard'][d] ?? String(d)
}
