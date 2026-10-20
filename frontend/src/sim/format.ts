import type { SimSource } from '../api'

/** Text for a setting value, rule field or diff cell. */
export function valueText(v: unknown): string {
  if (v === null || v === undefined) return '(not set)'
  if (typeof v === 'boolean') return v ? 'on' : 'off'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

export function sourceLabel(s: SimSource): string {
  switch (s.kind) {
    case 'profile':
      return `from Profile ${s.name ?? s.id ?? ''}`.trim()
    case 'edge-override':
      return 'overridden on this Edge'
    case 'edge-only':
      return 'Edge only'
    case 'customer':
      return 'customer setting'
    case 'default':
      return 'default'
    default:
      return 'unset'
  }
}
