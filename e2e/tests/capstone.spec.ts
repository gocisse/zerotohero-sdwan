import { expect, test, type APIRequestContext } from '@playwright/test'
import { existsSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { getJSON } from './support'

// Step 15: capstone labs, self-check, goal coverage, the VCO bridge of every lesson and the observation validators.
// Everything is discovered through the API; nothing about labs or questions is hard-coded except the capstone prefix.
interface LabSummary { id: string; title: string }
interface Progress { sections: { section_id: string; quiz_attempts: number }[]; labs: { completed: number } }
interface SelfCheckApi {
  questions: { id: string; section_id: string; topic: string; type: string }[]
  screens: { id: string; choices: { id: string }[]; screen: { path: string }; topology: string }[]
  lab: string
}

const contentDir = join(process.cwd(), '..', 'content')
const tempObservation = join(contentDir, 'reference', 'observations', 'zz-e2e-temp.md')
const cleanup = () => existsSync(tempObservation) && rmSync(tempObservation)

test.describe.configure({ mode: 'serial' })
test.afterAll(cleanup)

const stateSnapshot = async (request: APIRequestContext) => {
  const p = await getJSON<Progress>(request, '/api/progress')
  const topics = await getJSON<{ topics: { topic: string; answers: number }[] }>(request, '/api/progress/topics')
  const review = await getJSON<{ due_count: number; upcoming_count: number }>(request, '/api/review/queue')
  return JSON.stringify({ attempts: p.sections.map((s) => s.quiz_attempts), labs: p.labs.completed, topics: topics.topics, due: review.due_count, upcoming: review.upcoming_count })
}

test.describe('capstone', () => {
  test('the labs list has a Capstone entry with the three connected scenarios, and a card ends the last section', async ({ page, request }) => {
    const labs = (await getJSON<{ labs: LabSummary[] }>(request, '/api/labs')).labs
    const capstone = labs.filter((l) => l.id.startsWith('capstone-'))
    expect(capstone.length).toBeGreaterThanOrEqual(3)
    expect(labs.some((l) => l.id.startsWith('self-check-')), 'the self-check screen tasks are not labs').toBeFalsy()
    await page.goto('/sim/labs')
    const section = page.locator('[data-capstone-section]')
    await expect(section).toBeVisible()
    for (const l of capstone) await expect(section.locator(`[data-lab-card=${l.id}]`)).toBeVisible()
    // The other labs are listed outside the capstone box.
    await expect(page.locator('[data-lab-card]')).toHaveCount(labs.length)

    // The capstone card at the end of the last section's last lesson; never required.
    const course = await getJSON<{ sections: { id: string; lessons: { id: string }[] }[] }>(request, '/api/course')
    const last = course.sections[course.sections.length - 1]!
    await page.goto(`/course/${last.id}/${last.lessons[last.lessons.length - 1]!.id}`)
    await expect(page.locator('[data-capstone-card]')).toBeVisible()
    await expect(page.locator('[data-capstone-card]')).toContainText('optional')
    await page.locator('[data-capstone-open]').click()
    await expect(page).toHaveURL(/\/sim\/labs#capstone$/)
    await expect(page.locator('[data-capstone-section]')).toBeVisible()
    // The card is also on the last section's page, and no earlier section shows it.
    await page.goto(`/course/${last.id}`)
    await expect(page.locator('[data-capstone-card]')).toBeVisible()
    await page.goto(`/course/${course.sections[0]!.id}`)
    await expect(page.locator('[data-capstone-card]')).toHaveCount(0)
  })

  test('a capstone lab shows its debrief only after it is completed, with unverified paths marked', async ({ page, request }) => {
    const id = (await getJSON<{ labs: LabSummary[] }>(request, '/api/labs')).labs.find((l) => l.id.startsWith('capstone-'))!.id
    const put = (status: string) => request.put(`/api/labs/${id}/status`, { data: { status } })
    // Completed is never downgraded, so a database kept from an earlier run already serves the debrief.
    const status = (await getJSON<{ progress: { status: string } }>(request, `/api/labs/${id}`)).progress.status
    if (status !== 'completed') {
      expect((await put('started')).ok()).toBeTruthy()
      const before = await getJSON<{ available: boolean; items: unknown[] }>(request, `/api/labs/${id}/debrief`)
      expect(before.available).toBeFalsy()
      expect(before.items).toHaveLength(0)
      await page.goto(`/sim/labs/${id}`)
      await expect(page.locator('[data-lab-debrief]')).toHaveCount(0)
    }
    expect((await put('completed')).ok()).toBeTruthy()
    const after = await getJSON<{ available: boolean; items: { unverified: boolean; where: string }[] }>(request, `/api/labs/${id}/debrief`)
    expect(after.available).toBeTruthy()
    await page.goto(`/sim/labs/${id}`)
    const box = page.locator('[data-lab-debrief]')
    await expect(box).toBeVisible()
    await expect(box.locator('[data-lab-debrief-item]')).toHaveCount(after.items.length)
    const unverified = after.items.filter((i) => i.unverified).length
    expect(unverified).toBeGreaterThan(0)
    await expect(box.locator('[data-lab-debrief-item=unverified]')).toHaveCount(unverified)
    await expect(box).toContainText('unverified path')
  })
})

test.describe('self-check', () => {
  test('is ungraded: 12 questions across sections, 3 screen tasks, a local checklist, nothing stored on the server', async ({ page, request }) => {
    const before = await stateSnapshot(request)
    const api = await getJSON<SelfCheckApi>(request, '/api/self-check')
    expect(api.questions).toHaveLength(12)
    expect(new Set(api.questions.map((q) => q.section_id)).size).toBeGreaterThanOrEqual(5)
    expect(new Set(api.questions.map((q) => q.id)).size).toBe(12)
    expect(api.screens).toHaveLength(3)
    // Questions are drawn at random and an option may legitimately have the id "probe" or "value", so the lab-format keys are
    // only searched for in the screen tasks; "explanation" and "correct" must appear nowhere.
    for (const k of ['"explanation"', '"correct"']) expect(JSON.stringify(api)).not.toContain(k)
    for (const k of ['"value"', '"probe"', '"equals"']) expect(JSON.stringify(api.screens)).not.toContain(k)

    await page.goto('/sim/self-check')
    await expect(page.locator('[data-sim-banner]')).toBeVisible()
    await expect(page.locator('[data-self-check-note]')).toContainText('Nothing is scored')
    await expect(page.locator('[data-self-check-question]')).toHaveCount(12)
    await expect(page.locator('[data-self-check-screen]')).toHaveCount(3)
    await expect(page.locator('[data-checklist-goal]')).toHaveCount(4)

    // Answer the first single-choice or true/false question, then check: feedback comes back and no state changes.
    // (The page draws its own self-check, so pick from what it shows: the first question that has inputs.)
    const firstQ = page.locator('[data-self-check-question]').filter({ has: page.locator('input') }).first()
    await firstQ.locator('input').first().check()
    await page.locator('[data-self-check-submit]').click()
    await expect(page.locator('[data-self-check-summary]')).toContainText('not a grade')
    await expect(firstQ.locator('[data-result]')).toBeVisible()

    // Screen tasks: open the screen with the self-check network, check a wrong and then the right answer through the API.
    const task = api.screens[0]! // the screen tasks are fixed, so the API copy matches the page
    const card = page.locator(`[data-self-check-screen=${task.id}]`)
    const link = card.locator('[data-self-check-screen-link]')
    await expect(link).toHaveAttribute('href', new RegExp(`/sim/${task.topology}/.*lab=${api.lab}`))
    let rightId = ''
    for (const c of task.choices) {
      const r = await request.post(`/api/self-check/screens/${task.id}/check`, { data: { answer: c.id } })
      expect(r.ok()).toBeTruthy()
      if (((await r.json()) as { correct: boolean }).correct) rightId = c.id
    }
    expect(rightId).not.toBe('')
    const wrong = task.choices.find((c) => c.id !== rightId)!
    await card.locator(`[data-self-check-choice=${wrong.id}]`).check()
    await card.locator(`[data-self-check-screen-check=${task.id}]`).click()
    await expect(card.locator('[data-self-check-screen-result]')).toHaveAttribute('data-self-check-screen-result', 'incorrect')
    await card.locator(`[data-self-check-choice=${rightId}]`).check()
    await card.locator(`[data-self-check-screen-check=${task.id}]`).click()
    await expect(card.locator('[data-self-check-screen-result]')).toHaveAttribute('data-self-check-screen-result', 'correct')

    // The linked screen shows the self-check network, with a way back.
    await link.click()
    await expect(page.locator('[data-lab-banner]')).toContainText('self-check')
    await expect(page.locator('[data-sim-banner]')).toBeVisible()

    expect(await stateSnapshot(request), 'the self-check must store nothing on the server').toBe(before)
  })

  test('the checklist is ticked by hand and stays in this browser only', async ({ page, context }) => {
    await page.goto('/sim/self-check')
    const items = page.locator('[data-checklist-item]')
    expect(await items.count()).toBeGreaterThanOrEqual(8)
    await items.nth(0).check()
    await items.nth(3).check()
    await expect(page.locator('[data-checklist-count]')).toContainText('2 of')
    await page.reload()
    await expect(page.locator('[data-checklist-item]').nth(0)).toBeChecked()
    await expect(page.locator('[data-checklist-count]')).toContainText('2 of')
    // A fresh browser context starts empty: nothing was sent to the server.
    const other = await context.browser()!.newContext({ baseURL: page.url().split('/sim')[0] })
    const p2 = await other.newPage()
    await p2.goto('/sim/self-check')
    await expect(p2.locator('[data-checklist-count]')).toContainText('0 of')
    await other.close()
  })

  test('the dashboard shows goal coverage for the four goals and it is informational only', async ({ page }) => {
    await page.goto('/progress')
    const view = page.locator('[data-goal-coverage]')
    await expect(view).toBeVisible()
    await expect(view).toContainText('Informational only')
    for (const g of ['read-config', 'traffic-flow', 'tunnels', 'routes']) {
      const card = view.locator(`[data-goal=${g}]`)
      await expect(card).toBeVisible()
      for (const part of ['lessons', 'quiz', 'labs', 'self-check', 'checklist']) await expect(card.locator(`[data-goal-part=${part}]`)).toBeVisible()
      await expect(card.locator('[role=progressbar]')).toBeVisible()
    }
    // The ticked checklist items of the previous test are counted.
    await expect(view.locator('[data-goal-part=checklist]').first()).toContainText('of')
    // Nothing here disables anything: the review and quiz links stay available.
    await expect(page.locator('a[href="/review"]').first()).toBeVisible()
    await expect(view.locator('[data-goal-self-check]')).toBeVisible()
  })

  test('every goal names sections, labs and topics that exist', async ({ request }) => {
    const course = await getJSON<{ sections: { id: string }[] }>(request, '/api/course')
    const labs = (await getJSON<{ labs: { id: string }[] }>(request, '/api/labs')).labs.map((l) => l.id)
    const topics = new Set<string>()
    for (const s of course.sections) for (const q of (await getJSON<{ questions: { topic: string }[] }>(request, `/api/sections/${s.id}/quiz`)).questions) topics.add(q.topic)
    const src = await (await request.get('/src/progress/goals.ts')).text() // served by the Vite dev server
    const list = (key: string) => [...src.matchAll(new RegExp(`${key}:\\s*\\[([^\\]]*)\\]`, 'g'))].flatMap((m) => [...m[1]!.matchAll(/['"]([^'"]+)['"]/g)].map((x) => x[1]!))
    const sections = list('sections')
    const goalLabs = list('labs')
    const goalTopics = list('topics')
    expect(sections.length).toBeGreaterThan(4)
    expect(goalLabs.length).toBeGreaterThan(8)
    expect(goalTopics.length).toBeGreaterThan(8)
    for (const s of sections) expect(course.sections.some((c) => c.id === s), `section ${s}`).toBeTruthy()
    for (const l of goalLabs) expect(labs, `lab ${l}`).toContain(l)
    for (const t of goalTopics) {
      const hit = t.endsWith('-') ? [...topics].some((x) => x.startsWith(t)) : topics.has(t)
      expect(hit, `quiz topic ${t}`).toBeTruthy()
    }
  })
})

test.describe('learning path', () => {
  test('every lesson ends with the "what would you look for in the VCO" bridge', async ({ request }) => {
    const course = await getJSON<{ sections: { id: string; lessons: { id: string }[] }[] }>(request, '/api/course')
    let n = 0
    for (const s of course.sections)
      for (const l of s.lessons) {
        const src = (await getJSON<{ source: string }>(request, `/api/lessons/${l.id}`)).source
        const heads = src.split('\n').filter((x) => x.startsWith('## '))
        expect(heads[heads.length - 1], l.id).toBe('## What would you look for in the VCO')
        n++
      }
    expect(n).toBeGreaterThanOrEqual(42)
  })
})

test.describe('observation validators', () => {
  const broken = `# zz temp observations\n\n## Observation 1: broken on purpose\n\n- Release and build: \n- Screen / menu path: A > B\n- What was done: x\n- What was seen: y\n- Resolves: no-such-sheet: nothing here\n- Resolves: assumption 9999\n- Effect: confirms\n\n| claim | source URL | doc version | status |\n|---|---|---|---|\n| z | user observation 2026-10-20 | 7.0 (lab) | OBSERVED |\n`
  const valid = `# zz temp observations\n\n## Observation 1: valid\n\n- Date: 2026-10-20\n- Release and build: 7.0.0.2 build TEMP (e2e fixture, not a real observation)\n- Screen / menu path: A > B\n- What was done: x\n- What was seen: y\n- Resolves: assumption 22\n- Effect: confirms\n\n| claim | source URL | doc version | status |\n|---|---|---|---|\n| z | user observation 2026-10-20 | 7.0 (lab) | OBSERVED |\n`

  test('a broken observation file fails validation with every problem listed, and a valid one passes', async ({ request }) => {
    try {
      writeFileSync(tempObservation, broken)
      const res = await request.get('/api/course')
      expect(res.status()).toBe(503)
      const body = (await res.json()) as { error: string; details: { file: string; message: string }[] }
      expect(body.error).toBe('content_invalid')
      const msgs = body.details.filter((d) => d.file.includes('zz-e2e-temp')).map((d) => d.message).join('\n')
      expect(msgs).toContain('missing Release')
      expect(msgs).toContain('missing Date')
      expect(msgs).toContain('no row of reference/no-such-sheet.md')
      writeFileSync(tempObservation, valid)
      await expect.poll(async () => (await request.get('/api/course')).status(), { timeout: 15_000 }).toBe(200)
    } finally {
      cleanup()
    }
    await expect.poll(async () => (await request.get('/api/course')).status(), { timeout: 15_000 }).toBe(200)
  })

  test('an OBSERVED row that no observation file supports is flagged', async ({ request }) => {
    const sheet = join(contentDir, 'reference', 'observations', 'zz-e2e-temp.md')
    try {
      // A claims table with an OBSERVED row and no observation block at all.
      writeFileSync(sheet, `# zz temp\n\n| claim | source URL | doc version | status |\n|---|---|---|---|\n| z | user observation 2026-10-20 | 7.0 (lab) | OBSERVED |\n`)
      const res = await request.get('/api/course')
      expect(res.status()).toBe(503)
      const msgs = ((await res.json()) as { details: { message: string }[] }).details.map((d) => d.message).join('\n')
      expect(msgs).toContain('no observation block')
      expect(msgs).toContain('OBSERVED row needs an observation block')
    } finally {
      cleanup()
    }
    await expect.poll(async () => (await request.get('/api/course')).status(), { timeout: 15_000 }).toBe(200)
  })
})
