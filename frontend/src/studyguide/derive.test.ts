import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { GuideLesson, GuideTopic, StudyGuide } from '../api'
import { groupGlossary, primaryGroup, splitMatches, termMatches } from './glossaryGroups.ts'
import {
  guideLessonOrder,
  heatmapWeeks,
  intensity,
  nextRecommended,
  overallStats,
  sectionStats,
  sessionsThisWeek,
  sparkPoints,
} from './derive.ts'

const lesson = (id: string, status: GuideLesson['status'] = 'not_started', section = 's1'): GuideLesson => ({
  id,
  title: id,
  section_id: section,
  status,
})
const topic = (
  id: string,
  coverage: GuideTopic['coverage'],
  lessons: GuideLesson[],
  over: Partial<GuideTopic> = {},
): GuideTopic => ({
  id,
  title: id,
  kind: 'concept',
  coverage,
  note: 'n',
  lessons,
  quiz_topics: [],
  diagrams: [],
  labs: [],
  sim_screens: [],
  progress: {
    lessons_completed: lessons.filter((l) => l.status === 'completed').length,
    lessons_total: lessons.length,
    best_section_score_pct: null,
    topic_accuracy_pct: null,
    quiz_answers: 0,
    labs_completed: 0,
    labs_total: 0,
    review_due: 0,
  },
  ...over,
})
const guide = (): StudyGuide => ({
  sections: [
    {
      id: 'G1',
      title: 'One',
      topics: [
        topic('a', 'full', [lesson('l1', 'completed')]),
        topic('b', 'partial', [lesson('l1', 'completed'), lesson('l2', 'visited')]),
      ],
    },
    { id: 'G2', title: 'Two', topics: [topic('c', 'none', [])] },
  ],
  beyond: { title: 'Beyond', reason: 'r' },
  unmapped_lessons: [lesson('l9')],
  warnings: [],
  sim_screens: [],
})

test('progress is computed over covered topics; uncovered topics are reported separately', () => {
  const o = overallStats(guide())
  assert.deepEqual(
    { covered: o.covered, done: o.done, notCovered: o.notCovered },
    { covered: 2, done: 1, notCovered: 1 },
  )
  assert.equal(o.fraction, 0.5)
})

test('a guide with nothing covered has zero progress, not NaN', () => {
  const g = guide()
  g.sections = [{ id: 'G5', title: 'x', topics: [topic('z', 'none', [])] }]
  assert.equal(overallStats(g).fraction, 0)
})

test('section stats count a shared lesson once and keep uncovered topics out of "covered"', () => {
  const s = sectionStats(guide().sections[0]!)
  assert.equal(s.lessonsTotal, 2)
  assert.equal(s.lessonsCompleted, 1)
  assert.deepEqual(s.byCoverage, { full: 1, partial: 1, none: 0 })
  const s2 = sectionStats(guide().sections[1]!)
  assert.equal(s2.covered, 0)
  assert.equal(s2.byCoverage.none, 1)
})

test('average best quiz score takes each course section once', () => {
  const g = guide()
  const [a, b] = g.sections[0]!.topics as [GuideTopic, GuideTopic]
  a.progress.best_section_score_pct = 80
  b.progress.best_section_score_pct = 80
  assert.equal(sectionStats(g.sections[0]!).avgBestQuizPct, 80)
})

test('guide order lists each lesson once, unmapped last, and the next step skips completed and unmapped lessons', () => {
  const g = guide()
  assert.deepEqual(
    guideLessonOrder(g).map((x) => x.lesson.id),
    ['l1', 'l2', 'l9'],
  )
  assert.equal(nextRecommended(g)?.id, 'l2')
  g.sections[0]!.topics[1]!.lessons[1]!.status = 'completed'
  assert.equal(nextRecommended(g), null)
})

test('the heatmap has 12 Monday-first weeks ending with the current week and never counts the future', () => {
  const now = new Date(2026, 2, 4, 15, 0) // Wednesday
  const weeks = heatmapWeeks(
    [
      { kind: 'lesson_completed', at: new Date(2026, 2, 2, 9, 0).toISOString() },
      { kind: 'quiz_attempt', at: new Date(2026, 2, 2, 9, 30).toISOString() },
      { kind: 'lesson_visited', at: new Date(2026, 2, 5, 9, 0).toISOString() },
    ],
    now,
  )
  assert.equal(weeks.length, 12)
  const last = weeks[11]!
  assert.equal(last[0]!.date, '2026-03-02')
  assert.equal(last[0]!.total, 2)
  assert.equal(last[3]!.future, true)
  assert.equal(last[3]!.total, 0)
})

test('intensity buckets', () => {
  assert.deepEqual([0, 1, 2, 3, 4, 6, 7].map(intensity), [0, 1, 2, 2, 3, 3, 4])
})

test('events within 30 minutes are one session; older ones are not counted', () => {
  const now = new Date('2026-03-04T12:00:00Z')
  const at = (m: number) => new Date(now.getTime() - m * 60000).toISOString()
  const ev = (m: number) => ({ kind: 'lesson_visited' as const, at: at(m) })
  assert.equal(sessionsThisWeek([ev(10), ev(20), ev(35), ev(200), ev(10 * 24 * 60)], now), 2)
  assert.equal(sessionsThisWeek([], now), 0)
})

test('sparkline points stay inside the box', () => {
  const pts = sparkPoints([0, 50, 100], 100, 40)
    .split(' ')
    .map((p) => p.split(',').map(Number) as [number, number])
  assert.equal(pts.length, 3)
  for (const [x, y] of pts) assert.ok(x >= 0 && x <= 100 && y >= 0 && y <= 40)
  assert.equal(sparkPoints([], 100, 40), '')
})

const term = (id: string, name: string, topics: string[], beyond = false) => ({
  id,
  term: name,
  aliases: [],
  short: 's',
  related: [],
  guide_topics: topics,
  beyond_guide: beyond,
})

test('glossary groups follow the guide order, list a multi-topic term under each topic and put beyond terms last', () => {
  const g = guide()
  const groups = groupGlossary(g, [
    term('z', 'Zed', ['a', 'b']),
    term('y', 'Alpha', ['a']),
    term('r', 'Route', [], true),
  ])
  assert.deepEqual(
    groups.map((s) => s.key),
    ['G1', 'G2', 'beyond'],
  )
  assert.deepEqual(
    groups[0]!.groups.map((x) => [x.key, x.terms.map((t) => t.id)]),
    [
      ['a', ['y', 'z']],
      ['b', ['z']],
    ],
  )
  assert.equal(groups[1]!.groups[0]!.terms.length, 0)
  assert.deepEqual(
    groups[2]!.groups[0]!.terms.map((t) => t.id),
    ['r'],
  )
  assert.equal(primaryGroup(term('z', 'Zed', ['a', 'b'])), 'a')
  assert.equal(primaryGroup(term('r', 'Route', [], true)), 'beyond')
})

test('term search matches names and aliases, and matches are split for highlighting', () => {
  const t = { ...term('x', 'Business Policy', ['a']), aliases: ['BP'] }
  assert.ok(termMatches(t, 'polic') && termMatches(t, 'bp') && !termMatches(t, 'ospf') && termMatches(t, '  '))
  assert.deepEqual(splitMatches('Business Policy', 'polic'), [
    { text: 'Business ', match: false },
    { text: 'Polic', match: true },
    { text: 'y', match: false },
  ])
  assert.deepEqual(splitMatches('abc', ''), [{ text: 'abc', match: false }])
})
