import type { TunnelState } from '../api'
import { stateLabel } from './overlayText'

const chip = 'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap'

const stateStyle: Record<TunnelState, string> = {
  up: 'border-emerald-400 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-100',
  degraded:
    'border-amber-400 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100',
  down: 'border-red-400 bg-red-50 text-red-900 dark:border-red-700 dark:bg-red-950/40 dark:text-red-100',
  'not-established':
    'border-dashed border-slate-400 bg-transparent text-slate-700 dark:border-slate-500 dark:text-slate-200',
  'torn-down':
    'border-slate-400 bg-slate-100 text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200',
  unknown:
    'border-violet-400 bg-violet-50 text-violet-900 dark:border-violet-700 dark:bg-violet-950/40 dark:text-violet-100',
}

// A symbol as well as a colour: state never depends on colour alone.
const stateSymbol: Record<TunnelState, string> = {
  up: '●',
  degraded: '▲',
  down: '✕',
  'not-established': '○',
  'torn-down': '⊘',
  unknown: '?',
}

/** The simulator's own tunnel state (the real VCO's List Paths states are Stable, Unstable, BW_Measurement, Quiet). */
export function StateBadge({ state }: { state: TunnelState | 'up' | 'degraded' | 'down' }) {
  return (
    <span className={`${chip} ${stateStyle[state]}`} data-state={state}>
      <span aria-hidden>{stateSymbol[state]}</span>
      {stateLabel[state]}
    </span>
  )
}
