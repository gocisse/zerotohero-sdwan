import { expect, test } from '@playwright/test'
import { answerUI, getCourse, getQuiz, gotoQuestion, waitForAttempt, type AttemptResult } from './support'

// Every section's quiz can be failed (nothing locks) and then retried and passed.
// Sections are discovered through the API, so new sections are covered automatically.
test('every section quiz can be failed, retried and passed, and nothing is ever blocked', async ({ page, request }) => {
  const course = await getCourse(request)
  const sections = course.sections.filter((s) => s.question_count > 0)
  expect(sections.length).toBeGreaterThan(0)
  // The work grows with every section, so the time budget does too (fail, retry and pass each quiz).
  test.setTimeout(30_000 + 40_000 * sections.length)

  for (const [n, s] of sections.entries()) {
    const questions = await getQuiz(request, s.id)
    const nextSection = course.sections[course.sections.findIndex((x) => x.id === s.id) + 1]

    // Fail on purpose: submit with nothing answered.
    await page.goto(`/course/${s.id}/quiz`)
    await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
    const failing = waitForAttempt(page)
    await page.getByRole('button', { name: 'Submit quiz' }).click()
    await page.getByRole('button', { name: 'Submit anyway' }).click()
    const failed = (await (await failing).json()) as AttemptResult
    expect(failed.score, `${s.id}: unanswered counts as wrong`).toBe(0)
    await expect(page.getByTestId('score')).toHaveText(`0 / ${questions.length}`)
    await expect(page.locator('[data-result]')).toHaveCount(questions.length)
    await expect(page.locator('main')).not.toContainText(/\block(ed)?\b|unlock|must pass/i)

    // Whatever the score, Continue is enabled and the next section is reachable.
    const cont = page.getByRole('link', { name: /Continue to/ })
    await expect(cont, `${s.id}: Continue enabled after failing`).toBeEnabled()
    const first = nextSection?.lessons[0]
    await expect(cont).toHaveAttribute('href', first ? `/course/${nextSection!.id}/${first.id}` : '/progress')
    await cont.click()
    await expect(page.locator('h1').first()).toBeVisible()

    // Retry with the answers revealed by the failed attempt.
    const key = new Map(failed.results.map((r) => [r.question_id, r.correct_answer]))
    await page.goto(`/course/${s.id}/quiz`)
    await expect(page.getByText(`Question 1 of ${questions.length}`)).toBeVisible()
    for (const [i, q] of questions.entries()) {
      await gotoQuestion(page, i + 1)
      await answerUI(page, q, key.get(q.id)!)
    }
    const passing = waitForAttempt(page)
    await page.getByRole('button', { name: 'Submit quiz' }).click()
    const passed = (await (await passing).json()) as AttemptResult
    expect(passed.score, `${s.id}: retry passes`).toBe(passed.total)
    await expect(page.getByTestId('score')).toHaveText(`${passed.total} / ${passed.total}`)
    expect(n).toBeGreaterThanOrEqual(0)
  }
})
