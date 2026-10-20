import { useMemo, useState } from 'react'
import { postPath, postTunnels, type Basis, type Conditions, type Flow, type TunnelSet } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { useTopologyCtx } from './topologyContext'
import { useAsync } from './useAsync'
import { Bases } from './ui'

// The engine's own reason set, in the words a learner would use.
const REASON_TEXT: Record<string, string> = {
  cloud_vpn_disabled: 'Cloud VPN is not enabled for these Edges',
  branch_to_branch_disabled: 'Branch to Branch VPN is not enabled',
  dynamic_disabled: 'Dynamic Branch to Branch is off for one of the two Edges',
  no_traffic_yet: 'No traffic has flowed between them yet, so no direct tunnel was built',
  torn_down_after_silence: 'There were 180 seconds or more of silence, so the direct tunnel was torn down',
  nat_blocks_direct: 'A NAT in front of the peer blocks incoming UDP 2426, so the direct tunnel does not come up',
  peer_unreachable: 'The preferred peer (Hub or Gateway) is unreachable',
  build_decision_never: 'The build decision was set to never',
  branch_to_branch_one_site: 'Branch to Branch VPN is enabled at only one of the two sites',
}
const DISTRACTORS = [
  'cloud_vpn_disabled',
  'branch_to_branch_disabled',
  'dynamic_disabled',
  'no_traffic_yet',
  'torn_down_after_silence',
  'nat_blocks_direct',
]

type Kind = 'direct' | 'hub'

interface Question {
  kind: Kind
  prompt: string
  context: string
  correct: string
  options: string[]
  explanation: string
  basis: Basis
  also: Basis[]
}

const VARIANTS = [
  { id: 'none', context: 'Nobody has sent traffic between them yet.', ago: null as number | null },
  { id: 'recent', context: 'Traffic between them has been flowing for the last minute.', ago: 5 },
  { id: 'stale', context: 'They last exchanged traffic 200 seconds ago.', ago: 200 },
]

function pick<T>(list: T[], n: number, seed: () => number): T[] {
  const copy = [...list]
  const out: T[] = []
  while (out.length < n && copy.length > 0) out.push(copy.splice(Math.floor(seed() * copy.length), 1)[0]!)
  return out
}

function options(correct: string, seed: () => number): string[] {
  const others = DISTRACTORS.filter((d) => d !== correct)
  return pick([correct, ...pick(others, 3, seed)], 4, seed).sort((a, b) => a.localeCompare(b))
}

/**
 * Overlay challenge: "Why is there no direct tunnel between A and B?" and "Why does this flow go through the Hub?".
 * The answers come from the engine's own reason set. Ungraded, nothing is stored, never blocks anything.
 */
export default function OverlayChallengePage() {
  const { detail, topologyId } = useTopologyCtx()
  const t = detail.topology
  const edges = useMemo(() => t.sites.flatMap((s) => s.edges), [t])
  const [kind, setKind] = useState<Kind>('direct')

  const data = useAsync(async () => {
    const name = (id: string) => edges.find((e) => e.id === id)?.name ?? id
    const traffic = (ago: number | null): Conditions =>
      ago === null
        ? {}
        : {
            traffic: edges.flatMap((a, i) =>
              edges
                .slice(i + 1)
                .map((b) => ({ a: a.id, b: b.id, first_packet_ago_s: ago + 60, last_packet_ago_s: ago })),
            ),
          }
    const sets: { variant: (typeof VARIANTS)[number]; ts: TunnelSet }[] = []
    for (const v of VARIANTS) sets.push({ variant: v, ts: await postTunnels(topologyId, traffic(v.ago)) })
    // Flows between Edges whose Profile sends branch-to-branch traffic through the Hubs.
    const viaHub = edges.filter((e) => {
      const p = t.profiles.find((x) => x.id === e.profile)
      return p?.cloud_vpn?.branch_to_branch?.enabled && p.cloud_vpn.branch_to_branch.via === 'hubs_for_vpn'
    })
    const flows: { flow: Flow; conditions: Conditions; context: string; hub?: string }[] = []
    for (let i = 0; i < viaHub.length; i++)
      for (let j = i + 1; j < viaHub.length; j++) {
        const a = viaHub[i]!.id
        const b = viaHub[j]!.id
        for (const v of VARIANTS)
          flows.push({
            flow: { source: a, destination: { kind: 'edge', id: b }, application: '' },
            conditions: traffic(v.ago),
            context: v.context,
          })
        const hubs = t.profiles.find((x) => x.id === viaHub[i]!.profile)?.cloud_vpn?.branch_to_hub?.hubs ?? []
        if (hubs[0] && edges.some((e) => e.id === hubs[0]))
          flows.push({
            flow: { source: a, destination: { kind: 'edge', id: b } },
            conditions: { hubs_unreachable: [hubs[0]] },
            context: `Hub ${name(hubs[0])} is unreachable.`,
            hub: hubs[0],
          })
      }
    const decisions = await Promise.all(
      flows.slice(0, 24).map((f) => postPath(topologyId, f.flow, f.conditions).then((d) => ({ f, d }))),
    )
    return { sets, decisions }
  }, `oc/${topologyId}`)

  const pool = useMemo<Question[]>(() => {
    if (data.status !== 'ready') return []
    const out: Question[] = []
    const rnd = Math.random
    const name = (id: string) => edges.find((e) => e.id === id)?.name ?? id
    if (kind === 'direct') {
      for (const { variant, ts } of data.data.sets)
        for (const tn of ts.tunnels) {
          if (tn.type !== 'dynamic' || (tn.state !== 'not-established' && tn.state !== 'torn-down')) continue
          out.push({
            kind: 'direct',
            prompt: `Why is there no direct tunnel between ${tn.a.name} and ${tn.b.name}?`,
            context: variant.context,
            correct: tn.reason_code,
            options: options(tn.reason_code, rnd),
            explanation: tn.reason,
            basis: tn.basis,
            also: tn.also ?? [],
          })
        }
    } else {
      for (const { f, d } of data.data.decisions) {
        if (!d.chosen) continue
        const first = d.options[0]
        if (f.hub) {
          // The primary Hub is down: the flow uses another Hub.
          if (d.chosen.kind !== 'via-hub') continue
          out.push({
            kind: 'hub',
            prompt: `${name(d.flow.source)}'s flow to ${name(d.flow.destination.id)} goes through ${d.chosen.label.replace('Via Hub ', '')}. Why not through the primary Hub?`,
            context: f.context,
            correct: 'peer_unreachable',
            options: options('peer_unreachable', rnd),
            explanation: d.options.find((o) => !o.usable && o.kind === 'via-hub')?.reason ?? d.why,
            basis: d.chosen.basis,
            also: d.chosen.also ?? [],
          })
        } else if (d.chosen.kind === 'via-hub' && first && first.kind === 'direct' && !first.usable) {
          out.push({
            kind: 'hub',
            prompt: `${name(d.flow.source)}'s flow to ${name(d.flow.destination.id)} goes through the Hub (Branch to Branch is set to Hubs for VPN). Why is it not using a direct tunnel?`,
            context: f.context,
            correct: first.reason_code,
            options: options(first.reason_code, rnd),
            explanation: first.reason,
            basis: first.basis,
            also: first.also ?? [],
          })
        }
      }
    }
    return out
  }, [data, kind, edges])

  const [seed] = useState(() => Math.random())
  const [chosen, setChosen] = useState<{ pool: Question[]; q: Question } | null>(null)
  const current = chosen && chosen.pool === pool ? chosen.q : (pool[Math.floor(seed * pool.length)] ?? null)
  const [picked, setPicked] = useState<string | null>(null)

  const next = () => {
    if (pool.length === 0) return
    const choices = pool.length > 1 ? pool.filter((q) => q !== current) : pool
    const q = choices[Math.floor(Math.random() * choices.length)]
    if (q) setChosen({ pool, q })
    setPicked(null)
  }

  if (data.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (data.status === 'error') return <ErrorPanel title="The overlay challenge is unavailable" error={data.error} />

  return (
    <div className="max-w-2xl" data-overlay-challenge>
      <p className="text-sm text-slate-600 dark:text-slate-300">
        A practice question about tunnels and paths, answered from the simulator's own reasons. It is not graded,
        nothing is saved, and you can skip it at any time.
      </p>
      <label className="mt-3 block text-sm">
        <span className="font-medium">Question</span>
        <select
          className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900"
          value={kind}
          onChange={(e) => {
            setKind(e.target.value as Kind)
            setPicked(null)
            setChosen(null)
          }}
          data-oc-kind
        >
          <option value="direct">Why is there no direct tunnel between A and B?</option>
          <option value="hub">Why does this flow go through the Hub?</option>
        </select>
      </label>
      {!current ? (
        <p className="mt-4 text-sm text-slate-500">
          This topology has no question of this kind (it needs Edges that send branch-to-branch traffic through Hubs, or
          Edges with dynamic tunnels).
        </p>
      ) : (
        <>
          <fieldset
            className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
            disabled={picked !== null}
          >
            <legend className="px-1 text-base font-semibold" data-oc-question>
              {current.prompt}
            </legend>
            <p className="mb-3 text-sm text-slate-600 dark:text-slate-300" data-oc-context>
              {current.context}
            </p>
            <div className="space-y-2">
              {current.options.map((o) => (
                <label
                  key={o}
                  className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-200 p-2 text-sm has-[:checked]:border-sky-500 dark:border-slate-700"
                >
                  <input
                    type="radio"
                    name="oc"
                    value={o}
                    checked={picked === o}
                    onChange={() => setPicked(o)}
                    className="mt-1"
                    data-oc-choice={o}
                  />
                  <span>{REASON_TEXT[o] ?? o}</span>
                </label>
              ))}
            </div>
          </fieldset>
          {picked !== null && (
            <div
              role="status"
              data-oc-result={picked === current.correct ? 'correct' : 'incorrect'}
              className={`mt-4 rounded-lg border p-4 text-sm ${
                picked === current.correct
                  ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-100'
                  : 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100'
              }`}
            >
              <p className="font-semibold">{picked === current.correct ? 'Correct.' : 'Not quite.'}</p>
              <p className="mt-1" data-oc-explanation>
                The reason is: {REASON_TEXT[current.correct] ?? current.correct}. {current.explanation}
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-1">
                <Bases basis={current.basis} also={current.also} />
              </div>
            </div>
          )}
          <div className="mt-4">
            <button
              type="button"
              onClick={next}
              className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700"
              data-oc-next
            >
              {picked !== null ? 'Next question' : 'Skip'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}
