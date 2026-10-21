/**
 * How wide a node is drawn: its disc, or its label when that is wider. DiagramCanvas lays callouts out with these widths and
 * Diagram uses the same ones when it pans a narrow diagram to a step, so a label such as "Edge A" is never cut off at the edge
 * of the scroll box. Pure so it can be unit tested.
 */

/** The text size multiplier for a diagram drawn at `scale` (box width over viewBox width): text grows on phones, up to 2.2. */
export const boostFor = (scale: number): number => Math.min(Math.max(1, 0.66 / scale), 2.2)

/** Half the width of a node's disc or label, in viewBox units. */
export const nodeHalfWidth = (label: string, boost: number): number =>
  Math.max(41, (label.length * 17 * boost * 0.56) / 2)
