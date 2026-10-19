import type { ReactNode } from 'react'

type Variant = 'info' | 'warning' | 'tip' | 'gotcha'

const calloutStyles: Record<Variant, { box: string; label: string; icon: string }> = {
  info: {
    box: 'border-sky-300 bg-sky-50 dark:border-sky-800 dark:bg-sky-950/40',
    label: 'text-sky-800 dark:text-sky-200',
    icon: 'ℹ️',
  },
  warning: {
    box: 'border-amber-300 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/40',
    label: 'text-amber-800 dark:text-amber-200',
    icon: '⚠️',
  },
  tip: {
    box: 'border-emerald-300 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/40',
    label: 'text-emerald-800 dark:text-emerald-200',
    icon: '💡',
  },
  gotcha: {
    box: 'border-rose-300 bg-rose-50 dark:border-rose-800 dark:bg-rose-950/40',
    label: 'text-rose-800 dark:text-rose-200',
    icon: '🪤',
  },
}

export function Callout({
  variant = 'info',
  title,
  children,
}: {
  variant?: Variant
  title?: string
  children?: ReactNode
}) {
  const s = calloutStyles[variant] ?? calloutStyles.info
  return (
    <aside data-callout={variant} className={`not-prose my-5 rounded-lg border-l-4 border p-4 ${s.box}`}>
      <div className={`mb-1 flex items-center gap-2 text-sm font-semibold ${s.label}`}>
        <span aria-hidden>{s.icon}</span>
        <span>{title ?? variant[0]?.toUpperCase() + variant.slice(1)}</span>
      </div>
      <div className="text-sm leading-relaxed [&>p]:my-1">{children}</div>
    </aside>
  )
}

// Table elements are wrapped/styled explicitly (outside typography's `prose`) for full control.
export function Table(props: React.ComponentProps<'table'>) {
  return (
    <div className="not-prose my-5 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-800">
      <table className="w-full border-collapse text-left text-sm" {...props} />
    </div>
  )
}
export const Th = (props: React.ComponentProps<'th'>) => (
  <th
    className="border-b border-slate-200 bg-slate-100 px-3 py-2 font-semibold dark:border-slate-800 dark:bg-slate-900"
    {...props}
  />
)
export const Td = (props: React.ComponentProps<'td'>) => (
  <td className="border-b border-slate-100 px-3 py-2 last:border-b-0 dark:border-slate-800/60" {...props} />
)
