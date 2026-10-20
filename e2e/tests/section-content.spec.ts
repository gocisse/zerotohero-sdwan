import { expect, test } from '@playwright/test'
import { getCourse, getJSON, lessonSource, scanSource, type Diagram } from './support'

// Real-content sections added after Section 2 (id, section that must come right before it).
const SECTIONS: Array<[string, string]> = [
  ['dmpo', 'architecture'],
  ['configuration', 'dmpo'],
  ['overlay', 'configuration'],
  ['routing', 'overlay'],
]

// Per-section checks. Everything is discovered from the API: no lesson, diagram or question counts are hard-coded.
for (const [sid, before] of SECTIONS)
  test.describe(`Section ${sid}`, () => {
    test('sits right after its predecessor, with real lessons', async ({ request }) => {
      const course = await getCourse(request)
      const ids = course.sections.map((s) => s.id)
      expect(ids.indexOf(sid)).toBe(ids.indexOf(before) + 1)
      const s = course.sections.find((x) => x.id === sid)!
      expect(s.placeholder).toBe(false)
      expect(s.lessons.length).toBeGreaterThan(0)
      expect(s.question_count).toBeGreaterThan(0)
    })

    test('every diagram its lessons use has several steps and at least one step with packets', async ({ request }) => {
      const course = await getCourse(request)
      const s = course.sections.find((x) => x.id === sid)!
      const used = new Set<string>()
      for (const l of s.lessons) scanSource(await lessonSource(request, l.id)).diagrams.forEach((d) => used.add(d))
      expect(used.size).toBeGreaterThan(0)
      for (const id of used) {
        const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
        expect(d.steps.length, `${id}: steps`).toBeGreaterThan(1)
        expect(
          d.steps.some((st) => st.packets.length > 0),
          `${id}: packets`,
        ).toBe(true)
      }
    })

    test('the quiz mixes all four question types and the last lesson still links onward', async ({ page, request }) => {
      const qs = await getJSON<{ questions: { type: string }[] }>(request, `/api/sections/${sid}/quiz`)
      const types = new Set(qs.questions.map((q) => q.type))
      for (const t of ['single_choice', 'multi_choice', 'true_false', 'ordering']) expect(types.has(t), t).toBe(true)
      const course = await getCourse(request)
      const last = course.sections.find((x) => x.id === sid)!.lessons.at(-1)!
      await page.goto(`/course/${sid}/${last.id}`)
      await expect(page.getByRole('link', { name: /View your progress|→/ }).last()).toBeVisible()
    })
  })
