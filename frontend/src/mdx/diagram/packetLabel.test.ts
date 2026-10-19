import assert from 'node:assert/strict'
import { test } from 'node:test'
import { overlaps } from './calloutPlacement.ts'
import { labelRect } from './calloutPlacement.ts'
import { nodeDisc, packetLabelDy } from './packetLabel.ts'

test('on an empty link the label stays just above the dot', () => {
  assert.equal(packetLabelDy({ x: 300, y: 200 }, 'via Hub', 13, [nodeDisc(600, 200)]), -15)
})

test('a packet resting on a node gets its label above the node disc', () => {
  const disc = nodeDisc(750, 400)
  const dy = packetLabelDy({ x: 750, y: 400 }, 'via Hub', 13, [disc])
  assert.equal(overlaps(labelRect(750, 400 + dy, 'via Hub', 13), disc), false)
})

test('the offset scales with the text size', () => {
  const disc = nodeDisc(100, 100)
  const small = packetLabelDy({ x: 100, y: 100 }, 'x', 13, [disc])
  const big = packetLabelDy({ x: 100, y: 100 }, 'x', 26, [disc])
  assert.ok(big < small)
})
