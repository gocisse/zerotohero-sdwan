import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import ErrorPanel from '../components/ErrorPanel'
import { AssumptionsPanel, RoutingConditions } from './RoutingControls'
import { addressOf, netmask, sortPrefixes, sourceText } from './routingText'
import { withLab } from './labParams'
import { useTopologyCtx } from './topologyContext'
import { useRouteSet } from './useRouteSet'
import { Bases, BasisBadge, NotModelledNote } from './ui'

const field =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-900'
const th = 'px-2 py-1.5 text-left text-xs font-semibold whitespace-nowrap'
const thSim = `${th} bg-orange-50 text-orange-950 dark:bg-orange-950/30 dark:text-orange-100`
const td = 'px-2 py-1.5 align-top text-xs'

/**
 * A mock of the Remote Diagnostics test "Route Table Dump": the columns are the ones the Troubleshooting Guide lists.
 * Columns the model cannot fill (Type, Order, Next Hop Name) are left out; columns the simulator adds are marked.
 */
export default function RoutesPage() {
  const { detail, topologyId, labSearch } = useTopologyCtx()
  const { data, error, busy } = useRouteSet()
  const edges = useMemo(() => detail.topology.sites.flatMap((s) => s.edges), [detail])
  const [edgeId, setEdgeId] = useState(edges[0]?.id ?? '')
  const [segment, setSegment] = useState('')
  const [preferredOnly, setPreferredOnly] = useState(false)
  const [prefix, setPrefix] = useState('')
  const bases = detail.ui_bases

  if (error && !data) return <ErrorPanel title="Routes are unavailable" error={error} />
  if (!data) return <p className="text-slate-500">Loading…</p>
  const er = data.edges.find((e) => e.edge === edgeId) ?? data.edges[0]
  const segs = er?.segments.map((s) => s.segment) ?? []
  const seg = er?.segments.find((s) => s.segment === segment) ?? er?.segments[0]
  const rows = (seg?.routes ?? []).filter(
    (r) => (!preferredOnly || r.selected) && (!prefix || r.prefix === prefix.trim()),
  )
  const prefixes = [...new Set((seg?.routes ?? []).map((r) => r.prefix))].sort(sortPrefixes)

  return (
    <div data-routes-page>
      {error && <ErrorPanel title="The last update failed" error={error} />}
      <section className="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
        <h2 className="text-base font-semibold">Remote Diagnostics &gt; Route Table Dump (mock)</h2>
        <p className="mt-1 text-sm text-slate-700 dark:text-slate-200">
          In the product the per-Edge route table is the Diagnostics test <strong>Route Table Dump</strong> (Diagnostics
          tab, choose the Edge, pick the test, Run). Monitor &gt; Edges has no per-Edge routes tab in the documentation
          read, and Monitor &gt; Routing has the tabs Multicast Groups, PIM Neighbors, BGP Edge Neighbor State, BFD, BGP
          Gateway Neighbor State and Gateway Route Table.{' '}
          {bases.route_table_dump && (
            <BasisBadge
              basis={bases.route_table_dump}
              extra={[bases.no_edge_routes_tab, bases.routing_monitor_tabs].filter(
                (b): b is NonNullable<typeof b> => !!b,
              )}
            />
          )}
        </p>
        {data.not_modelled
          .filter((n) => n.rule === 'nm-rt-gateway-reflector')
          .map((n) => (
            <p key={n.rule} className="mt-1 text-xs text-slate-600 dark:text-slate-300" data-gateway-note>
              Not modelled: Gateway as route reflector. {n.reason} <BasisBadge basis={n.basis} />
            </p>
          ))}
        {bases.gateway_route_table && (
          <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
            The Gateway Route Table tab is not built: its columns were not read.{' '}
            <BasisBadge basis={bases.gateway_route_table} />
          </p>
        )}
        <div className="mt-3 grid gap-3 sm:grid-cols-4">
          <label className="text-sm">
            <span className="font-medium">Edge</span>
            <select
              className={field}
              value={er?.edge ?? ''}
              onChange={(e) => setEdgeId(e.target.value)}
              data-routes-edge
            >
              {data.edges.map((e) => (
                <option key={e.edge} value={e.edge}>
                  {e.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Segment</span>
            <select
              className={field}
              value={seg?.segment ?? ''}
              onChange={(e) => setSegment(e.target.value)}
              data-routes-segment
            >
              {segs.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Prefix</span>
            <select className={field} value={prefix} onChange={(e) => setPrefix(e.target.value)} data-routes-prefix>
              <option value="">all prefixes</option>
              {prefixes.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="font-medium">Routes</span>
            <select
              className={field}
              value={preferredOnly ? 'preferred' : 'all'}
              onChange={(e) => setPreferredOnly(e.target.value === 'preferred')}
              data-routes-filter
            >
              <option value="all">all</option>
              <option value="preferred">preferred only</option>
            </select>
          </label>
        </div>
        <p className="mt-2 text-xs text-slate-500 dark:text-slate-400" aria-live="polite">
          {busy ? 'Updating…' : `${rows.length} route${rows.length === 1 ? '' : 's'}.`} Shaded column headers are added
          by the simulator; the other headers are the documented columns. Columns the model cannot fill (Type, Order,
          Next Hop Name) are not shown.
        </p>
        <div className="mt-2 overflow-x-auto" tabIndex={0} role="region" aria-label="Route table">
          <table className="min-w-full border-collapse text-left" data-route-table>
            <thead>
              <tr className="border-b border-slate-300 dark:border-slate-700">
                <th className={th}>Address</th>
                <th className={th}>Netmask</th>
                <th className={th}>Segment</th>
                <th className={th}>Cost</th>
                <th className={th}>Preference</th>
                <th className={th}>Reachable</th>
                <th className={th}>Next Hop</th>
                <th className={th}>Destination Name</th>
                <th className={th}>Lost Reason</th>
                <th className={th}>(Not) Reachable Reason</th>
                <th className={thSim}>Route (simulator)</th>
                <th className={thSim}>Selected (simulator)</th>
                <th className={thSim}>Why (simulator)</th>
                <th className={thSim}>Basis</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr
                  key={r.id}
                  className={`border-b border-slate-200 dark:border-slate-800 ${r.selected ? 'bg-emerald-50/50 dark:bg-emerald-950/20' : ''}`}
                  data-route-row
                  data-prefix={r.prefix}
                  data-source={r.source}
                  data-selected={r.selected ? 'true' : 'false'}
                  data-reachable={r.reachable ? 'true' : 'false'}
                  data-origin={r.origin ?? ''}
                >
                  <td className={td}>{addressOf(r.prefix)}</td>
                  <td className={td}>{netmask(r.prefix)}</td>
                  <td className={td}>{r.segment}</td>
                  <td className={td}>{r.cost ?? '-'}</td>
                  <td className={td} title={r.preference_note}>
                    {r.preference ?? (r.preference_note ? 'not modelled' : '-')}
                  </td>
                  <td className={td}>{r.reachable ? 'True' : 'False'}</td>
                  <td className={td}>{r.next_hop}</td>
                  <td className={td}>{r.origin_name ?? '-'}</td>
                  <td className={td}>{r.lost_reason_code ?? ''}</td>
                  <td className={td}>{r.reachable_code ?? ''}</td>
                  <td className={td}>
                    {sourceText(r)}
                    {r.local_preference !== undefined && <span className="block">local pref {r.local_preference}</span>}
                    {r.as_path && r.as_path.length > 0 && <span className="block">AS path {r.as_path.join(' ')}</span>}
                    {r.metric !== undefined && <span className="block">MED {r.metric}</span>}
                  </td>
                  <td className={td}>{r.selected ? 'selected' : `rank ${r.rank}`}</td>
                  <td className={`${td} min-w-56 whitespace-normal`}>
                    {r.reason}
                    {r.notes?.map((n) => (
                      <span key={n} className="block text-slate-500 dark:text-slate-400">
                        {n}
                      </span>
                    ))}
                    {r.not_modelled?.map((n) => (
                      <span
                        key={n.rule}
                        className="mt-1 block text-rose-800 dark:text-rose-200"
                        data-route-not-modelled={n.rule}
                      >
                        Not modelled: {n.reason}
                      </span>
                    ))}
                  </td>
                  <td className={td}>
                    <Bases basis={r.basis} also={r.also} />
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td className={td} colSpan={14}>
                    No routes match.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      {er && (
        <section className="mt-5 rounded-lg border border-slate-200 p-4 dark:border-slate-800" data-lan-adverts>
          <h2 className="text-base font-semibold">What {er.name} sends to its LAN routers</h2>
          <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
            The simulator's view of redistribution (the product's tests are List OSPF Redistributed Routes and List BGP
            Redistributed Routes). Only the best route of a prefix is redistributed.
          </p>
          {er.lan_advertisements.length === 0 ? (
            <p className="mt-2 text-sm text-slate-500">
              This Edge runs no OSPF or BGP toward its LAN in this topology.
            </p>
          ) : (
            <div className="mt-2 overflow-x-auto" tabIndex={0} role="region" aria-label="LAN advertisements">
              <table className="min-w-full border-collapse text-left">
                <thead>
                  <tr className="border-b border-slate-300 dark:border-slate-700">
                    <th className={th}>Protocol</th>
                    <th className={th}>Prefix</th>
                    <th className={th}>Kind</th>
                    <th className={th}>Sent</th>
                    <th className={th}>Why</th>
                    <th className={th}>Basis</th>
                  </tr>
                </thead>
                <tbody>
                  {er.lan_advertisements.map((a, i) => (
                    <tr
                      key={i}
                      className="border-b border-slate-200 dark:border-slate-800"
                      data-advert
                      data-advertised={a.advertised ? 'true' : 'false'}
                      data-advert-switch={a.switch ?? ''}
                    >
                      <td className={td}>
                        {a.protocol.toUpperCase()}
                        {a.neighbor ? ` ${a.neighbor}` : ''}
                      </td>
                      <td className={td}>{a.prefix}</td>
                      <td className={td}>
                        {a.kind}
                        {a.ospf_type ? ` (${a.ospf_type.toUpperCase()}, metric ${a.metric ?? '-'})` : ''}
                      </td>
                      <td className={td}>{a.advertised ? 'sent' : 'not sent'}</td>
                      <td className={`${td} min-w-56 whitespace-normal`}>
                        {a.reason}
                        {a.attributes?.map((x) => (
                          <span key={x} className="block text-slate-500 dark:text-slate-400">
                            {x}
                          </span>
                        ))}
                      </td>
                      <td className={td}>
                        <Bases basis={a.basis} also={a.also} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {er.withheld.length > 0 && (
            <div className="mt-4" data-withheld>
              <h3 className="text-sm font-semibold">Offered by the LAN but not installed</h3>
              <ul className="mt-1 space-y-1 text-xs">
                {er.withheld.map((w, i) => (
                  <li key={i}>
                    <code>{w.prefix}</code> ({w.source.toUpperCase()} via {w.via}): {w.reason}{' '}
                    <Bases basis={w.basis} also={w.also} />
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>
      )}

      <div className="mt-5 grid gap-5 lg:grid-cols-2">
        <RoutingConditions />
        <AssumptionsPanel switches={data.assumptions} />
      </div>
      {data.not_modelled.length > 0 && (
        <section className="mt-5" aria-labelledby="rt-nm">
          <h2 id="rt-nm" className="text-base font-semibold">
            Not modelled
          </h2>
          <div className="mt-2 grid gap-2 md:grid-cols-2" data-routes-not-modelled>
            {data.not_modelled.map((n) => (
              <NotModelledNote key={n.rule} nm={n} />
            ))}
          </div>
        </section>
      )}
      <p className="mt-4 text-sm">
        Why is a prefix missing or unreachable?{' '}
        <Link to={withLab(`/sim/${topologyId}/reachability`, labSearch)} className="font-medium underline">
          Ask the reachability explainer
        </Link>
        .
      </p>
    </div>
  )
}
