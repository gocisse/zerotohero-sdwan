import type { Diagram, DiagramGroup } from '../../api'

export interface Pt {
  x: number
  y: number
}

/** Path from a to b (center to center; node discs are drawn on top), plus the label anchor. */
export function linkGeometry(a: Pt, b: Pt, curve = 0): { d: string; mid: Pt } {
  const mx = (a.x + b.x) / 2
  const my = (a.y + b.y) / 2
  if (!curve) return { d: `M ${a.x} ${a.y} L ${b.x} ${b.y}`, mid: { x: mx, y: my } }
  const len = Math.hypot(b.x - a.x, b.y - a.y) || 1
  // Quadratic control point; the apex sits halfway between the chord and the control point.
  const cx = mx + (-(b.y - a.y) / len) * curve * 2
  const cy = my + ((b.x - a.x) / len) * curve * 2
  return {
    d: `M ${a.x} ${a.y} Q ${cx} ${cy} ${b.x} ${b.y}`,
    mid: { x: 0.25 * a.x + 0.5 * cx + 0.25 * b.x, y: 0.25 * a.y + 0.5 * cy + 0.25 * b.y },
  }
}

const PAD = 62

/** Boundary of a group: its explicit box, or the padded bounds of its member nodes. */
export function groupBox(d: Diagram, g: DiagramGroup) {
  if (g.box) return g.box
  const members = d.nodes.filter((n) => n.group === g.id || g.nodes.includes(n.id))
  if (members.length === 0) return null
  const xs = members.map((n) => n.x)
  const ys = members.map((n) => n.y)
  const x = Math.min(...xs) - PAD
  const y = Math.min(...ys) - PAD
  return { x, y, width: Math.max(...xs) + PAD - x, height: Math.max(...ys) + PAD - y }
}

export function groupMembers(d: Diagram, g: DiagramGroup): string[] {
  return d.nodes.filter((n) => n.group === g.id || g.nodes.includes(n.id)).map((n) => n.id)
}

/**
 * Which elements are hidden at step `index`. An element listed in any step's `show` starts hidden;
 * steps 0..index are replayed in order, so `show` reveals and `hide` removes.
 */
export function hiddenAt(d: Diagram, index: number): Set<string> {
  const hidden = new Set<string>()
  for (const s of d.steps) for (const id of s.show) hidden.add(id)
  for (let i = 0; i <= index && i < d.steps.length; i++) {
    const s = d.steps[i]!
    for (const id of s.show) hidden.delete(id)
    for (const id of s.hide) hidden.add(id)
  }
  return hidden
}
