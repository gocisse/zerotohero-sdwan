import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Progress, TopicMastery } from '../api'
import { GOALS, goalCoverage, topicMatches } from './goals.ts'

const progress = (over: Partial<Progress> = {}): Progress => ({
  overall_percentage: 0,
  lessons_completed: 0,
  lessons_total: 0,
  lessons: [],
  sections: [
    {
      section_id: 'routing',
      lessons_completed: 4,
      lessons_total: 8,
      quiz_attempts: 1,
      best_score_pct: 80,
      latest_score_pct: 70,
      quiz_skipped: false,
      quiz_skipped_at: null,
      last_visited_at: null,
    },
  ],
  labs: {
    total: 3,
    started: 0,
    completed: 1,
    skipped: 0,
    items: [
      { lab_id: 'which-route-wins', title: 'x', status: 'completed', attempts: 2, hints_used: 0 },
      { lab_id: 'branch-b-cannot-reach', title: 'y', status: 'not_started', attempts: 0, hints_used: 0 },
    ],
  },
  ...over,
})

const routes = GOALS.find((g) => g.id === 'routes')!

test('a prefix topic matches every tag with that prefix and an exact one only itself', () => {
  assert.equal(topicMatches(routes, 'rt-bgp'), true)
  assert.equal(topicMatches(routes, 'dmpo-steering'), false)
  const read = GOALS.find((g) => g.id === 'read-config')!
  assert.equal(topicMatches(read, 'edge-role'), true)
  assert.equal(topicMatches(read, 'edge-roles'), false)
})

test('coverage is the mean of the parts that have data, and a goal with no data is null', () => {
  const c = goalCoverage(routes, progress(), null, null, {})
  assert.equal(c.lessons, 50)
  assert.equal(c.quiz, 80)
  assert.equal(c.labs, 50) // 1 completed of the 2 routes labs the dashboard knows about
  assert.equal(c.selfCheck, null)
  assert.equal(c.overall, (50 + 80 + 50) / 3)
  const none = goalCoverage(
    GOALS.find((g) => g.id === 'tunnels')!,
    progress({ sections: [], labs: { total: 0, started: 0, completed: 0, skipped: 0, items: [] } }),
    null,
    null,
    {},
  )
  assert.equal(none.overall, null)
})

test('the self-check, weak topics and the checklist are counted but never required', () => {
  const topics: TopicMastery[] = [
    { topic: 'rt-bgp', answers: 5, accuracy_pct: 40, weak: true, lessons: [] },
    { topic: 'dmpo-steering', answers: 5, accuracy_pct: 40, weak: true, lessons: [] },
  ]
  const memory = {
    at: '2026-10-20T00:00:00Z',
    perSection: { routing: { correct: 1, total: 2 } },
    screens: { correct: 0, total: 0 },
  }
  const ticked = Object.fromEntries(routes.explain.map((i) => [i.id, true]))
  const c = goalCoverage(routes, progress(), topics, memory, ticked)
  assert.equal(c.selfCheck, 50)
  assert.deepEqual(c.weakTopics, ['rt-bgp'])
  assert.deepEqual(c.checklist, { ticked: routes.explain.length, total: routes.explain.length + routes.find.length })
})

test('every goal has explain and find items with unique ids', () => {
  const ids = GOALS.flatMap((g) => [...g.explain, ...g.find].map((i) => i.id))
  assert.equal(new Set(ids).size, ids.length)
  for (const g of GOALS) assert.ok(g.explain.length >= 2 && g.find.length >= 2, g.id)
})
