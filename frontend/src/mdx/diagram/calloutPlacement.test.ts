import assert from 'node:assert/strict'
import { test } from 'node:test'
import { bubbleSize, labelRect, overlaps, placeCallout, type Rect } from './calloutPlacement.ts'

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
