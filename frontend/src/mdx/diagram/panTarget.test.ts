import assert from 'node:assert/strict'
import { test } from 'node:test'
import { panTarget } from './panTarget.ts'

const view = 358
const visible = (left: number, b: { left: number; right: number }) => b.left >= left && b.right <= left + view

test('no bubbles: the nodes are centred and the result stays in range', () => {
  assert.equal(panTarget({ nodes: { left: 300, right: 500 }, bubbles: [], view, maxScroll: 400 }), 221)
  assert.equal(panTarget({ nodes: { left: 0, right: 40 }, bubbles: [], view, maxScroll: 400 }), 0)
  assert.equal(panTarget({ nodes: { left: 900, right: 1000 }, bubbles: [], view, maxScroll: 400 }), 400)
})

test('nodes and bubbles that fit together are centred together', () => {
  const left = panTarget({
    nodes: { left: 200, right: 260 },
    bubbles: [{ left: 150, right: 250 }],
    view,
    maxScroll: 800,
  })
  assert.equal(left, (150 + 260) / 2 - view / 2)
})

test('wide node span: a bubble at its edge is still fully visible (the Silver option 2 case)', () => {
  // Nodes span 500 units (wider than the view); one bubble sits at the left end, one near the middle.
  const bubbles = [
    { left: 100, right: 230 },
    { left: 300, right: 400 },
  ]
  const left = panTarget({ nodes: { left: 150, right: 650 }, bubbles, view, maxScroll: 800 })
  for (const b of bubbles) assert.ok(visible(left, b), `bubble ${b.left}-${b.right} cut off at scroll ${left}`)
})

test('wide node span: a bubble at the right end is fully visible too', () => {
  const bubbles = [{ left: 560, right: 700 }]
  const left = panTarget({ nodes: { left: 50, right: 650 }, bubbles, view, maxScroll: 900 })
  assert.ok(visible(left, bubbles[0]!))
})

test('bubbles wider than the view: the nodes are centred', () => {
  const left = panTarget({ nodes: { left: 300, right: 400 }, bubbles: [{ left: 0, right: 700 }], view, maxScroll: 900 })
  assert.equal(left, 350 - view / 2)
})

test('a bubble beyond the scroll range is clamped, never negative', () => {
  assert.equal(
    panTarget({ nodes: { left: 0, right: 60 }, bubbles: [{ left: -40, right: 80 }], view, maxScroll: 300 }),
    0,
  )
})

test('nodes that fit stay whole when a scroll also keeps the bubble whole (the Edge A case at 390 px)', () => {
  // Nodes 100..300, bubble 330..450: together 350 wide, a little under the 358 box, so they are centred together.
  const nodes = { left: 100, right: 300 }
  const left = panTarget({ nodes, bubbles: [{ left: 330, right: 450 }], view, maxScroll: 800 })
  assert.ok(visible(left, nodes) && visible(left, { left: 330, right: 450 }))
  // Bubble 380..500 does not fit beside them (400 wide): it is kept whole, and nodes whole only if that is possible.
  const bubble = { left: 360, right: 480 }
  const l2 = panTarget({ nodes: { left: 150, right: 330 }, bubbles: [bubble], view, maxScroll: 800 })
  assert.ok(visible(l2, bubble))
})

test('nodes wider than the box keep the earlier rule: a bubble that fits is not cut off', () => {
  const bubbles = [{ left: 560, right: 700 }]
  const left = panTarget({ nodes: { left: 50, right: 650 }, bubbles, view, maxScroll: 900 })
  assert.ok(visible(left, bubbles[0]!))
})
