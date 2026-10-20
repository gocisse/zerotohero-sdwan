import type { Coverage } from '../api'

const styles: Record<Coverage, { label: string; cls: string }> = {
  full: {
    label: 'Full',
    cls: 'border-emerald-500 bg-emerald-50 text-emerald-900 dark:border-emerald-500 dark:bg-emerald-950/40 dark:text-emerald-200',
  },
  partial: {
    label: 'Partial',
    cls: 'border-amber-500 bg-amber-50 text-amber-900 dark:border-amber-500 dark:bg-amber-950/40 dark:text-amber-200',
  },
  none: {
    label: 'Not covered',
    cls: 'border-dashed border-slate-400 bg-transparent text-slate-600 dark:border-slate-500 dark:text-slate-300',
  },
}

/** The coverage level of a guide topic; the note (what is and is not covered) shows on hover and on keyboard focus. */
export default function CoverageBadge({ coverage, note }: { coverage: Coverage; note: string }) {
  const s = styles[coverage]
  return (
    <span className="group relative inline-block">
      <span
        tabIndex={0}
        aria-describedby={undefined}
        data-coverage={coverage}
        className={`inline-block rounded-full border px-2 py-0.5 text-xs font-medium focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${s.cls}`}
      >
        {s.label}
        <span className="sr-only">: {note}</span>
      </span>
      <span
        role="tooltip"
        aria-hidden
        className="pointer-events-none invisible absolute top-full left-0 z-20 mt-1 w-64 max-w-[70vw] rounded-md border border-slate-300 bg-white p-2 text-xs font-normal text-slate-800 shadow-lg group-focus-within:visible group-hover:visible dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
      >
        {note}
      </span>
    </span>
  )
}
