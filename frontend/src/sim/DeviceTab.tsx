import type { EffectiveConfig, EffectiveSetting } from '../api'
import { Bases, SourceBadge } from './ui'
import { valueText } from './format'

/** One setting row: label, the Override checkbox state, value, where it comes from and why it is believed. */
export function SettingRow({ s }: { s: EffectiveSetting }) {
  const overridden = s.source.kind === 'edge-override'
  return (
    <li
      data-setting={s.id}
      data-source-kind={s.source.kind}
      className="rounded-md border border-slate-200 p-3 dark:border-slate-800"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="font-medium">{s.label}</span>
        <label className="flex items-center gap-1.5 text-xs text-slate-600 dark:text-slate-300">
          <input
            type="checkbox"
            checked={overridden}
            disabled
            readOnly
            aria-label={`Override ${s.label} (read-only)`}
          />
          Override
        </label>
      </div>
      <p className="mt-1 text-lg" data-setting-value>
        {valueText(s.value)}
      </p>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <SourceBadge source={s.source} />
        <Bases basis={s.basis} also={s.also} />
      </div>
      <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">{s.why}</p>
      {overridden && s.profile_value !== undefined && (
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Profile value: {valueText(s.profile_value)}</p>
      )}
    </li>
  )
}

const ORDER = ['Connectivity', 'Routing & NAT', 'Other (section not confirmed)']

/** The Device tab: Connectivity (settings, VLANs, interfaces), Routing & NAT, and settings whose section was not read. */
export default function DeviceTab({ ec }: { ec: EffectiveConfig }) {
  // Stateful Firewall is shown on the Firewall tab.
  const settings = ec.device.filter((s) => s.section !== 'Firewall')
  const bySection = ORDER.map((name) => ({ name, items: settings.filter((s) => s.section === name) }))
  return (
    <div className="space-y-8">
      {bySection.map(({ name, items }) => (
        <section key={name} aria-labelledby={`sec-${name}`} data-device-section={name}>
          <h3 id={`sec-${name}`} className="text-base font-semibold">
            {name}
          </h3>
          {name.startsWith('Other') && (
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              The Device tab section for these settings was not read, so they are grouped here (assumption 36).
            </p>
          )}
          <ul className="mt-2 space-y-2">
            {items.map((s) => (
              <SettingRow key={s.id} s={s} />
            ))}
          </ul>
          {name === 'Connectivity' && (
            <>
              <h4 className="mt-5 text-sm font-semibold">VLANs</h4>
              <ul className="mt-2 space-y-2" data-vlans>
                {ec.vlans.map((v) => (
                  <li
                    key={v.id}
                    data-vlan={v.id}
                    className="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">
                        VLAN {v.id}: {v.name}
                      </span>
                      <SourceBadge source={v.source} />
                      <Bases basis={v.basis} also={v.also} />
                    </div>
                    {v.name_id_locked && (
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        A Profile VLAN's name and ID cannot be overridden on an Edge.
                      </p>
                    )}
                    {v.settings && <p className="mt-1 text-xs">Overridden settings: {valueText(v.settings)}</p>}
                  </li>
                ))}
                {ec.vlans.length === 0 && <li className="text-sm text-slate-500">No VLANs.</li>}
              </ul>
              <h4 className="mt-5 text-sm font-semibold">Interfaces</h4>
              <ul className="mt-2 space-y-2" data-interfaces>
                {ec.interfaces.map((i) => (
                  <li
                    key={i.id}
                    data-interface={i.id}
                    className="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">
                        {i.id} ({i.kind})
                      </span>
                      <SourceBadge source={i.source} />
                      <Bases basis={i.basis} also={i.also} />
                    </div>
                    <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                      {i.vlan !== undefined && <>VLAN {i.vlan} · </>}
                      {i.link && <>WAN link {i.link} · </>}
                      Segments: {i.segments.join(', ')}
                      {i.segments_source === 'profile-default' ? " (all of the Profile's segments: the default)" : ''}
                      {i.override ? ' · interface configuration overridden' : ''}
                    </p>
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>
      ))}

      <section aria-labelledby="sec-customer">
        <h3 id="sec-customer" className="text-base font-semibold">
          Enterprise settings
        </h3>
        <ul className="mt-2 space-y-2">
          {ec.customer.map((c) => (
            <li
              key={c.id}
              className="flex flex-wrap items-center gap-2 rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800"
            >
              <span className="font-medium">{c.label}:</span> {valueText(c.value)}
              <SourceBadge source={c.source} />
              <Bases basis={c.basis} />
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
