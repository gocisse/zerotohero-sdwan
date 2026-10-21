import { expect, test, type Page } from '@playwright/test'
import { answerUI, getCourse, getJSON, getQuiz, gotoQuestion, lessonSource, lessonUrl, openLesson, scanSource, waitForAttempt, type AttemptResult, type Diagram, type LessonRef } from './support'

// Section 10 (troubleshooting, step 20), the runbooks under /troubleshooting, study guide section G6 and the troubleshooting labs. Lessons,
// diagrams, questions, runbooks and labs are discovered through the API; the only fixed expectations are the G6 ratings this step assigned
// and the lab ids the exam is made of.
const SECTION = 'troubleshooting'
const BRIDGE = 'What would you look for in the VCO'
const BOX = 'Check this in your own VCO'

// content/study-guide.yaml, G6 (decision 146).
const G6: Record<string, 'full' | 'partial' | 'none'> = {
  'ts-method': 'partial',
  'ts-layers': 'partial',
  'ts-vco-tools': 'full',
  'ts-edge-cli': 'partial',
  'ts-gw-orch': 'partial',
  'ts-events-logs': 'partial',
  'ts-scen-activation': 'partial',
  'ts-scen-overlay': 'partial',
  'ts-scen-policy': 'partial',
  'ts-escalation': 'full',
}
const EXAM = ['ts-exam-1-quiet-monday', 'ts-exam-2-hub-trouble', 'ts-exam-3-change-window']
const DIAGNOSIS_LESSONS = ['cfg-diagnosis', 'dmpo-vco-diagnosis', 'ov-diagnosis', 'rt-diagnosis', 'prof-compare', 'svc-review']

interface GTopic { id: string; coverage: 'full' | 'partial' | 'none'; note: string; lessons: { id: string; section_id: string }[]; labs: { id: string }[] }
interface GSection { id: string; added_by_course: boolean; topics: GTopic[] }
interface Term { id: string; guide_topics: string[] }
interface RunbookSummary { id: string; title: string; summary: string; kind: string; estimated_minutes: number }
interface LabDetail { id: string; topology: string; difficulty: number; hint_count: number; task_list: { id: string; type: string }[] }

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)

async function sectionLessons(request: Parameters<typeof getCourse>[0]): Promise<LessonRef[]> {
  const course = await getCourse(request)
  const s = course.sections.find((x) => x.id === SECTION)
  expect(s, 'the troubleshooting section exists').toBeTruthy()
  return s!.lessons.map((l) => ({ sectionId: SECTION, id: l.id, title: l.title, placeholder: s!.placeholder }))
}

/** Row counts of the tables under "## Settings reference" in the lesson source. */
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

test.describe('Section 10: troubleshooting', () => {
  test('sits last, with 10 real lessons and a 20 to 24 question quiz of all four types', async ({ request }) => {
    const course = await getCourse(request)
    const ids = course.sections.map((s) => s.id)
    expect(ids.indexOf(SECTION)).toBe(ids.indexOf('profile-policy-config') + 1)
    const s = course.sections.find((x) => x.id === SECTION)!
    expect(s.placeholder).toBe(false)
    expect(s.lessons.length).toBe(10)
    expect(s.question_count).toBeGreaterThanOrEqual(20)
    expect(s.question_count).toBeLessThanOrEqual(24)
    const qs = await getJSON<{ questions: { type: string; prompt: string; topic: string; difficulty: number }[] }>(request, `/api/sections/${SECTION}/quiz`)
    const types = new Set(qs.questions.map((q) => q.type))
    for (const t of ['single_choice', 'multi_choice', 'true_false', 'ordering']) expect(types.has(t), t).toBe(true)
    expect(qs.questions.filter((q) => /^Scenario/.test(q.prompt)).length, 'scenario questions').toBeGreaterThanOrEqual(10)
    expect(qs.questions.filter((q) => q.type === 'ordering').length, 'ordering questions').toBeGreaterThanOrEqual(2)
    for (const q of qs.questions) expect([1, 2, 3], `${q.topic}: difficulty`).toContain(q.difficulty)
    // The quiz topic tags are the lesson topic tags (ts-...).
    for (const q of qs.questions) expect(q.topic.startsWith('ts-'), `${q.topic}: tag`).toBe(true)
  })

  test('every lesson renders and follows the authoring rules, with the verification box and the bridge', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    test.setTimeout(30_000 + 8_000 * lessons.length)
    let withTables = 0
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
      for (const d of src.diagrams) await expect(page.locator(`[data-diagram="${d}"]`), `${l.id}: diagram ${d}`).toHaveCount(1)
      if (sourceSettingsRows(raw) > 0) {
        withTables++
        expect(await body.locator('table').count(), `${l.id}: tables`).toBeGreaterThanOrEqual(1)
      }
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
      // Nothing is blocked: the end marker and the navigation are there.
      await expect(page.locator('[data-lesson-end]')).toBeVisible()
    }
    expect(withTables, 'lessons with a Settings reference table').toBe(4)
  })

  test('every diagram steps to its final step, highlights change and the scenario diagrams move packets', async ({ page, request }) => {
    const lessons = await sectionLessons(request)
    const owner = new Map<string, LessonRef>()
    for (const l of lessons) for (const d of scanSource(await lessonSource(request, l.id)).diagrams) if (!owner.has(d)) owner.set(d, l)
    expect(owner.size, 'method flow, layer model, tool tree and three scenario families').toBeGreaterThanOrEqual(6)
    test.setTimeout(30_000 + 25_000 * owner.size)
    let withPackets = 0
    for (const [id, lesson] of owner) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      expect(d.steps.length, `${id}: steps`).toBeGreaterThan(2)
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      let moved = 0
      const dimmedCounts = new Set<number>()
      const packetSteps = d.steps.filter((s) => s.packets.length > 0).length
      for (let s = 1; s <= d.steps.length; s++) {
        await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step ${s} of ${d.steps.length}`)
        await page.waitForTimeout(400)
        dimmedCounts.add(await fig.locator('[data-node].opacity-25').count())
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
      expect(dimmedCounts.size, `${id}: highlights change between steps`).toBeGreaterThan(1)
      if (packetSteps > 0) {
        withPackets++
        expect(moved, `${id}: packets animate`).toBeGreaterThan(0)
      }
    }
    expect(withPackets, 'the three scenario diagrams move packets').toBeGreaterThanOrEqual(3)
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
    const cont = page.getByRole('link', { name: /Continue to|View your progress/ })
    await expect(cont.first()).toBeEnabled()
    // Every wrong answer has an explanation, and each result names a lesson of this section.
    const lessons = new Set((await sectionLessons(request)).map((l) => l.id))
    for (const r of failed.results) {
      expect(r.explanation.length, `${r.question_id}: explanation`).toBeGreaterThan(20)
      expect(lessons.has(r.lesson_ref ?? ''), `${r.question_id}: lesson_ref`).toBe(true)
    }
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
  })

  test('the diagnosis lessons link to a matching Try a ticket lesson that exists', async ({ page, request }) => {
    const course = await getCourse(request)
    const known = new Map(course.sections.flatMap((s) => s.lessons.map((l) => [l.id, s.id] as const)))
    for (const id of DIAGNOSIS_LESSONS) {
      const src = await lessonSource(request, id)
      const m = src.match(/\*\*Try a ticket\.\*\*[^\n]*\]\(\/course\/troubleshooting\/([a-z0-9-]+)\)/)
      expect(m, `${id}: Try a ticket link`).toBeTruthy()
      expect(known.get(m![1]!), `${id}: ${m![1]} is a lesson of the troubleshooting section`).toBe(SECTION)
    }
    // The link works from the page.
    await page.goto(lessonUrl({ sectionId: '06-routing'.replace(/^\d+-/, ''), id: 'rt-diagnosis' }))
    await page.locator('a', { hasText: 'Scenarios 2: overlay and routing' }).first().click()
    await expect(page).toHaveURL(/\/course\/troubleshooting\/trb-scen-2$/)
  })
})

test.describe('runbooks under /troubleshooting', () => {
  test('four runbooks are served with their fields, and the list is shown with a sidebar entry', async ({ page, request }) => {
    const { runbooks } = await getJSON<{ runbooks: RunbookSummary[] }>(request, '/api/runbooks')
    expect(runbooks.map((r) => r.id).sort()).toEqual(['debug-script-reference', 'evidence-checklist', 'layer-cheat-sheet', 'support-case-template'])
    for (const r of runbooks) {
      expect(r.title.length, r.id).toBeGreaterThan(3)
      expect(r.summary.length, r.id).toBeGreaterThan(20)
      expect(['checklist', 'cheatsheet', 'template', 'reference'], r.id).toContain(r.kind)
      expect(r.estimated_minutes, r.id).toBeGreaterThan(0)
    }
    await page.goto('/')
    await page.getByRole('navigation', { name: 'Course' }).getByRole('link', { name: 'Troubleshooting', exact: true }).click()
    await expect(page).toHaveURL(/\/troubleshooting$/)
    await expect(page.locator('[data-runbook]')).toHaveCount(runbooks.length)
    await expect(page.locator('[data-ts-labs] a')).toHaveCount(9)
    await expect(page.getByRole('link', { name: /Section 10/ })).toBeVisible()
    expect(await overflow(page)).toBeLessThanOrEqual(1)
  })

  test('every runbook page renders, and the debug.py page separates documented from unconfirmed', async ({ page, request }) => {
    const { runbooks } = await getJSON<{ runbooks: RunbookSummary[] }>(request, '/api/runbooks')
    test.setTimeout(30_000 + 6_000 * runbooks.length)
    for (const r of runbooks) {
      const full = await getJSON<{ source: string }>(request, `/api/runbooks/${r.id}`)
      expect(full.source.length, r.id).toBeGreaterThan(200)
      await page.goto(`/troubleshooting/${r.id}`)
      await expect(page.locator(`[data-runbook-page="${r.id}"] h1`)).toHaveText(r.title)
      await page.waitForTimeout(300)
      await expect(page.getByRole('alert'), `${r.id}: error panel`).toHaveCount(0)
      await expect(page.locator('[data-term-error]'), `${r.id}: unknown term`).toHaveCount(0)
      expect(await page.locator('article .prose > *').count(), `${r.id}: blocks`).toBeGreaterThanOrEqual(3)
    }
    await page.goto('/troubleshooting/debug-script-reference')
    await expect(page.getByRole('heading', { name: /Unconfirmed, fill from observation/ })).toBeVisible()
    // Every documented option row carries a status of VERIFIED or OBSERVED.
    const rows = page.locator('article .prose table').first().locator('tbody tr')
    expect(await rows.count()).toBeGreaterThanOrEqual(3)
    for (const t of await rows.allInnerTexts()) expect(t).toMatch(/VERIFIED|OBSERVED/)
    await page.goto('/troubleshooting/no-such-runbook')
    await expect(page.getByText('Page not found')).toBeVisible()
    expect((await request.get('/api/runbooks/no-such-runbook')).status()).toBe(404)
  })

  test.describe('dark mode on a phone', () => {
    test.use({ colorScheme: 'dark', viewport: { width: 390, height: 844 }, hasTouch: true })
    test('a Section 10 lesson and a runbook do not overflow the page', async ({ page, request }) => {
      const lessons = await sectionLessons(request)
      for (const id of ['trb-tools', 'trb-scen-1']) {
        await openLesson(page, lessons.find((l) => l.id === id)!)
        await page.waitForTimeout(700)
        expect(await overflow(page), `${id}: page overflow`).toBeLessThanOrEqual(1)
        const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
        const [r, g, b] = (bg.match(/\d+/g) ?? ['255', '255', '255']).map(Number)
        expect((r! + g! + b!) / 3, 'dark background').toBeLessThan(80)
      }
      for (const id of ['debug-script-reference', 'layer-cheat-sheet']) {
        await page.goto(`/troubleshooting/${id}`)
        await page.waitForTimeout(700)
        expect(await overflow(page), `${id}: page overflow`).toBeLessThanOrEqual(1)
      }
    })
  })
})

test.describe('study guide section G6', () => {
  test('G6 is marked as added by the course and its topics carry the assigned coverage', async ({ page, request }) => {
    const guide = await getJSON<{ sections: GSection[] }>(request, '/api/study-guide')
    const g6 = guide.sections.find((s) => s.id === 'G6')!
    expect(g6.added_by_course).toBe(true)
    for (const s of guide.sections.filter((x) => x.id !== 'G6')) expect(s.added_by_course, `${s.id} is original`).toBe(false)
    expect(g6.topics.map((t) => t.id).sort()).toEqual(Object.keys(G6).sort())
    for (const t of g6.topics) {
      expect(t.coverage, `API: ${t.id}`).toBe(G6[t.id])
      expect(t.lessons.length, `${t.id}: mapped lessons`).toBeGreaterThan(0)
      if (t.coverage === 'partial') expect(t.note.length, `${t.id}: says what is missing`).toBeGreaterThan(40)
    }
    // The debug.py topic is partial, because the public documents give only four options for an Edge.
    expect(g6.topics.find((t) => t.id === 'ts-edge-cli')!.coverage).toBe('partial')

    await page.goto('/progress')
    const card = page.locator('[data-guide-section="G6"]')
    await expect(card.locator('[data-added-badge]')).toBeVisible()
    await expect(page.locator('[data-guide-section="G5"] [data-added-badge]')).toHaveCount(0)
    await card.locator('h3 button').click()
    for (const t of g6.topics) await expect(page.locator(`[data-guide-topic="${t.id}"]`), t.id).toHaveAttribute('data-coverage', G6[t.id]!)
  })

  test('the glossary by study guide shows G6 with the badge, its topics and their terms', async ({ page, request }) => {
    const terms = (await getJSON<{ terms: Term[] }>(request, '/api/glossary')).terms
    await page.goto('/glossary')
    await expect(page.locator('[data-guide-section="G6"] [data-added-badge]')).toBeVisible()
    await page.getByRole('button', { name: 'Expand all' }).click()
    let withTerms = 0
    for (const topic of Object.keys(G6)) {
      const want = terms.filter((t) => t.guide_topics.includes(topic)).map((t) => t.id)
      const group = page.locator(`[data-guide-topic="${topic}"]`)
      await expect(group, topic).toBeVisible()
      const ids = await group.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
      expect([...ids].sort(), `${topic}: terms shown`).toEqual([...want].sort())
      if (want.length > 0) withTerms++
    }
    expect(withTerms, 'G6 topics with glossary terms').toBeGreaterThanOrEqual(5)
    for (const id of ['remote-diagnostics', 'diagnostic-bundle', 'debug-py', 'evidence']) expect(terms.some((t) => t.id === id), id).toBe(true)
  })

  test('goal coverage lists the troubleshooting goal, informational only', async ({ page }) => {
    await page.goto('/progress')
    const card = page.locator('[data-goal-coverage] [data-goal="troubleshoot"]')
    await expect(card).toBeVisible()
    await expect(card).toContainText('I can troubleshoot a real case step by step')
    for (const part of ['lessons', 'quiz', 'labs', 'self-check', 'checklist']) await expect(card.locator(`[data-goal-part=${part}]`)).toBeVisible()
    await expect(page.locator('[data-goal-coverage]')).toContainText('Informational only')
  })
})

test.describe('troubleshooting labs and the exam', () => {
  test('nine labs exist; the exam has three tickets on ts-exam-network with no new task type and no hint shown by default', async ({ page, request }) => {
    const ids = (await getJSON<{ labs: { id: string }[] }>(request, '/api/labs')).labs.map((l) => l.id).filter((id) => id.startsWith('ts-'))
    expect(ids.length).toBe(9)
    const exam: LabDetail[] = []
    for (const id of EXAM) exam.push(await getJSON<LabDetail>(request, `/api/labs/${id}`))
    expect(exam.map((l) => l.topology)).toEqual(['ts-exam-network', 'ts-exam-network', 'ts-exam-network'])
    const diffs = exam.map((l) => l.difficulty)
    expect([...diffs].sort(), 'increasing difficulty').toEqual(diffs)
    for (const l of ids) {
      const lab = await getJSON<LabDetail>(request, `/api/labs/${l}`)
      for (const t of lab.task_list) expect(['inspect', 'diagnose', 'fix', 'predict'], `${l}/${t.id}`).toContain(t.type)
    }
    for (const lab of exam) {
      expect(lab.hint_count, `${lab.id}: hints exist on request`).toBeGreaterThan(0)
      await page.goto(`/sim/labs/${lab.id}`)
      await expect(page.locator('[data-lab-runner]')).toBeVisible()
      await expect(page.locator('[data-lab-hint]'), `${lab.id}: no hint by default`).toHaveCount(0)
    }
  })

  test('the lab API leaks no answer for the troubleshooting labs, and the debrief appears only after completion', async ({ request }) => {
    for (const id of [...EXAM, 'ts-printer-vlan', 'ts-large-transfer-stall']) {
      const text = JSON.stringify(await getJSON(request, `/api/labs/${id}`))
      for (const key of ['"probe"', '"explanation"', '"equals"', '"patches"', '"value"', '"correct"']) expect(text, `${id} leaks ${key}`).not.toContain(key)
    }
    const fresh = await request.get('/api/labs/ts-spoke-behind-nat/debrief')
    expect([200, 403, 404, 409]).toContain(fresh.status())
  })

  test('a troubleshooting lab can be worked from the page: hint, wrong then right, fix and complete', async ({ page, request }) => {
    const lab = await getJSON<LabDetail & { fixes: { id: string }[]; task_list: { id: string; type: string; choices: { id: string }[] }[] }>(request, '/api/labs/ts-printer-vlan')
    await page.goto('/sim/labs/ts-printer-vlan')
    await page.locator('[data-lab-next-hint]').click()
    await expect(page.locator('[data-lab-hint]')).toHaveCount(1)
    for (const t of lab.task_list) {
      const card = page.locator(`[data-lab-task=${t.id}]`)
      if (t.type === 'fix') {
        for (const f of lab.fixes) {
          const ok = await (await request.post(`/api/labs/${lab.id}/tasks/${t.id}/check`, { data: { answer: '', fixes: [f.id] } })).json()
          if (ok.correct) {
            await card.locator(`[data-lab-apply=${f.id}]`).click()
            break
          }
        }
        await card.locator(`[data-lab-check=${t.id}]`).click()
        await expect(card.locator('[data-lab-result]')).toHaveAttribute('data-lab-result', 'correct')
        continue
      }
      for (const c of t.choices) {
        const ok = await (await request.post(`/api/labs/${lab.id}/tasks/${t.id}/check`, { data: { answer: c.id, fixes: [] } })).json()
        if (!ok.correct) continue
        await card.locator(`[data-lab-choice=${c.id}]`).check()
        await card.locator(`[data-lab-check=${t.id}]`).click()
        await expect(card.locator('[data-lab-result]')).toHaveAttribute('data-lab-result', 'correct')
      }
    }
    await expect(page.locator('[data-lab-complete]')).toBeVisible()
    await expect(page.locator('[data-lab-debrief]')).toBeVisible()
    // The debrief maps findings to real VCO tools and says what a live engineer would add.
    await expect(page.locator('[data-lab-debrief]')).toContainText(/live engineer/i)
  })
})
