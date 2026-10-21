import assert from 'node:assert/strict'
import { test } from 'node:test'
import { boostFor, nodeHalfWidth } from './nodeExtent.ts'

test('the text boost is 1 on wide screens, grows as the diagram shrinks and stops at 2.2', () => {
  assert.equal(boostFor(1), 1)
  assert.equal(boostFor(0.66), 1)
  assert.ok(boostFor(0.4) > 1.6 && boostFor(0.4) < 1.7)
  assert.equal(boostFor(0.1), 2.2)
})

test('a short label stays within the disc; a long or boosted label is wider than the disc', () => {
  assert.equal(nodeHalfWidth('Edge', 1), 41)
  // "Edge A" at the phone boost is wider than the 41 unit disc: this is the label that was clipped at 390 px.
  assert.ok(nodeHalfWidth('Edge A', 2.2) > 41)
  assert.ok(nodeHalfWidth('Activation and management', 1) > nodeHalfWidth('Edge A', 1))
})
