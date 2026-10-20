import { expect, test, type Page } from '@playwright/test'
import { answerUI, getCourse, getJSON, getQuiz, gotoQuestion, lessonSource, lessonUrl, openLesson, scanSource, waitForAttempt, type AttemptResult, type Diagram, type LessonRef } from './support'

// Section 9 (segment, Profile and policy configuration walkthroughs, step 19). Lessons, diagrams, questions and settings tables are
// discovered through the API; the only fixed expectations are the G3 coverage ratings this step assigned.
const SECTION = 'profile-policy-config'
const BRIDGE = 'What would you look for in the VCO'
const BOX = 'Check this in your own VCO'

// content/study-guide.yaml, G3 (decision 138).
const G3: Record<string, 'full' | 'partial' | 'none'> = {
  'segmentation-1': 'full',
  'segmentation-2': 'full',
  'profile-1': 'full',
  'profile-2': 'full',
  'profile-3': 'partial',
  'profile-4': 'partial',
  'policy-1': 'full',
  'policy-2': 'full',
}

interface GTopic { id: string; coverage: 'full' | 'partial' | 'none'; note: string; lessons: { id: string; section_id: string }[] }
interface GSection { id: string; topics: GTopic[] }
interface Term { id: string; guide_topics: string[] }

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)

async function sectionLessons(request: Parameters<typeof getCourse>[0]): Promise<LessonRef[]> {
  const course = await getCourse(request)
  const s = course.sections.find((x) => x.id === SECTION)
  expect(s, 'the network services config section exists').toBeTruthy()
  return s!.lessons.map((l) => ({ sectionId: SECTION, id: l.id, title: l.title, placeholder: s!.placeholder }))
}

/** Row counts of the tables under "## Settings reference" in the lesson source (header and separator rows excluded). */
function sourceSettingsRows(src: string): number {
  let inSection = false
  let header = false
  let rows = 0
  for (const line of src.split('\n')) {
    const t = line.trim()
    if (t.startsWith('## ')) {
      inSection = t.slice(3).trim() === 'Settings reference'
      header = false
    } else if (inSection && t.startsWith('|')) {
      if (/^\|\s*-{3}/.test(t)) continue
      if (!header) header = true
      else rows++
    } else if (inSection && t === '') header = false
  }
  return rows
}

test.describe.configure({ mode: 'serial' })

test.describe('Section 9: segment, Profile and policy configuration', () => {
  test('sits right after network-services-config with 7 real lessons and a 18 to 20 question quiz of all four types', async ({ request }) => {
    const course = await getCourse(request)
    const ids = course.sections.map((s) => s.id)
    expect(ids.indexOf(SECTION)).toBe(ids.indexOf('network-services-config') + 1)
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

  test('every lesson renders with the authoring rules, the settings reference tables, the verification box and the bridge', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    test.setTimeout(30_000 + 8_000 * lessons.length)
    for (const l of lessons) {
      const raw = await lessonSource(request, l.id)
      const src = scanSource(raw)
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

      // The settings reference: a heading, then tables whose rendered rows match the rows of the source.
      const sourceRows = sourceSettingsRows(raw)
      expect(sourceRows, `${l.id}: source settings rows`).toBeGreaterThanOrEqual(3)
      await expect(body.getByRole('heading', { name: 'Settings reference', level: 2 }), `${l.id}: settings heading`).toHaveCount(1)
      const rendered = await body.locator('h2').filter({ hasText: /^Settings reference$/ }).evaluate((h) => {
        let n = 0
        let tables = 0
        for (let el = h.nextElementSibling; el && el.tagName !== 'H2'; el = el.nextElementSibling) {
          const ts = el.tagName === 'TABLE' ? [el] : Array.from(el.querySelectorAll('table'))
          for (const t of ts) {
            tables++
            n += t.querySelectorAll('tbody tr').length
          }
        }
        return { n, tables }
      })
      expect(rendered.tables, `${l.id}: rendered tables`).toBeGreaterThanOrEqual(1)
      expect(rendered.n, `${l.id}: rendered settings rows equal the source rows`).toBe(sourceRows)

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
      const headings = await body.locator('h2').allInnerTexts()
      expect(headings.at(-1)?.trim(), `${l.id}: last heading`).toBe(BRIDGE)
    }
  })

  test('every diagram steps to its final step and packets move where traffic is shown', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    const owner = new Map<string, LessonRef>()
    for (const l of lessons) for (const d of scanSource(await lessonSource(request, l.id)).diagrams) if (!owner.has(d)) owner.set(d, l)
    expect(owner.size, 'segment flow, Profile to Edge, rule build, firewall rule, comparison walk').toBeGreaterThanOrEqual(5)
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
    expect(withPackets, 'the Profile, rule-build and firewall diagrams move packets').toBeGreaterThanOrEqual(3)
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
    const lessons = new Set((await sectionLessons(request)).map((l) => l.id))
    for (const r of passed.results) expect(lessons.has(r.lesson_ref ?? ''), `${r.question_id}: lesson_ref`).toBe(true)
  })

  test('the G3 topics show the assigned coverage on the dashboard, with their lessons', async ({ page, request }) => {
    const guide = await getJSON<{ sections: GSection[] }>(request, '/api/study-guide')
    const g4 = guide.sections.find((s) => s.id === 'G3')!
    expect(g4.topics.map((t) => t.id).sort()).toEqual(Object.keys(G3).sort())
    for (const t of g4.topics) {
      expect(t.coverage, `API: ${t.id}`).toBe(G3[t.id])
      expect(t.lessons.length, `${t.id}: mapped lessons`).toBeGreaterThan(0)
    }
    // Every G3 topic is walked through by a Section 9 lesson, and a partial topic says what is missing.
    for (const t of g4.topics.filter((x) => x.id.endsWith('-config'))) {
      expect(t.lessons.some((l) => l.section_id === SECTION), `${t.id}: a Section 9 lesson`).toBe(true)
      if (t.coverage === 'partial') expect(t.note, `${t.id}: says what is missing`).toMatch(/not covered/i)
    }
    await page.goto('/progress')
    const card = page.locator('[data-guide-section="G3"]')
    await card.locator('h3 button').click()
    for (const t of g4.topics) {
      const row = page.locator(`[data-guide-topic="${t.id}"]`)
      await expect(row, t.id).toHaveAttribute('data-coverage', G3[t.id]!)
      for (const l of t.lessons.filter((x) => x.section_id === SECTION)) {
        await expect(row.locator(`a[href="${lessonUrl({ sectionId: SECTION, id: l.id })}"]`), `${t.id}: link to ${l.id}`).toHaveCount(1)
      }
    }
  })

  test('the glossary by study guide shows the G3 topics with their terms', async ({ page, request }) => {
    const terms = (await getJSON<{ terms: Term[] }>(request, '/api/glossary')).terms
    await page.goto('/glossary')
    await page.getByRole('button', { name: 'Expand all' }).click()
    let withTerms = 0
    for (const topic of ['segmentation-2', 'profile-2', 'profile-3', 'policy-2']) {
      const want = terms.filter((t) => t.guide_topics.includes(topic)).map((t) => t.id)
      const group = page.locator(`[data-guide-topic="${topic}"]`)
      await expect(group, topic).toBeVisible()
      const ids = await group.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
      expect([...ids].sort(), `${topic}: terms shown`).toEqual([...want].sort())
      if (want.length > 0) withTerms++
    }
    expect(withTerms, 'the configuration topics have glossary terms').toBeGreaterThanOrEqual(3)
  })

  test('the Try it in the simulator links of the lessons open real mock Edge screens', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    const links = new Set<string>()
    for (const l of lessons) for (const m of (await lessonSource(request, l.id)).matchAll(/\]\((\/sim\/[^)\s]+)\)/g)) links.add(m[1]!)
    expect(links.size, 'simulator links in Section 9').toBeGreaterThanOrEqual(5)
    test.setTimeout(30_000 + 8_000 * links.size)
    for (const href of links) {
      await page.goto(href)
      await expect(page.locator('[data-sim-banner]'), href).toBeVisible()
      await expect(page.getByRole('alert'), href).toHaveCount(0)
    }
  })

  test.describe('dark mode on a phone', () => {
    test.use({ colorScheme: 'dark', viewport: { width: 390, height: 844 }, hasTouch: true })
    test('a Section 9 lesson with wide tables does not overflow the page', async ({ page, request }) => {
      const lessons = await sectionLessons(request)
      for (const id of ['prof-device', 'prof-bp-rules']) {
        const target = lessons.find((l) => l.id === id) ?? lessons[0]!
        await openLesson(page, target)
        await page.waitForTimeout(700)
        expect(await overflow(page), `${target.id}: page overflow`).toBeLessThanOrEqual(1)
        const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
        const [r, g, b] = (bg.match(/\d+/g) ?? ['255', '255', '255']).map(Number)
        expect((r! + g! + b!) / 3, 'dark background').toBeLessThan(80)
      }
    })
  })
})
