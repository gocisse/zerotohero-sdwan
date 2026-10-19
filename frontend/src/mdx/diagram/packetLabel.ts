import { labelRect, overlaps, type Rect } from './calloutPlacement.ts'

/** Half-size of a node disc in viewBox units (same value the callout obstacles use). */
export const NODE_HALF = 41

/** Obstacle rectangle for a node disc centred at (x, y). */
export const nodeDisc = (x: number, y: number): Rect => ({
  x: x - NODE_HALF,
  y: y - NODE_HALF,
  width: 2 * NODE_HALF,
  height: 2 * NODE_HALF,
})

/**
 * Vertical offset of a packet's label from the packet centre. The default sits just above the dot; when that would
 * cover a node disc (a packet resting on a node) it moves above the disc instead, or below it as a last resort.
 * Pure geometry in viewBox units so it can be unit tested.
 */
export function packetLabelDy(at: { x: number; y: number }, text: string, fontSize: number, discs: Rect[]): number {
  const offsets = [-15, -(NODE_HALF + 8), NODE_HALF + 8 + fontSize].map((d) => d * (fontSize / 13))
  const free = (dy: number) => {
    const r = labelRect(at.x, at.y + dy, text, fontSize)
    return !discs.some((d) => overlaps(r, d))
  }
  return offsets.find(free) ?? offsets[0]!
}
