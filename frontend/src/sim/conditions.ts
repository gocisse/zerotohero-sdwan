import type { Conditions } from '../api'

export const noConditions: Conditions = {}

export function isEmptyConditions(c: Conditions): boolean {
  return (
    !c.links_down?.length &&
    !c.links_degraded?.length &&
    !c.hubs_unreachable?.length &&
    !c.gateways_unreachable?.length &&
    !c.nsds_unreachable?.length &&
    !c.orchestrator_unreachable &&
    (!c.dynamic_build || c.dynamic_build === 'auto') &&
    !c.traffic?.length &&
    !c.neighbors_down?.length &&
    !c.probes_down?.length
  )
}

export type LinkSetting = 'up' | 'degraded' | 'down'

export function linkSetting(c: Conditions, id: string): LinkSetting {
  if (c.links_down?.includes(id)) return 'down'
  if (c.links_degraded?.includes(id)) return 'degraded'
  return 'up'
}

export function withLink(c: Conditions, id: string, s: LinkSetting): Conditions {
  const down = (c.links_down ?? []).filter((x) => x !== id)
  const degraded = (c.links_degraded ?? []).filter((x) => x !== id)
  if (s === 'down') down.push(id)
  if (s === 'degraded') degraded.push(id)
  return { ...c, links_down: down, links_degraded: degraded }
}

type ListKey = 'hubs_unreachable' | 'gateways_unreachable' | 'nsds_unreachable'

export function withToggled(c: Conditions, key: ListKey, id: string, on: boolean): Conditions {
  const rest = (c[key] ?? []).filter((x) => x !== id)
  return { ...c, [key]: on ? [...rest, id] : rest }
}
