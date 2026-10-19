/**
 * Where a step callout goes. Pure geometry in viewBox units so it can be unit tested without a browser
 * (`npm run test:unit`). Rule: a callout must not cover a link label, a node disc or a node label; it tries the
 * side it prefers first, then moves away from the target in steps, then tries the other side, and only gives up
 * (returning the first candidate) when nothing is free.
 */
export interface Rect {
  x: number
  y: number
  width: number
  height: number
}

export interface PlaceInput {
  /** Point the callout points at (node top or link label anchor). */
  anchor: { x: number; y: number }
  /** Estimated bubble size in viewBox units. */
  size: { width: number; height: number }
  /** Rectangles the bubble must not cover. */
  obstacles: Rect[]
  /** Canvas in viewBox units. */
  canvas: Rect
  prefer: 'above' | 'below'
}

export interface Placed {
  /** Horizontal centre of the bubble. */
  cx: number
  /** Top edge of the bubble. */
  top: number
}

export const overlaps = (a: Rect, b: Rect): boolean =>
  a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height

const overlapArea = (a: Rect, b: Rect): number =>
  Math.max(0, Math.min(a.x + a.width, b.x + b.width) - Math.max(a.x, b.x)) *
  Math.max(0, Math.min(a.y + a.height, b.y + b.height) - Math.max(a.y, b.y))

const GAP = 6
/** Gap between the anchor column and a bubble placed beside it (a node disc is 41 units wide each side). */
const BESIDE = 50
const STEP = 22

export function placeCallout({ anchor, size, obstacles, canvas, prefer }: PlaceInput): Placed {
  const sides: Array<'above' | 'below'> = prefer === 'above' ? ['above', 'below'] : ['below', 'above']
  const half = size.width / 2
  const minCx = canvas.x + half
  const maxCx = canvas.x + canvas.width - half
  const clampCx = (cx: number) => Math.min(Math.max(cx, minCx), Math.max(minCx, maxCx))
  const candidates: Placed[] = []
  for (const side of sides) {
    for (let i = 0; i < 6; i++) {
      const lift = GAP + i * STEP
      const top = side === 'above' ? anchor.y - lift - size.height : anchor.y + lift
      for (const dx of [0, -(half + 14), half + 14]) candidates.push({ cx: clampCx(anchor.x + dx), top })
    }
  }
  // Beside the target: needed when the target sits against the top or bottom edge, so neither side has room.
  for (const dx of [-(half + BESIDE), half + BESIDE]) {
    for (const rise of [0, size.height / 2, size.height])
      candidates.push({ cx: clampCx(anchor.x + dx), top: anchor.y - rise })
  }
  const rectOf = (p: Placed): Rect => ({ x: p.cx - half, y: p.top, width: size.width, height: size.height })
  const fits = (p: Placed) => {
    const r = rectOf(p)
    if (r.y < canvas.y || r.y + r.height > canvas.y + canvas.height) return false
    return !obstacles.some((o) => overlaps(r, o))
  }
  const found = candidates.find(fits)
  if (found) return found
  // Nothing is free: take the least bad spot instead of the first one (which can sit on top of the target).
  const cost = (p: Placed) => {
    const r = rectOf(p)
    const outside = Math.max(0, canvas.y - r.y) + Math.max(0, r.y + r.height - (canvas.y + canvas.height))
    return outside * size.width + obstacles.reduce((sum, o) => sum + overlapArea(r, o), 0)
  }
  return candidates.reduce((best, c) => (cost(c) < cost(best) ? c : best))
}

/** Rough text box of an SVG label, in viewBox units. */
export function labelRect(cx: number, baselineY: number, text: string, fontSize: number): Rect {
  const width = text.length * fontSize * 0.56 + 8
  return { x: cx - width / 2, y: baselineY - fontSize, width, height: fontSize + 5 }
}

/** Rough size of a callout bubble (max 200px wide, 12px text) converted to viewBox units. */
export function bubbleSize(text: string, scale: number): { width: number; height: number } {
  const px = Math.min(200, text.length * 6.6 + 22)
  const lines = Math.max(1, Math.ceil((text.length * 6.6) / (px - 22)))
  return { width: px / scale, height: (lines * 16 + 14) / scale }
}
