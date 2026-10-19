import type { DiagramPacket } from '../../api'

export const PACKET_SPEED = 260 // viewBox units per second
export const PACKET_MIN_MS = 700

/** Total animation time of a packet (delay + travel); autoplay waits this long before advancing. */
export function packetDurationMs(p: DiagramPacket, lengthOf: (linkId: string) => number): number {
  if (!p.route) return 0
  const len = p.route.links.reduce((sum, id) => sum + lengthOf(id), 0)
  return p.delay_ms + Math.max(PACKET_MIN_MS, (len / PACKET_SPEED) * 1000)
}
