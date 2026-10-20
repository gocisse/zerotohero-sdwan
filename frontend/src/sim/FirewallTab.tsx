import type { EffectiveConfig, EffectiveFirewallRule } from '../api'
import { fwMatchText } from './rules'
import { SettingRow } from './DeviceTab'
import { Bases, BasisBadge, NotModelledNote, SourceBadge } from './ui'

const actionStyle: Record<string, string> = {
  allow: 'border-emerald-300 text-emerald-900 dark:border-emerald-800 dark:text-emerald-100',
  drop: 'border-rose-300 text-rose-900 dark:border-rose-800 dark:text-rose-100',
  reject: 'border-rose-300 text-rose-900 dark:border-rose-800 dark:text-rose-100',
  skip: 'border-slate-300 text-slate-800 dark:border-slate-600 dark:text-slate-100',
}

function Rule({ r }: { r: EffectiveFirewallRule }) {
  return (
    <li
      data-fw-rule={r.rule.id}
      data-fw-group={r.group}
      className="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">{r.rule.name ?? r.rule.id}</span>
        <code className="text-xs text-slate-500 dark:text-slate-400">{r.rule.id}</code>
        <span
          className={`rounded-full border px-2 py-0.5 text-xs font-semibold uppercase ${actionStyle[r.rule.action] ?? ''}`}
        >
          {r.rule.action}
        </span>
        <SourceBadge source={r.owner} />
        <Bases basis={r.basis} also={r.also} />
      </div>
      <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">Match: {fwMatchText(r.rule)}</p>
    </li>
  )
}

/** The Firewall tab: the Override checkbox, Stateful Firewall, and the rules with Edge override rules first. */
export default function FirewallTab({ ec }: { ec: EffectiveConfig }) {
  const fw = ec.firewall
  const edgeRules = fw.rules.filter((r) => r.group === 'edge-override')
  const profileRules = fw.rules.filter((r) => r.group === 'profile')
  return (
    <div className="space-y-6">
      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={fw.override} disabled readOnly aria-label="Override (read-only)" />
        <span className="font-medium">Override</span>
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {fw.override ? 'This Edge has its own firewall rules.' : "This Edge inherits its Profile's firewall rules."}
        </span>
      </label>
      <ul>
        <SettingRow s={fw.stateful_firewall} />
      </ul>
      <NotModelledNote nm={fw.evaluation_order} />
      <section aria-labelledby="fw-edge" data-fw-group-section="edge-override">
        <h3 id="fw-edge" className="flex flex-wrap items-center gap-2 text-base font-semibold">
          Edge override rules <BasisBadge basis={fw.group_priority} />
        </h3>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Edge override rules take priority over the rules inherited from the Profile. Within a group the rules are
          shown in authored order (a listing, not an evaluation order: assumption 37).
        </p>
        <ul className="mt-2 space-y-2">
          {edgeRules.map((r) => (
            <Rule key={r.rule.id} r={r} />
          ))}
          {edgeRules.length === 0 && <li className="text-sm text-slate-500">No Edge override rules.</li>}
        </ul>
      </section>
      <section aria-labelledby="fw-profile" data-fw-group-section="profile">
        <h3 id="fw-profile" className="text-base font-semibold">
          Rules from the Profile
        </h3>
        <ul className="mt-2 space-y-2">
          {profileRules.map((r) => (
            <Rule key={r.rule.id} r={r} />
          ))}
          {profileRules.length === 0 && <li className="text-sm text-slate-500">No Profile rules.</li>}
        </ul>
      </section>
    </div>
  )
}
