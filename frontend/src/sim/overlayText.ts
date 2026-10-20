import type { TunnelState, TunnelType } from '../api'

export const stateLabel: Record<TunnelState, string> = {
  up: 'up',
  degraded: 'degraded',
  down: 'down',
  'not-established': 'not established',
  'torn-down': 'torn down',
  unknown: 'unknown',
}

export const typeLabel: Record<TunnelType, string> = {
  gateway: 'Gateway tunnel',
  hub: 'Static Hub tunnel',
  dynamic: 'Dynamic branch-to-branch',
  'nsd-gateway': 'NSD via Gateway',
  'nsd-edge': 'NSD via Edge',
}

export const modeLabel = (mode: string) => (mode === 'dynamic' ? 'Dynamic' : 'Static')
