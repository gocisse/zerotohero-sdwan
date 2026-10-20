import { useMemo, useState } from 'react'
import { postWhatIf, type Change, type ChangeType, type EdgeDiff, type WhatIfResult } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useTopologyCtx } from './topologyContext'
import { Bases, BasisBadge, NotModelledNote, SourceBadge } from './ui'
import { diffValue } from './rules'

const TYPES: { id: ChangeType; label: string }[] = [
  { id: 'set_profile_setting', label: 'Change a setting in a Profile' },
  { id: 'remove_profile_setting', label: 'Remove a setting from a Profile' },
  { id: 'set_edge_override', label: 'Add an override on an Edge' },
  { id: 'remove_edge_override', label: 'Remove an override from an Edge' },
  { id: 'assign_profile', label: 'Move an Edge to another Profile' },
]

function parseValue(text: string): unknown {
  const t = text.trim()
  if (t === 'true') return true
  if (t === 'false') return false
  if (t !== '' && !Number.isNaN(Number(t))) return Number(t)
  return t
}

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'

const statusStyle: Record<EdgeDiff['status'], string> = {
  changed: 'border-sky-400 bg-sky-50 text-sky-900 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-100',
  'kept-override':
    'border-amber-400 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100',
  unchanged: 'border-slate-300 bg-slate-100 text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200',
}
const statusText: Record<EdgeDiff['status'], string> = {
  changed: 'changed',
  'kept-override': 'keeps its override',
  unchanged: 'no change',
}

function EdgeDiffCard({ d }: { d: EdgeDiff }) {
  return (
    <li
      data-diff-edge={d.edge_id}
      data-status={d.status}
      className="rounded-lg border border-slate-200 p-4 dark:border-slate-800"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="font-semibold">{d.edge_name}</h3>
        <span className={`rounded-full border px-2 py-0.5 text-xs font-medium ${statusStyle[d.status]}`}>
          {statusText[d.status]}
        </span>
      </div>
      {d.items.length > 0 && (
        <ul className="mt-3 space-y-3" data-diff-items>
          {d.items.map((it) => (
            <li key={it.key} data-diff-item={it.key} className="text-sm">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{it.label}</span>
                <BasisBadge basis={it.basis} />
              </div>
              <p className="mt-1 [overflow-wrap:anywhere]">
                <span className="text-slate-500 line-through dark:text-slate-400">
                  {diffValue(it.kind, it.before, 'before')}
                </span>
                <span aria-hidden> → </span>
                <span className="sr-only"> becomes </span>
                <span className="font-medium">{diffValue(it.kind, it.after, 'after')}</span>
              </p>
              <div className="mt-1 flex flex-wrap items-center gap-2 text-xs">
                {it.before_source && <SourceBadge source={it.before_source} />}
                {it.before_source && it.after_source && <span aria-hidden>→</span>}
                {it.after_source && <SourceBadge source={it.after_source} />}
              </div>
            </li>
          ))}
        </ul>
      )}
      {d.notes.length > 0 && (
        <ul className="mt-3 space-y-2" data-diff-notes>
          {d.notes.map((n, i) => (
            <li key={i} className="flex flex-wrap items-start gap-2 text-xs text-slate-700 dark:text-slate-200">
              <Bases basis={n.basis} />
              <span className="min-w-0 flex-1">{n.text}</span>
            </li>
          ))}
        </ul>
      )}
    </li>
  )
}

/** "What if?": pick a hypothetical change and see the effective-configuration diff for every Edge. */
export default function WhatIfPage() {
  const { detail, topologyId } = useTopologyCtx()
  const t = detail.topology
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])
  const settings = detail.settings.filter((s) => s.section !== 'Firewall' || s.id === 'firewall.stateful_firewall')

  const [type, setType] = useState<ChangeType>('set_profile_setting')
  const [profile, setProfile] = useState(t.profiles[0]?.id ?? '')
  const [edge, setEdge] = useState(edges[0]?.id ?? '')
  const [setting, setSetting] = useState(settings[0]?.id ?? '')
  const [value, setValue] = useState('')
  const [result, setResult] = useState<WhatIfResult | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const [busy, setBusy] = useState(false)

  const def = settings.find((s) => s.id === setting)
  const edgeObj = edges.find((e) => e.id === edge)
  const overrideIds = Object.keys(edgeObj?.overrides.device ?? {})
  const needsProfile = type === 'set_profile_setting' || type === 'remove_profile_setting' || type === 'assign_profile'
  const needsEdge = type === 'set_edge_override' || type === 'remove_edge_override' || type === 'assign_profile'
  const needsSetting = type !== 'assign_profile'
  const needsValue = type === 'set_profile_setting' || type === 'set_edge_override'
  const settingChoices = type === 'remove_edge_override' ? settings.filter((s) => overrideIds.includes(s.id)) : settings
  const effectiveSetting = settingChoices.some((s) => s.id === setting) ? setting : (settingChoices[0]?.id ?? '')
  const effectiveDef = settings.find((s) => s.id === effectiveSetting) ?? def

  const run = async () => {
    const change: Change = { type }
    if (needsProfile) change.profile = profile
    if (needsEdge) change.edge = edge
    if (needsSetting) change.setting = effectiveSetting
    if (needsValue) change.value = effectiveDef?.kind === 'bool' ? value !== 'off' : parseValue(value)
    setBusy(true)
    setError(null)
    try {
      setResult(await postWhatIf(topologyId, change))
    } catch (e) {
      setResult(null)
      setError(e as Error)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <p className="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
        Nothing is saved. Pick a hypothetical change and see which Edges change, which keep their override, and what a
        Profile switch takes with it.
      </p>
      <form
        className="mt-4 grid max-w-3xl gap-4 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault()
          void run()
        }}
        data-whatif-form
      >
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium">Change</span>
          <select
            className={field}
            value={type}
            onChange={(e) => setType(e.target.value as ChangeType)}
            data-whatif-type
          >
            {TYPES.map((x) => (
              <option key={x.id} value={x.id}>
                {x.label}
              </option>
            ))}
          </select>
        </label>
        {needsProfile && (
          <label className="block text-sm">
            <span className="font-medium">{type === 'assign_profile' ? 'Move to Profile' : 'Profile'}</span>
            <select className={field} value={profile} onChange={(e) => setProfile(e.target.value)} data-whatif-profile>
              {t.profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </label>
        )}
        {needsEdge && (
          <label className="block text-sm">
            <span className="font-medium">Edge</span>
            <select className={field} value={edge} onChange={(e) => setEdge(e.target.value)} data-whatif-edge>
              {edges.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
            </select>
          </label>
        )}
        {needsSetting && (
          <label className="block text-sm">
            <span className="font-medium">Setting</span>
            <select
              className={field}
              value={effectiveSetting}
              onChange={(e) => setSetting(e.target.value)}
              data-whatif-setting
              disabled={settingChoices.length === 0}
            >
              {settingChoices.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.label}
                </option>
              ))}
            </select>
            {settingChoices.length === 0 && (
              <span className="mt-1 block text-xs text-slate-500">This Edge has no overrides.</span>
            )}
          </label>
        )}
        {needsValue && (
          <label className="block text-sm">
            <span className="font-medium">New value</span>
            {effectiveDef?.kind === 'bool' ? (
              <select
                className={field}
                value={value === 'off' ? 'off' : 'on'}
                onChange={(e) => setValue(e.target.value)}
                data-whatif-value
              >
                <option value="on">on</option>
                <option value="off">off</option>
              </select>
            ) : (
              <input
                className={field}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="a number, on, off or text"
                data-whatif-value
              />
            )}
            <span className="mt-1 block text-xs text-slate-500 dark:text-slate-400">
              Values are opaque here: their meaning is not modelled (assumption 33).
            </span>
          </label>
        )}
        <div className="sm:col-span-2">
          <button
            type="submit"
            disabled={busy || (needsSetting && settingChoices.length === 0)}
            className="rounded-md bg-sky-700 px-4 py-2 text-sm font-semibold text-white hover:bg-sky-800 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none disabled:opacity-50"
            data-whatif-run
          >
            {busy ? 'Working…' : 'Show what changes'}
          </button>
        </div>
      </form>

      <div aria-live="polite" className="mt-6">
        {error && <ErrorPanel title="That change cannot be applied" error={error} />}
        {result && (
          <div data-whatif-result>
            <h2 className="text-lg font-semibold">{result.summary}</h2>
            {result.issues.length > 0 && (
              <ul className="mt-2 space-y-1 text-sm" data-whatif-issues>
                {result.issues.map((i, k) => (
                  <li
                    key={k}
                    className={
                      i.level === 'error' ? 'text-red-700 dark:text-red-300' : 'text-amber-800 dark:text-amber-200'
                    }
                  >
                    {i.level === 'error' ? 'The changed topology would be invalid' : 'Warning'}: <code>{i.path}</code>:{' '}
                    {i.message}
                  </li>
                ))}
              </ul>
            )}
            <ul className="mt-4 space-y-3" data-diff-list>
              {result.edges.map((d) => (
                <EdgeDiffCard key={d.edge_id} d={d} />
              ))}
            </ul>
            {(result.not_modelled ?? []).length > 0 && (
              <div className="mt-4 space-y-2">
                {(result.not_modelled ?? []).map((n) => (
                  <NotModelledNote key={n.rule} nm={n} />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
