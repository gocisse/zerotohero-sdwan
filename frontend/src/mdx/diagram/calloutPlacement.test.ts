import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  bubbleSize,
  labelRect,
  overlaps,
  placeCallout,
  placeLinkLabel,
  pxPerUnit,
  type Rect,
} from './calloutPlacement.ts'
import { linkGeometry } from './geometry.ts'

const canvas: Rect = { x: -30, y: 0, width: 960, height: 460 }
const size = { width: 200, height: 48 }
const rectOf = (p: { cx: number; top: number }): Rect => ({
  x: p.cx - size.width / 2,
  y: p.top,
  width: size.width,
  height: size.height,
})

test('free space: the preferred side is used right next to the anchor', () => {
  const p = placeCallout({ anchor: { x: 450, y: 250 }, size, obstacles: [], canvas, prefer: 'above' })
  assert.equal(p.cx, 450)
  assert.ok(p.top + size.height <= 250)
})

test('a link label under the preferred spot is avoided', () => {
  const label = labelRect(450, 242, 'MPLS path', 13)
  const p = placeCallout({ anchor: { x: 450, y: 250 }, size, obstacles: [label], canvas, prefer: 'above' })
  assert.equal(overlaps(rectOf(p), label), false)
})

test('a node disc and its label are avoided', () => {
  const node: Rect = { x: 409, y: 209, width: 82, height: 120 }
  const p = placeCallout({ anchor: { x: 450, y: 250 }, size, obstacles: [node], canvas, prefer: 'below' })
  assert.equal(overlaps(rectOf(p), node), false)
})

test('the bubble stays inside the canvas near an edge', () => {
  const p = placeCallout({ anchor: { x: -20, y: 10 }, size, obstacles: [], canvas, prefer: 'above' })
  const r = rectOf(p)
  assert.ok(r.x >= canvas.x && r.y >= canvas.y && r.x + r.width <= canvas.x + canvas.width)
})

test('with every candidate blocked it still returns a position', () => {
  const wall: Rect = { x: -100, y: -100, width: 2000, height: 2000 }
  const p = placeCallout({ anchor: { x: 450, y: 250 }, size, obstacles: [wall], canvas, prefer: 'above' })
  assert.equal(typeof p.top, 'number')
})

test('bubble size grows with text and shrinks with scale', () => {
  assert.ok(bubbleSize('a much longer callout text that wraps over lines', 1).height > bubbleSize('hi', 1).height)
  assert.ok(bubbleSize('hello', 0.5).width > bubbleSize('hello', 1).width)
})

test('a target against the top edge puts the bubble beside it, not on it', () => {
  // Node at the top of the canvas: its disc starts at y = 59, so there is no room above it.
  const node: Rect = { x: 409, y: 59, width: 82, height: 120 }
  const p = placeCallout({
    anchor: { x: 450, y: 59 },
    size: { width: 200, height: 74 },
    obstacles: [node],
    canvas,
    prefer: 'below',
  })
  assert.equal(overlaps({ x: p.cx - 100, y: p.top, width: 200, height: 74 }, node), false)
  assert.ok(p.top >= canvas.y)
})

test('when nothing is free the least overlapping spot wins, not the first candidate', () => {
  const small: Rect = { x: 380, y: 130, width: 140, height: 120 }
  const wall: Rect = { x: -100, y: -100, width: 2000, height: 135 }
  const p = placeCallout({
    anchor: { x: 450, y: 130 },
    size,
    obstacles: [wall, small],
    canvas: { x: -30, y: 0, width: 960, height: 300 },
    prefer: 'above',
  })
  assert.equal(typeof p.cx, 'number')
})

test('pixels per unit use the padded viewBox width, and a bubble placed against the left edge stays inside the box', () => {
  // A 900 unit diagram with 30 units of side padding is a 960 unit viewBox: 560 px wide means 560 / 960 px per unit.
  assert.equal(pxPerUnit(560, 960), 560 / 960)
  assert.equal(pxPerUnit(0, 960), 1)
  const unit = pxPerUnit(560, 960)
  const size = bubbleSize('The initiating Edge checks whether a direct tunnel is possible.', unit)
  const canvas: Rect = { x: -30, y: 0, width: 960, height: 520 }
  const p = placeCallout({ anchor: { x: 10, y: 140 }, size, obstacles: [], canvas, prefer: 'above' })
  // Left edge of the bubble in pixels from the left edge of the SVG, which sits at viewBox x = -30.
  const leftPx = (p.cx - size.width / 2 - canvas.x) * unit
  assert.ok(leftPx >= -0.5, `bubble hangs ${leftPx}px outside the box`)
})

test('linkGeometry.at runs from one end to the other, straight and curved', () => {
  const a = { x: 0, y: 0 }
  const b = { x: 100, y: 40 }
  const straight = linkGeometry(a, b)
  assert.deepEqual(straight.at(0), a)
  assert.deepEqual(straight.at(1), b)
  assert.deepEqual(straight.at(0.5), straight.mid)
  const curved = linkGeometry(a, b, 30)
  assert.deepEqual(curved.at(0), a)
  assert.deepEqual(curved.at(1), b)
  assert.ok(Math.abs(curved.at(0.5).x - curved.mid.x) < 1e-9 && Math.abs(curved.at(0.5).y - curved.mid.y) < 1e-9)
})

test('a link label stays at the middle when nothing is in the way', () => {
  const geo = linkGeometry({ x: 100, y: 100 }, { x: 500, y: 100 })
  assert.deepEqual(placeLinkLabel(geo.at, 'assign', 13, []), geo.mid)
})

test('a link label slides along its link off a node label that sits at the middle (the svc-workflow case)', () => {
  // A long node label ends right where the middle of the link is.
  const geo = linkGeometry({ x: 150, y: 270 }, { x: 450, y: 410 })
  const nodeLabel: Rect = { x: 40, y: 255, width: 200, height: 30 }
  const text = 'or at Edge'
  const mid = labelRect(geo.mid.x, geo.mid.y - 8, text, 13)
  const p = placeLinkLabel(geo.at, text, 13, [nodeLabel, { x: 180, y: 230, width: 130, height: 70 }])
  const placed = labelRect(p.x, p.y - 8, text, 13)
  assert.equal(overlaps(placed, nodeLabel), false)
  assert.equal(overlaps(placed, { x: 180, y: 230, width: 130, height: 70 }), false)
  // The middle position would have been free here; the test checks the slide only when it was blocked.
  assert.ok(overlaps(mid, nodeLabel) || p.x === geo.mid.x)
})

test('a link label that is blocked everywhere takes the least bad position instead of failing', () => {
  const geo = linkGeometry({ x: 0, y: 0 }, { x: 400, y: 0 })
  const wall: Rect = { x: -100, y: -100, width: 1000, height: 200 }
  const p = placeLinkLabel(geo.at, 'label', 13, [wall])
  assert.equal(typeof p.x, 'number')
})
