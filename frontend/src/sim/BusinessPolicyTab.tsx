import { useState } from 'react'
import type { EffectiveConfig, EffectiveRule } from '../api'
import { actionText, matchText } from './rules'
import { Bases, NotModelledNote, SourceBadge } from './ui'

function RuleCard({ r }: { r: EffectiveRule }) {
  const dim = r.superseded_by ? 'opacity-70' : ''
  return (
    <li
      data-rule={r.rule.id}
      data-superseded={r.superseded_by ? 'true' : 'false'}
      className={`rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800 ${dim}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className={`font-medium ${r.superseded_by ? 'line-through' : ''}`}>{r.rule.name ?? r.rule.id}</span>
        <code className="text-xs text-slate-500 dark:text-slate-400">{r.rule.id}</code>
        <SourceBadge source={r.owner} />
        <Bases basis={r.basis} also={r.also} />
      </div>
      <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">Match: {matchText(r.rule.match)}</p>
      <p className="text-xs text-slate-600 dark:text-slate-300">
        Action: {actionText(r.rule.action, r.effective_network_service, r.effective_steering_mode)}
      </p>
      {r.superseded_by && (
        <p className="mt-1 text-xs font-medium text-amber-800 dark:text-amber-200">
          Superseded by the Edge rule {r.superseded_by} (same match; Profile rules cannot be edited on the Edge).
        </p>
      )}
      {r.supersedes && r.supersedes.length > 0 && (
        <p className="mt-1 text-xs font-medium text-amber-800 dark:text-amber-200">
          Supersedes the Profile rule {r.supersedes.join(', ')}.
        </p>
      )}
      {r.rewritten && (
        <p className="mt-1 text-xs text-slate-700 dark:text-slate-200">
          This segment is Private, so Internet Multipath is shown as Direct (the Orchestrator changes it automatically).
        </p>
      )}
    </li>
  )
}

/** The Business Policy tab: a segment drop-down, "Rules From Profile" and "Edge Overrides" (names as in the real tab). */
export default function BusinessPolicyTab({ ec }: { ec: EffectiveConfig }) {
  const [segment, setSegment] = useState('global')
  const sp = ec.business_policy.find((s) => s.segment.id === segment) ?? ec.business_policy[0]
  if (!sp) return <p className="text-sm text-slate-500">This Edge has no segments.</p>
  const label = (s: { name: string; type: string }) => `${s.name} [${s.type === 'private' ? 'Private' : 'Regular'}]`
  return (
    <div className="space-y-5">
      <label className="block text-sm">
        <span className="font-medium">Segment</span>
        <select
          value={sp.segment.id}
          onChange={(e) => setSegment(e.target.value)}
          className="mt-1 block w-full max-w-sm rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900"
        >
          {ec.business_policy.map((s) => (
            <option key={s.segment.id} value={s.segment.id}>
              {label(s.segment)}
            </option>
          ))}
        </select>
      </label>
      <NotModelledNote nm={sp.precedence} />
      <section aria-labelledby="bp-edge" data-bp-section="edge-overrides">
        <h3 id="bp-edge" className="text-base font-semibold">
          Edge Overrides
        </h3>
        <ul className="mt-2 space-y-2">
          {sp.edge_overrides.map((r) => (
            <RuleCard key={r.rule.id} r={r} />
          ))}
          {sp.edge_overrides.length === 0 && <li className="text-sm text-slate-500">No Edge rules in this segment.</li>}
        </ul>
      </section>
      <section aria-labelledby="bp-profile" data-bp-section="from-profile">
        <h3 id="bp-profile" className="text-base font-semibold">
          Rules From Profile
        </h3>
        <ul className="mt-2 space-y-2">
          {sp.from_profile.map((r) => (
            <RuleCard key={r.rule.id} r={r} />
          ))}
          {sp.from_profile.length === 0 && (
            <li className="text-sm text-slate-500">No Profile rules in this segment.</li>
          )}
        </ul>
      </section>
    </div>
  )
}
