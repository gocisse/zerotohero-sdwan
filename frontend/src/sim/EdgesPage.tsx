import { Link } from 'react-router-dom'
import { withLab } from './labParams'
import { useTopologyCtx } from './topologyContext'

const link =
  'text-sky-700 underline decoration-sky-300 underline-offset-2 hover:text-sky-900 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:text-sky-300 dark:hover:text-sky-100'

/** Mock "Configure > Edges" (and a small "Configure > Profiles" with a Used By column). */
export default function EdgesPage() {
  const { detail, topologyId, labSearch } = useTopologyCtx()
  const t = detail.topology
  const edges = t.sites.flatMap((s) => s.edges.map((e) => ({ ...e, site: s })))
  const used = (pid: string) => edges.filter((e) => e.profile === pid).length
  return (
    <div className="space-y-8">
      <section aria-labelledby="edges-h">
        <h2 id="edges-h" className="text-lg font-semibold">
          Configure &gt; Edges
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          Open an Edge to see its Device, Business Policy and Firewall tabs with where every value comes from.
        </p>
        <div className="mt-3 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-800">
          <table className="w-full border-collapse text-left text-sm" data-edge-list>
            <thead>
              <tr className="bg-slate-100 dark:bg-slate-900">
                {['Edge', 'Site', 'Role', 'Profile', 'Overrides', 'Business Policy', 'Firewall'].map((h) => (
                  <th
                    key={h}
                    scope="col"
                    className="border-b border-slate-200 px-3 py-2 font-semibold dark:border-slate-800"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {edges.map((e) => {
                const p = t.profiles.find((x) => x.id === e.profile)
                const n = Object.keys(e.overrides.device ?? {}).length
                return (
                  <tr
                    key={e.id}
                    data-edge={e.id}
                    className="border-b border-slate-100 last:border-b-0 dark:border-slate-800/60"
                  >
                    <td className="px-3 py-2">
                      <Link to={withLab(`/sim/${topologyId}/edges/${e.id}`, labSearch)} className={link}>
                        {e.name}
                      </Link>
                    </td>
                    <td className="px-3 py-2">{e.site.name}</td>
                    <td className="px-3 py-2">{e.role}</td>
                    <td className="px-3 py-2">{p?.name ?? e.profile}</td>
                    <td className="px-3 py-2">{n > 0 ? `${n} device setting${n === 1 ? '' : 's'}` : 'none'}</td>
                    <td className="px-3 py-2">
                      <Link
                        to={withLab(`/sim/${topologyId}/edges/${e.id}?tab=business-policy`, labSearch)}
                        className={link}
                      >
                        Business Policy
                      </Link>
                    </td>
                    <td className="px-3 py-2">
                      <Link to={withLab(`/sim/${topologyId}/edges/${e.id}?tab=firewall`, labSearch)} className={link}>
                        Firewall
                      </Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </section>

      <section aria-labelledby="profiles-h">
        <h2 id="profiles-h" className="text-lg font-semibold">
          Configure &gt; Profiles
        </h2>
        <div className="mt-3 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-800">
          <table className="w-full border-collapse text-left text-sm" data-profile-list>
            <thead>
              <tr className="bg-slate-100 dark:bg-slate-900">
                {['Profile', 'Segments', 'Used By'].map((h) => (
                  <th
                    key={h}
                    scope="col"
                    className="border-b border-slate-200 px-3 py-2 font-semibold dark:border-slate-800"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {t.profiles.map((p) => (
                <tr key={p.id} className="border-b border-slate-100 last:border-b-0 dark:border-slate-800/60">
                  <td className="px-3 py-2">{p.name}</td>
                  <td className="px-3 py-2">{p.segments.join(', ')}</td>
                  <td className="px-3 py-2">{used(p.id)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
