import { expect, test, type APIRequestContext } from '@playwright/test'
import { getJSON } from './support'

// Lab tests (step 14). Labs, tasks, choices and fixes are discovered through the API. The right answers are found by
// asking the server's own check endpoint, so no answer is hard-coded here.
interface Task { id: string; type: string; choices: { id: string; text: string }[] }
interface Lab {
  id: string
  topology: string
  hint_count: number
  task_list: Task[]
  fixes: { id: string }[]
  goals: { id: string; section_id: string }[]
}
interface Check { correct: boolean }

const checkApi = async (request: APIRequestContext, lab: string, task: string, answer: string, fixes: string[]) => {
  const res = await request.post(`/api/labs/${lab}/tasks/${task}/check`, { data: { answer, fixes } })
  expect(res.ok()).toBeTruthy()
  return ((await res.json()) as Check).correct
}

/** The right choice of every choice task and a fix that passes every fix task, found through the check endpoint. */
async function solve(request: APIRequestContext, lab: Lab) {
  const right: Record<string, string> = {}
  const wrong: Record<string, string> = {}
  // Several fix tasks may each need their own fix (the capstone labs): the first allowed fix that passes each.
  const fixFor: Record<string, string> = {}
  for (const t of lab.task_list) {
    if (t.type === 'fix') {
      for (const f of lab.fixes) if (!fixFor[t.id] && (await checkApi(request, lab.id, t.id, '', [f.id]))) fixFor[t.id] = f.id
      continue
    }
    for (const c of t.choices) {
      if (await checkApi(request, lab.id, t.id, c.id, [])) right[t.id] = c.id
      else if (!wrong[t.id]) wrong[t.id] = c.id
    }
  }
  return { right, wrong, fixFor }
}

const allLabs = async (request: APIRequestContext): Promise<Lab[]> => {
  const ids = (await getJSON<{ labs: { id: string }[] }>(request, '/api/labs')).labs.map((l) => l.id)
  const out: Lab[] = []
  for (const id of ids) out.push(await getJSON<Lab>(request, `/api/labs/${id}`))
  return out
}

test.describe.configure({ mode: 'serial' })

test.describe('labs', () => {
  test('the lab API never leaks an answer before a task is checked', async ({ request }) => {
    for (const lab of await allLabs(request)) {
      const text = JSON.stringify(lab)
      for (const key of ['"probe"', '"explanation"', '"equals"', '"patches"', '"value"', '"correct"']) expect(text, `${lab.id} leaks ${key}`).not.toContain(key)
    }
    const list = JSON.stringify(await getJSON(request, '/api/labs'))
    for (const key of ['"explanation"', '"equals"', '"patches"', '"value"']) expect(list).not.toContain(key)
  })

  test('every lab can be started, hinted, checked wrong then right, fixed and completed', async ({ page, request }) => {
    const labs = await allLabs(request)
    expect(labs.length).toBeGreaterThan(0)
    test.setTimeout(60_000 + 60_000 * labs.length)
    for (const lab of labs) {
      const { right, wrong, fixFor } = await solve(request, lab)
      await page.goto(`/sim/labs/${lab.id}`)
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      await expect(page.locator('[data-lab-symptom]')).toBeVisible()
      await expect(page.locator('[data-lab-screen]').first()).toBeVisible()
      // Hints come one at a time.
      await page.locator('[data-lab-next-hint]').click()
      await expect(page.locator('[data-lab-hint]')).toHaveCount(1)
      for (const t of lab.task_list) {
        const card = page.locator(`[data-lab-task=${t.id}]`)
        if (t.type === 'fix') {
          // Applying a fix clears earlier fix results, so every fix is applied first and every fix task is checked afterwards.
          const fix = fixFor[t.id]!
          // Two fix tasks can need the same fix; it is applied once.
          if ((await card.locator(`[data-lab-fix=${fix}]`).getAttribute('data-applied')) !== 'true') {
            await card.locator(`[data-lab-apply=${fix}]`).click()
          }
          await expect(card.locator(`[data-lab-fix=${fix}]`)).toHaveAttribute('data-applied', 'true')
          continue
        }
        if (wrong[t.id]) {
          await card.locator(`[data-lab-choice=${wrong[t.id]}]`).check()
          await card.locator(`[data-lab-check=${t.id}]`).click()
          await expect(card.locator('[data-lab-result]'), `${lab.id}/${t.id} wrong`).toHaveAttribute('data-lab-result', 'incorrect')
          // A wrong check only gives feedback: the button is still usable.
          await expect(card.locator(`[data-lab-check=${t.id}]`)).toBeEnabled()
        }
        await card.locator(`[data-lab-choice=${right[t.id]}]`).check()
        await card.locator(`[data-lab-check=${t.id}]`).click()
        await expect(card.locator('[data-lab-result]'), `${lab.id}/${t.id} right`).toHaveAttribute('data-lab-result', 'correct')
      }
      for (const t of lab.task_list.filter((x) => x.type === 'fix')) {
        const card = page.locator(`[data-lab-task=${t.id}]`)
        await card.locator(`[data-lab-check=${t.id}]`).click()
        await expect(card.locator('[data-lab-result]'), `${lab.id}/${t.id} fixed`).toHaveAttribute('data-lab-result', 'correct')
      }
      await expect(page.locator('[data-lab-complete]'), lab.id).toBeVisible()
      await expect.poll(async () => (await getJSON<Lab & { progress: { status: string } }>(request, `/api/labs/${lab.id}`)).progress.status).toBe('completed')
    }
    // The dashboard counts them.
    await page.goto('/progress')
    await expect(page.locator('[data-labs-block]')).toContainText(`${labs.length} of ${labs.length} completed`)
  })

  test('a lab can be skipped, nothing is locked, and the lesson links reach a lab', async ({ page, request }) => {
    const labs = await allLabs(request)
    const lab = labs[0]!
    await page.goto(`/sim/labs/${lab.id}`)
    await page.locator('[data-lab-skip]').click()
    await expect(page).toHaveURL(/\/sim\/labs$/)
    // Completed stays completed (never downgraded by a skip): the list still shows the card.
    await expect(page.locator(`[data-lab-card=${lab.id}]`)).toBeVisible()
    // The next section is still reachable.
    const course = await getJSON<{ sections: { id: string; lessons: { id: string }[] }[] }>(request, '/api/course')
    const s = course.sections[1]!
    await page.goto(`/course/${s.id}/${s.lessons[0]!.id}`)
    await expect(page.locator('h1').first()).toBeVisible()
    // Every lab names lessons that exist, and at least one lesson of the course links to a lab.
    for (const l of labs) for (const g of l.goals) expect(course.sections.some((c) => c.lessons.some((x) => x.id === g.id)), `${l.id}: ${g.id}`).toBeTruthy()
    let linked = 0
    for (const sec of course.sections)
      for (const les of sec.lessons) {
        const src = (await getJSON<{ source: string }>(request, `/api/lessons/${les.id}`)).source
        if (src.includes('/sim/labs/')) linked++
      }
    expect(linked).toBeGreaterThan(0)
  })

  test("the lab's faulty network is what the simulator screens show", async ({ page, request }) => {
    const lab = (await allLabs(request)).find((l) => l.fixes.length > 0)!
    const faulty = await request.get(`/api/topologies/${lab.topology}?lab=${lab.id}`)
    expect(faulty.ok()).toBeTruthy()
    expect((await faulty.json()).lab.id).toBe(lab.id)
    await page.goto(`/sim/${lab.topology}/trace?lab=${lab.id}`)
    await expect(page.locator('[data-lab-banner]')).toBeVisible()
    await expect(page.locator('[data-trace-result]')).toBeVisible()
  })
})
