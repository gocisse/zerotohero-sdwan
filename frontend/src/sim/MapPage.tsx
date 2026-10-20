import { useMemo, useState } from 'react'
import type { TunnelType } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import ConditionsPanel from './ConditionsPanel'
import { MapLegend } from './TunnelMap'
import TunnelMap from './TunnelMap'
import TunnelList from './TunnelList'
import { useTopologyCtx } from './topologyContext'
import { useTunnels } from './useTunnels'
import { NotModelledNote } from './ui'
import { typeLabel } from './overlayText'

const ALL: TunnelType[] = ['gateway', 'hub', 'dynamic', 'nsd-gateway', 'nsd-edge']

/** The topology map with the tunnels that exist now, the Conditions panel and the tunnel list. */
export default function MapPage() {
  const { detail, topologyId, conditions } = useTopologyCtx()
  const state = useTunnels(topologyId, conditions)
  const [types, setTypes] = useState<Set<TunnelType>>(new Set(['hub', 'dynamic', 'nsd-gateway', 'nsd-edge']))
  const [secondary, setSecondary] = useState(false)
  const [potential, setPotential] = useState(false)
  const [listType, setListType] = useState<'all' | TunnelType>('all')
  const [existingOnly, setExistingOnly] = useState(true)

  const tunnels = useMemo(() => state.data?.tunnels ?? [], [state.data])
  const present = useMemo(() => new Set(tunnels.map((t) => t.type)), [tunnels])
  const rows = tunnels.filter(
    (t) =>
      (listType === 'all' || t.type === listType) &&
      (!existingOnly || (t.state !== 'not-established' && t.state !== 'torn-down')),
  )
  const unreachable = {
    edges: conditions.hubs_unreachable ?? [],
    gateways: conditions.gateways_unreachable ?? [],
    nsds: conditions.nsds_unreachable ?? [],
  }

  return (
    <div>
      <p className="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
        Which tunnels exist in this network, and why. Use the Conditions to break a link or a Hub and watch the tunnels
        change. Dynamic tunnels appear after traffic: add some in the panel.
      </p>
      {state.error && (
        <div className="mt-3">
          <ErrorPanel title="The tunnels could not be computed" error={state.error} />
        </div>
      )}
      {!state.data && !state.error && <p className="mt-4 text-slate-500">Loading…</p>}
      {state.data && (
        <div className="mt-4 space-y-5">
          <div className="min-w-0 space-y-3">
            <fieldset className="flex min-w-0 flex-wrap gap-x-4 gap-y-1 text-sm">
              <legend className="sr-only">Tunnels shown on the map</legend>
              {ALL.filter((x) => present.has(x)).map((x) => (
                <label key={x} className="flex items-center gap-1.5">
                  <input
                    type="checkbox"
                    checked={types.has(x)}
                    onChange={(e) => {
                      const n = new Set(types)
                      if (e.target.checked) n.add(x)
                      else n.delete(x)
                      setTypes(n)
                    }}
                    data-map-layer={x}
                  />
                  {typeLabel[x]}
                </label>
              ))}
              <label className="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={secondary}
                  onChange={(e) => setSecondary(e.target.checked)}
                  data-map-secondary
                />
                Secondary Gateway tunnels
              </label>
              <label className="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={potential}
                  onChange={(e) => setPotential(e.target.checked)}
                  data-map-potential
                />
                Potential tunnels
              </label>
            </fieldset>
            <TunnelMap
              topology={detail.topology}
              tunnels={tunnels}
              types={types}
              secondaryGateway={secondary}
              showPotential={potential}
              unreachable={unreachable}
            />
            <MapLegend />
            <p className="text-xs text-slate-500 dark:text-slate-400 lg:hidden">
              On a narrow screen the map scrolls sideways inside its box.
            </p>
          </div>
          <ConditionsPanel wide />
        </div>
      )}

      {state.data && (
        <section className="mt-8" aria-labelledby="tun-h">
          <h2 id="tun-h" className="text-lg font-semibold">
            Tunnels{' '}
            <span className="text-sm font-normal text-slate-500 dark:text-slate-400" data-tunnel-count>
              ({rows.length})
            </span>
          </h2>
          <div className="mt-2 flex flex-wrap items-center gap-4 text-sm">
            <label className="flex items-center gap-2">
              Type
              <select
                className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-900"
                value={listType}
                onChange={(e) => setListType(e.target.value as 'all' | TunnelType)}
                data-list-type
              >
                <option value="all">all</option>
                {ALL.filter((x) => present.has(x)).map((x) => (
                  <option key={x} value={x}>
                    {typeLabel[x]}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="checkbox"
                checked={existingOnly}
                onChange={(e) => setExistingOnly(e.target.checked)}
                data-list-existing
              />
              Hide tunnels that are not established or were torn down
            </label>
          </div>
          <div className="mt-3">
            <TunnelList tunnels={rows} />
          </div>
          {state.data.not_modelled.length > 0 && (
            <div className="mt-4 space-y-2">
              {state.data.not_modelled.map((n) => (
                <NotModelledNote key={n.rule} nm={n} />
              ))}
            </div>
          )}
          {state.data.cluster_assignments.length > 0 && (
            <div className="mt-4 text-sm" data-cluster-assignments>
              <h3 className="font-semibold">Cluster assignment</h3>
              <ul className="mt-1 list-disc pl-5">
                {state.data.cluster_assignments.map((c) => (
                  <li key={c.spoke}>
                    {c.spoke} (logical ID {c.logical_id}) is assigned to {c.member}: {c.logical_id} modulo {c.members}.
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>
      )}
    </div>
  )
}
