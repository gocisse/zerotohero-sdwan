import type { Tunnel } from '../api'
import { StateBadge } from './overlayUi'
import { modeLabel, typeLabel } from './overlayText'
import { Bases } from './ui'

/** One tunnel: type, endpoints, state, why, its paths and its evidence. */
export function TunnelCard({ t }: { t: Tunnel }) {
  return (
    <li
      data-tunnel-row={t.id}
      data-state={t.state}
      data-tunnel-type={t.type}
      className="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">{typeLabel[t.type]}</span>
        <span>
          {t.a.name} <span aria-hidden>↔</span>
          <span className="sr-only"> and </span> {t.b.name}
        </span>
        <StateBadge state={t.state} />
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {modeLabel(t.mode)}
          {t.role ? ` · ${t.role}` : ''}
        </span>
      </div>
      <p className="mt-1 text-xs text-slate-700 dark:text-slate-200" data-tunnel-reason>
        {t.reason}
      </p>
      {t.paths.length > 0 && (
        <ul className="mt-2 space-y-0.5 text-xs text-slate-600 dark:text-slate-300">
          {t.paths.map((p, i) => (
            <li key={i} className="flex flex-wrap items-center gap-2">
              <StateBadge state={p.state} />
              <span>
                {p.local_link} to {p.remote}
              </span>
              {p.reason && <span className="text-slate-500 dark:text-slate-400">{p.reason}</span>}
            </li>
          ))}
        </ul>
      )}
      {t.notes?.map((n, i) => (
        <p key={i} className="mt-1 text-xs text-slate-600 dark:text-slate-300">
          {n.text}
        </p>
      ))}
      <div className="mt-2 flex flex-wrap items-center gap-1">
        <Bases basis={t.basis} also={t.also} />
        {t.not_modelled?.map((n) => (
          <Bases key={n.rule} basis={n.basis} />
        ))}
      </div>
    </li>
  )
}

export default function TunnelList({ tunnels }: { tunnels: Tunnel[] }) {
  if (tunnels.length === 0) return <p className="text-sm text-slate-500">No tunnels match the filters.</p>
  return (
    <ul className="space-y-2" data-tunnel-list>
      {tunnels.map((t) => (
        <TunnelCard key={t.id} t={t} />
      ))}
    </ul>
  )
}
