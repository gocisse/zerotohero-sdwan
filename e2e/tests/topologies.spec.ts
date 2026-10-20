import { expect, test, type Page } from '@playwright/test'
import { answerUI, getCourse, getJSON, getQuiz, gotoQuestion, lessonSource, lessonUrl, openLesson, scanSource, waitForAttempt, type AttemptResult, type Diagram, type LessonRef } from './support'

// Section 7 (branch site topologies and Edge models, step 17). Lessons, diagrams and questions are discovered through the API;
// the only fixed expectations are the G5 coverage ratings the course assigned, which this step was asked to show.
const SECTION = 'topologies'
const BRIDGE = 'What would you look for in the VCO'
const BOX = 'Check this in your own VCO or site'

// The ratings content/study-guide.yaml assigns to the G5 topics (decision 126).
const G5: Record<string, 'full' | 'partial' | 'none'> = {
  'edge-models': 'partial',
  'branch-site-topologies': 'full',
  'bronze-site': 'full',
  'silver-site-1': 'full',
  'silver-site-2': 'partial',
  'gold-site': 'partial',
}

interface GTopic { id: string; title: string; coverage: 'full' | 'partial' | 'none'; lessons: { id: string; section_id: string }[] }
interface GSection { id: string; topics: GTopic[] }
interface Term { id: string; guide_topics: string[] }

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)

async function sectionLessons(request: Parameters<typeof getCourse>[0]): Promise<LessonRef[]> {
  const course = await getCourse(request)
  const s = course.sections.find((x) => x.id === SECTION)
  expect(s, 'the topologies section exists').toBeTruthy()
  return s!.lessons.map((l) => ({ sectionId: SECTION, id: l.id, title: l.title, placeholder: s!.placeholder }))
}

test.describe.configure({ mode: 'serial' })

test.describe('Section 7: topologies', () => {
  test('sits right after routing with real lessons and a 18 to 20 question quiz of all four types', async ({ request }) => {
    const course = await getCourse(request)
    const ids = course.sections.map((s) => s.id)
    expect(ids.indexOf(SECTION)).toBe(ids.indexOf('routing') + 1)
    const s = course.sections.find((x) => x.id === SECTION)!
    expect(s.placeholder).toBe(false)
    expect(s.lessons.length).toBe(7)
    expect(s.question_count).toBeGreaterThanOrEqual(18)
    expect(s.question_count).toBeLessThanOrEqual(20)
    const qs = await getJSON<{ questions: { type: string; prompt: string }[] }>(request, `/api/sections/${SECTION}/quiz`)
    const types = new Set(qs.questions.map((q) => q.type))
    for (const t of ['single_choice', 'multi_choice', 'true_false', 'ordering']) expect(types.has(t), t).toBe(true)
    expect(qs.questions.filter((q) => /^Scenario/.test(q.prompt)).length, 'scenario questions').toBeGreaterThanOrEqual(6)
    expect(qs.questions.filter((q) => q.type === 'ordering').length).toBeGreaterThanOrEqual(1)
  })

  test('every lesson renders and follows the authoring rules, with the verification box and the bridge', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    test.setTimeout(30_000 + 8_000 * lessons.length)
    for (const l of lessons) {
      const src = scanSource(await lessonSource(request, l.id))
      await openLesson(page, l)
      await page.waitForTimeout(250)
      await expect(page.getByRole('alert'), `${l.id}: error panel`).toHaveCount(0)
      await expect(page.locator('[data-term-error]'), `${l.id}: unknown term`).toHaveCount(0)
      const body = page.locator('article .prose').first()
      await expect(body.locator(':scope > p').first(), `${l.id}: objectives`).toContainText('By the end of this lesson')
      expect(await body.locator(':scope > ul').first().locator(':scope > li').count(), `${l.id}: objective bullets`).toBeGreaterThanOrEqual(3)
      await expect(body.locator('h1'), `${l.id}: no h1`).toHaveCount(0)
      expect(src.predicts.length, `${l.id}: Predict`).toBeGreaterThanOrEqual(1)
      expect(await page.locator('[data-callout]').count(), `${l.id}: Callout`).toBeGreaterThanOrEqual(1)
      expect(src.terms.length, `${l.id}: Term tags`).toBeGreaterThanOrEqual(3)
      await expect(body.getByRole('heading', { name: 'What you would see in the VCO' }), `${l.id}: VCO part`).toHaveCount(1)

      // The "Check this in your own VCO or site" box: a tip Callout with 3 to 5 tagged, structured items.
      const box = page.locator('[data-callout="tip"]', { hasText: BOX })
      await expect(box, `${l.id}: verification box`).toHaveCount(1)
      const items = box.locator('li', { hasText: '[observation candidate]' })
      const n = await items.count()
      expect(n, `${l.id}: observation candidates`).toBeGreaterThanOrEqual(3)
      expect(n, `${l.id}: observation candidates`).toBeLessThanOrEqual(5)
      for (let i = 0; i < n; i++) {
        const t = await items.nth(i).innerText()
        expect(t, `${l.id}: item ${i + 1}`).toContain('Expect:')
        expect(t, `${l.id}: item ${i + 1}`).toContain('Docs may differ:')
      }

      // The lesson ends with the bridge of 2 to 4 questions.
      const headings = await body.locator('h2').allInnerTexts()
      expect(headings.at(-1)?.trim(), `${l.id}: last heading`).toBe(BRIDGE)
      const bridge = await body.locator('h2').last().evaluate((h) => {
        let n = 0
        for (let el = h.nextElementSibling; el; el = el.nextElementSibling) if (el.tagName === 'UL') n += el.querySelectorAll(':scope > li').length
        return n
      })
      expect(bridge, `${l.id}: bridge items`).toBeGreaterThanOrEqual(2)
      expect(bridge, `${l.id}: bridge items`).toBeLessThanOrEqual(4)
    }
  })

  test('every diagram of the section steps to its final step, and the traffic diagrams move packets', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    const owner = new Map<string, LessonRef>()
    for (const l of lessons) for (const d of scanSource(await lessonSource(request, l.id)).diagrams) if (!owner.has(d)) owner.set(d, l)
    expect(owner.size, 'one diagram per topology, one for the datasheet and one decision diagram').toBeGreaterThanOrEqual(6)
    test.setTimeout(30_000 + 20_000 * owner.size)
    let withPackets = 0
    for (const [id, lesson] of owner) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      expect(d.steps.length, `${id}: steps`).toBeGreaterThan(2)
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      let moved = 0
      const packetSteps = d.steps.filter((s) => s.packets.length > 0).length
      for (let s = 1; s <= d.steps.length; s++) {
        await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step ${s} of ${d.steps.length}`)
        if (d.steps[s - 1]!.packets.length > 0) {
          await expect(fig.locator('[data-packet]'), `${id} step ${s}`).toHaveCount(d.steps[s - 1]!.packets.length)
          const pos = () =>
            page.evaluate((x) => {
              const r = document.querySelector(`[data-diagram="${x}"] [data-packet]`)?.getBoundingClientRect()
              return r ? [r.x, r.y] : null
            }, id)
          await page.waitForTimeout(200)
          const a = await pos()
          await page.waitForTimeout(250)
          const b = await pos()
          if (a && b && (Math.abs(a[0]! - b[0]!) > 1 || Math.abs(a[1]! - b[1]!) > 1)) moved++
        }
        if (s < d.steps.length) await fig.getByRole('button', { name: 'Next →' }).click()
      }
      await expect(fig.getByRole('button', { name: 'Next →' }), `${id}: final step reached`).toBeDisabled()
      if (packetSteps > 0) {
        withPackets++
        expect(moved, `${id}: packets animate`).toBeGreaterThan(0)
      }
    }
    expect(withPackets, 'the four topology diagrams show traffic').toBeGreaterThanOrEqual(4)
  })

  test('the quiz can be failed, retried and passed, and a low score blocks nothing', async ({ page, request }) => {
    const questions = await getQuiz(request, SECTION)
    test.setTimeout(120_000)
    await page.goto(`/course/${SECTION}/quiz`)
    await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
    const failing = waitForAttempt(page)
    await page.getByRole('button', { name: 'Submit quiz' }).click()
    await page.getByRole('button', { name: 'Submit anyway' }).click()
    const failed = (await (await failing).json()) as AttemptResult
    expect(failed.score).toBe(0)
    await expect(page.locator('main')).not.toContainText(/\block(ed)?\b|unlock|must pass/i)
    // Continue is enabled and leads on: to the first lesson of the next section, or to the dashboard after the last one.
    const course = await getCourse(request)
    const next = course.sections[course.sections.findIndex((x) => x.id === SECTION) + 1]
    const cont = page.getByRole('link', { name: /Continue to/ })
    await expect(cont).toBeEnabled()
    await expect(cont).toHaveAttribute('href', next?.lessons[0] ? `/course/${next.id}/${next.lessons[0].id}` : '/progress')

    const key = new Map(failed.results.map((r) => [r.question_id, r.correct_answer]))
    await page.goto(`/course/${SECTION}/quiz`)
    await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
    for (const [i, q] of questions.entries()) {
      await gotoQuestion(page, i + 1)
      await answerUI(page, q, key.get(q.id)!)
    }
    const passing = waitForAttempt(page)
    await page.getByRole('button', { name: 'Submit quiz' }).click()
    const passed = (await (await passing).json()) as AttemptResult
    expect(passed.score).toBe(passed.total)
    // Every result names the lesson to revisit, and every lesson_ref is a lesson of this section.
    const lessons = new Set((await sectionLessons(request)).map((l) => l.id))
    for (const r of passed.results) expect(lessons.has(r.lesson_ref ?? ''), `${r.question_id}: lesson_ref`).toBe(true)
  })

  test('the G5 topics show the assigned coverage on the dashboard, with their lessons', async ({ page, request }) => {
    const guide = await getJSON<{ sections: GSection[] }>(request, '/api/study-guide')
    const g5 = guide.sections.find((s) => s.id === 'G5')!
    expect(g5.topics.map((t) => t.id).sort()).toEqual(Object.keys(G5).sort())
    for (const t of g5.topics) {
      expect(t.coverage, `API: ${t.id}`).toBe(G5[t.id])
      expect(t.lessons.length, `${t.id}: mapped lessons`).toBeGreaterThan(0)
      for (const l of t.lessons) expect(l.section_id, `${t.id}: ${l.id}`).toBe(SECTION)
    }
    await page.goto('/progress')
    const card = page.locator('[data-guide-section="G5"]')
    await expect(card.locator('[data-section-mix]')).toHaveText(
      `${g5.topics.filter((t) => t.coverage === 'full').length} full · ${g5.topics.filter((t) => t.coverage === 'partial').length} partial · 0 not covered`,
    )
    await card.locator('h3 button').click()
    for (const t of g5.topics) {
      const row = page.locator(`[data-guide-topic="${t.id}"]`)
      await expect(row, t.id).toHaveAttribute('data-coverage', G5[t.id]!)
      await expect(row.locator('[data-not-covered]')).toHaveCount(0)
      for (const l of t.lessons) await expect(row.locator(`a[href="${lessonUrl({ sectionId: SECTION, id: l.id })}"]`), `${t.id}: link to ${l.id}`).toHaveCount(1)
    }
    await expect(card).not.toContainText('Not covered in this course yet')
  })

  test('the glossary by study guide shows the G5 topics with their terms', async ({ page, request }) => {
    const terms = (await getJSON<{ terms: Term[] }>(request, '/api/glossary')).terms
    await page.goto('/glossary')
    await page.getByRole('button', { name: 'Expand all' }).click()
    for (const topic of Object.keys(G5)) {
      const want = terms.filter((t) => t.guide_topics.includes(topic)).map((t) => t.id)
      expect(want.length, `${topic}: terms exist`).toBeGreaterThan(0)
      const group = page.locator(`[data-guide-topic="${topic}"]`)
      await expect(group, topic).toBeVisible()
      const ids = await group.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
      expect([...ids].sort(), `${topic}: terms shown`).toEqual([...want].sort())
      await expect(group).not.toContainText('No glossary terms for this topic yet')
    }
  })

  test.describe('dark mode on a phone', () => {
    test.use({ colorScheme: 'dark', viewport: { width: 390, height: 844 }, hasTouch: true })
    test('a Section 7 lesson and its wide diagram do not overflow the page', async ({ page, request }) => {
      const lessons = await sectionLessons(request)
      const target = lessons.find((l) => l.id === 'top-silver-2') ?? lessons[0]!
      await openLesson(page, target)
      await page.waitForTimeout(700)
      expect(await overflow(page), `${target.id}: page overflow`).toBeLessThanOrEqual(1)
      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
      const [r, g, b] = (bg.match(/\d+/g) ?? ['255', '255', '255']).map(Number)
      expect((r! + g! + b!) / 3, 'dark background').toBeLessThan(80)
      const fig = page.locator('[data-diagram]').first()
      await fig.scrollIntoViewIfNeeded()
      await fig.getByRole('button', { name: 'Next →' }).click()
      expect(await overflow(page), `${target.id}: overflow after stepping`).toBeLessThanOrEqual(1)
    })
  })
})
