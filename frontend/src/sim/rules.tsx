import type { BusinessRule, FirewallRule } from '../api'
import { valueText } from './format'

/** "key value" pairs of a rule match, skipping empty fields. */
export function matchText(m: Record<string, string | boolean | undefined>): string {
  const parts = Object.entries(m)
    .filter(([, v]) => v !== '' && v !== undefined && v !== false)
    .map(([k, v]) => (v === true ? k.replace(/_/g, ' ') : `${k.replace(/_/g, ' ')} ${String(v)}`))
  return parts.length ? parts.join(', ') : 'any traffic'
}

export function actionText(a: BusinessRule['action'], service?: string, steeringMode?: string): string {
  const parts: string[] = []
  if (a.priority) parts.push(`priority ${a.priority}`)
  const ns = service ?? a.network_service
  if (ns) parts.push(`service ${ns.replace(/_/g, ' ')}`)
  const s = a.steering
  const mode = steeringMode ?? s.mode ?? 'auto'
  let steer = `link steering ${mode.replace(/_/g, ' ')}`
  if (s.group) steer += ` (${s.group.replace(/_/g, ' ')})`
  if (s.interface) steer += ` (interface ${s.interface})`
  if (s.link) steer += ` (link ${s.link})`
  if (s.strictness) steer += `, ${s.strictness}`
  parts.push(steer)
  if (a.service_class) parts.push(`service class ${a.service_class.replace(/_/g, ' ')}`)
  if (a.rate_limit) parts.push('rate limit')
  if (a.nat) parts.push('NAT')
  return parts.join(' · ')
}

export const fwMatchText = (r: FirewallRule): string => matchText(r.match)

/** Readable text for a what-if before/after cell, by item kind. */
export function diffValue(kind: string, v: unknown, verb: 'before' | 'after'): string {
  if (v === null || v === undefined) return verb === 'before' ? '(not there)' : '(removed)'
  if (typeof v !== 'object' || Array.isArray(v)) return Array.isArray(v) ? v.join(', ') : valueText(v)
  const o = v as Record<string, unknown>
  switch (kind) {
    case 'business-rule': {
      const a = o.action as BusinessRule['action']
      const sup = o.superseded_by ? ` · superseded by ${String(o.superseded_by)}` : ''
      return `${actionText(a, typeof o.service === 'string' ? o.service : undefined)}${sup}`
    }
    case 'firewall-rule':
      return `${String(o.action)} · ${String(o.group).replace('-', ' ')} group · position ${String(o.position)}`
    case 'interface': {
      const vlan = o.vlan === null || o.vlan === undefined ? 'no VLAN' : `VLAN ${String(o.vlan)}`
      const segs = Array.isArray(o.segments) ? o.segments.join(', ') : ''
      return `${vlan} · segments ${segs} · interface override ${o.override ? 'on' : 'off'}`
    }
    case 'vlan': {
      const settings = o.settings && typeof o.settings === 'object' ? ` · overridden ${valueText(o.settings)}` : ''
      return `name ${String(o.name)}${settings}`
    }
    default:
      return valueText(v)
  }
}
