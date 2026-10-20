/** A progress ring (inline SVG). `fraction` is 0..1. */
export default function Ring({
  fraction,
  label,
  center,
  size = 120,
}: {
  fraction: number
  label: string
  center: string
  size?: number
}) {
  const r = size / 2 - 9
  const c = 2 * Math.PI * r
  const f = Math.max(0, Math.min(1, fraction))
  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      role="img"
      aria-label={label}
      className="shrink-0"
      data-ring
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        strokeWidth={10}
        className="stroke-slate-200 dark:stroke-slate-700"
      />
      {f > 0 && (
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          strokeWidth={10}
          strokeLinecap="round"
          strokeDasharray={`${c * f} ${c}`}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
          className="stroke-emerald-600 dark:stroke-emerald-400"
        />
      )}
      <text
        x="50%"
        y="50%"
        textAnchor="middle"
        dominantBaseline="central"
        className="fill-slate-900 text-xl font-semibold dark:fill-slate-100"
      >
        {center}
      </text>
    </svg>
  )
}
