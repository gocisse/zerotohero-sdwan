import { expect, test } from '@playwright/test'
import { answerUI, getCourse, getQuiz, gotoQuestion, waitForAttempt, type AttemptResult, type QuizQuestion } from './support'

// Regression for a flaky-suite root cause: the server shuffles ordering options on every request, so about one
// request in n! delivers them already in the correct order. The page counts an ordering answer as given only when the
// reader moves something or presses "Keep this order", so a script that only moves items never answers that question,
// "Submit quiz" opens the confirmation dialog and no attempt is posted. This test forces that shuffle every time.
test('an ordering question that arrives already in the correct order still needs "Keep this order" before it counts', async ({ page, request }) => {
  const course = await getCourse(request)
  let target: { id: string; questions: QuizQuestion[] } | undefined
  for (const s of course.sections.filter((x) => x.question_count > 0)) {
    const questions = await getQuiz(request, s.id)
    if (questions.some((q) => q.type === 'ordering')) {
      target = { id: s.id, questions }
      break
    }
  }
  test.skip(!target, 'no ordering question in the course')
  const { id, questions } = target!

  // Learn every correct answer from a blank attempt (an unanswered quiz is wrong, nothing is locked).
  const blank = await request.post(`/api/sections/${id}/quiz/attempts`, {
    data: { answers: questions.map((q) => ({ question_id: q.id, answer: null })) },
  })
  expect(blank.ok()).toBeTruthy()
  const key = new Map(((await blank.json()) as AttemptResult).results.map((r) => [r.question_id, r.correct_answer]))

  // Serve every ordering question already in its correct order.
  await page.route(`**/api/sections/${id}/quiz`, async (route) => {
    const res = await route.fetch()
    const body = (await res.json()) as { questions: QuizQuestion[] }
    for (const q of body.questions) {
      if (q.type !== 'ordering') continue
      const order = key.get(q.id) as string[]
      q.options.sort((a, b) => order.indexOf(a.id) - order.indexOf(b.id))
    }
    await route.fulfill({ response: res, json: body })
  })

  await page.goto(`/course/${id}/quiz`)
  await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
  const ordIndex = questions.findIndex((q) => q.type === 'ordering')
  const ord = questions[ordIndex]!
  await gotoQuestion(page, ordIndex + 1)
  // Already correct, so a script has nothing to move; the question is still "not answered" until it is confirmed.
  await expect(page.getByRole('button', { name: new RegExp(`^Question ${ordIndex + 1}, not answered`) })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Keep this order' })).toBeVisible()

  // answerUI confirms such an order, so the whole quiz can be answered and submitted without the "submit anyway" dialog.
  for (const [i, q] of questions.entries()) {
    await gotoQuestion(page, i + 1)
    await answerUI(page, q, key.get(q.id)!)
  }
  const attempt = waitForAttempt(page, 15_000)
  await page.getByRole('button', { name: 'Submit quiz' }).click()
  const result = (await (await attempt).json()) as AttemptResult
  expect(result.score, `${ord.id} and every other answer was given`).toBe(result.total)
})
