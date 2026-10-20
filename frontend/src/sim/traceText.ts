import type { TraceStatus } from '../api'

export const TRACE_STATUS_TEXT: Record<TraceStatus, string> = {
  ok: 'Passed',
  blocked: 'Blocked',
  failed: 'Failed',
  info: 'Note',
  'not-modelled': 'Not modelled',
  skipped: 'Not needed',
  'not-reached': 'Not reached',
}

export const TRACE_STATUS_MARK: Record<TraceStatus, string> = {
  ok: '✓',
  blocked: '✗',
  failed: '✗',
  info: 'i',
  'not-modelled': '?',
  skipped: '–',
  'not-reached': '·',
}

export const TRACE_STATUS_STYLE: Record<TraceStatus, string> = {
  ok: 'border-emerald-400 bg-emerald-50/60 dark:border-emerald-700 dark:bg-emerald-950/20',
  blocked: 'border-red-400 bg-red-50/70 dark:border-red-700 dark:bg-red-950/30',
  failed: 'border-red-400 bg-red-50/70 dark:border-red-700 dark:bg-red-950/30',
  info: 'border-sky-300 bg-sky-50/50 dark:border-sky-800 dark:bg-sky-950/20',
  'not-modelled': 'border-dashed border-rose-300 bg-rose-50/60 dark:border-rose-800 dark:bg-rose-950/20',
  skipped: 'border-slate-200 dark:border-slate-800',
  'not-reached': 'border-slate-200 opacity-70 dark:border-slate-800',
}

export const VERDICT_TEXT: Record<string, string> = {
  delivered: 'Delivered',
  blocked: 'Blocked',
  unreachable: 'Unreachable',
  'not-modelled': 'Cannot be followed to the end (not modelled)',
}
