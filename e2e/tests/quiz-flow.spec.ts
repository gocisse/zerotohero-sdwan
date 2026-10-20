import { expect, test } from '@playwright/test'
import {
  allLessons, answerUI, getCourse, getJSON, getQuiz, gotoQuestion, lessonUrl, openLesson, waitForAttempt,
  type AttemptResult, type Course, type QuestionResult, type QuizQuestion, type SectionSummary,
} from './support'

// Soft gating end to end. Tests share state (attempts, review queue), so they run in order.
test.describe.configure({ mode: 'serial' })

let course: Course
let section: SectionSummary // first section with a quiz, preferring non-placeholder
let next: SectionSummary | undefined
let questions: QuizQuestion[]
let key: Map<string, QuestionResult> // learned from the first (failing) attempt

const LOCK_WORDS = /\block(ed)?\b|unlock|must pass|not allowed|cannot continue/i

test.beforeAll(async ({ request }) => {
  course = await getCourse(request)
  const withQuiz = course.sections.filter((s) => s.question_count > 0)
  section = withQuiz.find((s) => !s.placeholder) ?? withQuiz[0]!
  const i = course.sections.findIndex((s) => s.id === section.id)
  next = course.sections[i + 1]
  questions = await getQuiz(request, section.id)
})

test('quiz page: navigator, confirmation dialog, and a failing submission', async ({ page, request }) => {
  const before = (await getJSON<{ attempts: unknown[] }>(request, `/api/sections/${section.id}/quiz/attempts`)).attempts.length
  await page.goto(`/course/${section.id}/quiz`)
  await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
  await expect(page.getByRole('button', { name: `Question ${questions.length}, not answered` })).toBeVisible()

  await page.getByRole('button', { name: 'Submit quiz' }).click()
  await expect(page.getByRole('alertdialog')).toContainText(`${questions.length} unanswered question`)
  await page.getByRole('button', { name: 'Keep working' }).click()
  await expect(page.getByRole('alertdialog')).toHaveCount(0)

  const attempt = waitForAttempt(page)
  await page.getByRole('button', { name: 'Submit quiz' }).click()
  await page.getByRole('button', { name: 'Submit anyway' }).click()
  const result = (await (await attempt).json()) as AttemptResult
  key = new Map(result.results.map((r) => [r.question_id, r]))

  await expect(page.getByTestId('score')).toHaveText(`${result.score} / ${result.total}`)
  expect(result.score).toBe(0)
  expect(result.passed).toBe(false)
  await expect(page.locator('[data-result]')).toHaveCount(questions.length)

  // Explanations, correct answers and per-option explanations are shown.
  const text = await page.locator('main').innerText()
  expect(text).toContain('Topics worth another look')
  expect(text).not.toMatch(LOCK_WORDS)
  const withOptionExplanations = result.results.filter((r) => Object.keys(r.option_explanations).length > 0)
  for (const r of withOptionExplanations) for (const why of Object.values(r.option_explanations)) expect(text).toContain(why)
  await expect(page.getByText('Correct answer:').first()).toBeVisible()

  // Weak-topic panel lists every missed topic with lesson links.
  const missedTopics = [...new Set(result.results.map((r) => r.topic))]
  const panel = page.locator('section[aria-labelledby=revisit]')
  for (const t of missedTopics) await expect(panel).toContainText(t)

  // Continue is always enabled and goes forward.
  const cont = page.getByRole('link', { name: /Continue to/ })
  await expect(cont).toBeEnabled()
  const first = next?.lessons[0]
  await expect(cont).toHaveAttribute('href', first ? `/course/${next!.id}/${first.id}` : '/progress')

  const after = (await getJSON<{ attempts: unknown[] }>(request, `/api/sections/${section.id}/quiz/attempts`)).attempts.length
  expect(after).toBe(before + 1)
})

test('failing changes nothing else: every lesson and section stays reachable', async ({ page, request }) => {
  const progress = await getJSON<{ sections: { section_id: string; quiz_attempts: number }[] }>(request, '/api/progress')
  expect(progress.sections.find((s) => s.section_id === section.id)!.quiz_attempts).toBeGreaterThanOrEqual(1)
  // The next section's lessons open normally after the failed attempt.
  const target = next ?? course.sections[0]!
  for (const l of target.lessons) {
    await openLesson(page, { sectionId: target.id, id: l.id, title: l.title, placeholder: target.placeholder })
    await expect(page.getByRole('alert')).toHaveCount(0)
  }
  await page.goto(`/course/${target.id}`)
  await expect(page.locator('h1')).toBeVisible()
  const res = await request.put(`/api/lessons/${target.lessons[0]!.id}/progress`, { data: { status: 'visited' } })
  expect(res.ok()).toBeTruthy()
})

test('dashboard shows the failed attempt and the weak topics the API reports', async ({ page, request }) => {
  const prog = await getJSON<{ sections: { section_id: string; quiz_attempts: number; best_score_pct: number | null; latest_score_pct: number | null }[] }>(request, '/api/progress')
  const sp = prog.sections.find((s) => s.section_id === section.id)!
  const weak = (await getJSON<{ topics: { topic: string; weak: boolean }[] }>(request, '/api/progress/topics')).topics.filter((t) => t.weak).map((t) => t.topic).sort()

  await page.goto('/progress')
  await expect(page.getByTestId('overall-pct')).toBeVisible()
  const row = page.locator(`[data-section="${section.id}"]`)
  await expect(row.locator('[data-col=attempts]')).toHaveText(String(sp.quiz_attempts))
  await expect(row.locator('[data-col=best]')).toHaveText(`${Math.round(sp.best_score_pct!)}%`)
  await expect(row.locator('[data-col=latest]')).toHaveText(`${Math.round(sp.latest_score_pct!)}%`)
  if (weak.length > 0) {
    await expect(page.locator('[data-weak-topic]').first()).toBeVisible()
    const shown = (await page.locator('[data-weak-topic]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.weakTopic!))).sort()
    expect(shown).toEqual(weak)
    for (const link of await page.locator('[data-weak-topic] a').all()) await expect(link).toHaveAttribute('href', /^\/course\//)
  }
})

test('review: missed answers are due immediately, answering them updates the queue, and no attempt is created', async ({ page, request }) => {
  const queueOf = () => getJSON<{ due_count: number; questions: { id: string }[] }>(request, '/api/review/queue')
  const first = await queueOf()
  expect(first.due_count).toBeGreaterThanOrEqual(Math.min(questions.length, 1))
  expect(first.questions.length).toBeLessThanOrEqual(15)
  // The queue is shared state, most overdue first: on a database that already has answers it can start with questions of
  // quizzes this file never failed. Learn those answers with blank review answers: a blank answer is wrong, so the
  // question stays due (and moves to the back of the queue), and review answers never create a quiz attempt.
  const learned = new Map<string, string | string[]>([...key].map(([id, r]) => [id, r.correct_answer]))
  let queue = first
  for (let guard = 0; !learned.has(queue.questions[0]!.id); guard++) {
    expect(guard, 'every due question can be learned').toBeLessThan(first.due_count + 1)
    const learn = await request.post('/api/review/answers', { data: { answers: [{ question_id: queue.questions[0]!.id, answer: null }] } })
    expect(learn.ok()).toBeTruthy()
    const r = ((await learn.json()) as { results: QuestionResult[] }).results[0]!
    learned.set(r.question_id, r.correct_answer)
    queue = await queueOf()
  }
  const qid = queue.questions[0]!.id
  const correct = learned.get(qid)!
  const attemptsBefore = (await getJSON<{ attempts: unknown[] }>(request, `/api/sections/${section.id}/quiz/attempts`)).attempts.length

  await page.goto('/review')
  await expect(page.getByText(/Question 1 of /)).toBeVisible()
  const q = (await getQuizFor(request, qid)) ?? questions.find((x) => x.id === qid)!
  await answerUI(page, q, correct)
  await page.getByRole('button', { name: 'Check answer' }).click()
  await expect(page.locator('[data-result=correct]')).toBeVisible()

  await expect.poll(async () => (await getJSON<{ due_count: number }>(request, '/api/review/queue')).due_count).toBe(queue.due_count - 1)
  const attemptsAfter = (await getJSON<{ attempts: unknown[] }>(request, `/api/sections/${section.id}/quiz/attempts`)).attempts.length
  expect(attemptsAfter).toBe(attemptsBefore)
})

async function getQuizFor(request: Parameters<typeof getQuiz>[0], qid: string): Promise<QuizQuestion | undefined> {
  const c = await getCourse(request)
  for (const s of c.sections) {
    if (s.question_count === 0) continue
    const found = (await getQuiz(request, s.id)).find((x) => x.id === qid)
    if (found) return found
  }
  return undefined
}

test('retry with the learned answers passes, and "Continue" is still enabled', async ({ page }) => {
  await page.goto(`/course/${section.id}/quiz`)
  await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
  for (let i = 0; i < questions.length; i++) {
    const q = questions[i]!
    await gotoQuestion(page, i + 1)
    await answerUI(page, q, key.get(q.id)!.correct_answer)
  }
  await expect(page.getByRole('button', { name: new RegExp(`^Question ${questions.length}, answered`) })).toBeVisible()
  const attempt = waitForAttempt(page)
  await page.getByRole('button', { name: 'Submit quiz' }).click()
  const result = (await (await attempt).json()) as AttemptResult
  expect(result.score).toBe(result.total)
  expect(result.passed).toBe(true)
  await expect(page.getByTestId('score')).toHaveText(`${result.total} / ${result.total}`)
  await expect(page.locator('main')).toContainText('Nice work')
  await expect(page.getByRole('link', { name: /Continue to/ })).toBeEnabled()

  // Retry starts a fresh run with nothing answered.
  await page.getByRole('button', { name: 'Retry quiz' }).click()
  await expect(page.getByRole('button', { name: 'Question 1, not answered' })).toBeVisible()
})

test('ordering questions work from the keyboard', async ({ page, request }) => {
  for (const s of course.sections.filter((x) => x.question_count > 0)) {
    const qs = await getQuiz(request, s.id)
    const idx = qs.findIndex((q) => q.type === 'ordering')
    if (idx < 0) continue
    await page.goto(`/course/${s.id}/quiz`)
    await expect(page.getByText(/Question 1 of /)).toBeVisible()
    await gotoQuestion(page, idx + 1)
    const q = qs[idx]!
    const ids = () => page.locator(`button[id^="ord-${q.id}-"][id$="-up"]`).evaluateAll((els, p) => els.map((e) => e.id.slice((p as string).length, -3)), `ord-${q.id}-`)
    const before = await ids()
    const moveDown = page.locator(`#ord-${q.id}-${before[0]}-down`)
    await moveDown.focus()
    await page.keyboard.press('Enter')
    const after = await ids()
    expect(after[1]).toBe(before[0])
    expect(after[0]).toBe(before[1])
    await expect(page.getByRole('button', { name: 'Keep this order' }).or(page.getByRole('button', { name: /Order confirmed/ }))).toBeVisible()
    return
  }
  test.skip(true, 'no ordering question in the course')
})

test('skipping from a section card records the skip and leads forward', async ({ page, request }) => {
  const candidates = course.sections.filter((s) => s.question_count > 0)
  const target = candidates[candidates.length - 1]!
  const last = target.lessons[target.lessons.length - 1]!
  await page.goto(lessonUrl({ sectionId: target.id, id: last.id }))
  await expect(page.getByText('Check your understanding')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Take the quiz' })).toBeVisible()
  const i = course.sections.findIndex((s) => s.id === target.id)
  const after = course.sections[i + 1]
  await page.getByRole('button', { name: 'Skip for now' }).click()
  if (after && after.lessons[0]) await page.waitForURL(`**/course/${after.id}/${after.lessons[0].id}`)
  else await page.waitForURL('**/progress')
  const prog = await getJSON<{ sections: { section_id: string; quiz_skipped: boolean }[] }>(request, '/api/progress')
  expect(prog.sections.find((s) => s.section_id === target.id)!.quiz_skipped).toBe(true)
  // Skipping again is harmless (idempotent).
  const again = await request.post(`/api/sections/${target.id}/quiz/skip`)
  expect(again.ok()).toBeTruthy()
})

test('progress persists across a reload', async ({ page }) => {
  await page.goto('/progress')
  await expect(page.getByTestId('overall-pct')).toBeVisible()
  await page.waitForTimeout(600)
  const before = await page.locator('main').innerText()
  await page.reload()
  await expect(page.getByTestId('overall-pct')).toBeVisible()
  await page.waitForTimeout(600)
  expect(await page.locator('main').innerText()).toBe(before)
  expect(allLessons(course).length).toBeGreaterThan(0)
})
