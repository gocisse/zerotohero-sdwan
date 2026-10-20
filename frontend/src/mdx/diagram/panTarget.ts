/**
 * Where a narrow diagram box should scroll for a step. Pure (pixels in the scroll box's own coordinates) so it can be unit
 * tested. Rules, in order:
 * 1. the step's nodes and callout bubbles fit together: centre them together;
 * 2. otherwise centre the nodes, but never let a callout bubble be cut off when the bubbles themselves fit in the box
 *    (the bubbles are the step's explanation, a node at the edge only needs to be partly in view);
 * 3. the bubbles do not fit either: centre the nodes.
 * The result is clamped to the scrollable range.
 */
export interface PanInput {
  /** Horizontal extent of the nodes the step is about. */
  nodes: { left: number; right: number }
  /** Horizontal extent of each callout bubble. */
  bubbles: Array<{ left: number; right: number }>
  /** Width of the visible box. */
  view: number
  /** scrollWidth - clientWidth. */
  maxScroll: number
}

const clamp = (v: number, lo: number, hi: number) => Math.min(Math.max(v, lo), hi)

export function panTarget({ nodes, bubbles, view, maxScroll }: PanInput): number {
  const centred = (lo: number, hi: number) => (lo + hi) / 2 - view / 2
  let left = centred(nodes.left, nodes.right)
  if (bubbles.length > 0) {
    const bl = Math.min(...bubbles.map((b) => b.left))
    const br = Math.max(...bubbles.map((b) => b.right))
    const lo = Math.min(nodes.left, bl)
    const hi = Math.max(nodes.right, br)
    if (hi - lo <= view) left = centred(lo, hi)
    else if (br - bl <= view) left = clamp(left, br - view, bl)
  }
  return clamp(left, 0, Math.max(0, maxScroll))
}
