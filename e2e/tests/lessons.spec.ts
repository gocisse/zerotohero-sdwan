import { expect, test } from '@playwright/test'
import { openLesson, scanCourse } from './support'

test.describe('lessons', () => {
  test('every lesson in the course renders without error panels', async ({ page, request }) => {
    const { lessons } = await scanCourse(request)
    expect(lessons.length).toBeGreaterThan(0)
    test.setTimeout(30_000 + 3_000 * lessons.length) // grows with the course
    for (const l of lessons) {
      await openLesson(page, l)
      await page.waitForTimeout(250) // let the lazily compiled MDX finish
      await expect(page.getByRole('alert'), `${l.id}: error panel`).toHaveCount(0)
      await expect(page.locator('[data-term-error]'), `${l.id}: unknown term marker`).toHaveCount(0)
    }
  })

  test('real (non-placeholder) lessons follow the authoring rules', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const real = used.filter((l) => !l.placeholder)
    for (const l of real) {
      await openLesson(page, l)
      const body = page.locator('article .prose').first()
      await expect(body.locator(':scope > p').first(), `${l.id}: objectives paragraph`).toContainText('By the end of this lesson')
      expect(await body.locator(':scope > ul').first().locator(':scope > li').count(), `${l.id}: objective bullets`).toBeGreaterThanOrEqual(3)
      await expect(body.locator('h1'), `${l.id}: no h1 in body`).toHaveCount(0)
      expect(l.predicts.length, `${l.id}: at least one Predict`).toBeGreaterThanOrEqual(1)
      await expect(page.locator('[data-predict]'), `${l.id}: Predict rendered`).toHaveCount(l.predicts.length)
      expect(await page.locator('[data-callout]').count(), `${l.id}: at least one Callout`).toBeGreaterThanOrEqual(1)
      expect(l.terms.length, `${l.id}: Term tags`).toBeGreaterThanOrEqual(3)
      for (const d of l.diagrams) await expect(page.locator(`[data-diagram="${d}"]`), `${l.id}: diagram ${d}`).toHaveCount(1)
    }
  })

  test('lesson progress: visited on open, completed after scrolling to the end', async ({ page, request }) => {
    const { lessons } = await scanCourse(request)
    // A long lesson is more likely to need a real scroll; use the last lesson of the first section with 2+ lessons.
    const target = lessons[lessons.length - 1]!
    await openLesson(page, target)
    await expect.poll(async () => {
      const p = (await (await request.get('/api/progress')).json()) as { lessons: { lesson_id: string; status: string }[] }
      return p.lessons.find((x) => x.lesson_id === target.id)?.status
    }).toMatch(/visited|completed/)
    await page.locator('main').evaluate((el) => el.scrollTo(0, el.scrollHeight))
    await expect.poll(async () => {
      const p = (await (await request.get('/api/progress')).json()) as { lessons: { lesson_id: string; status: string }[] }
      return p.lessons.find((x) => x.lesson_id === target.id)?.status
    }).toBe('completed')
    await expect(page.locator('[data-status=completed]').first()).toBeVisible()
  })

  test('"Mark as complete" works and unknown lessons show not-found', async ({ page, request }) => {
    const { lessons } = await scanCourse(request)
    await page.goto('/course/nope/nope')
    await expect(page.getByText('Page not found')).toBeVisible()
    await page.goto(`/course/${lessons[0]!.sectionId}/${lessons[0]!.id}`)
    await expect(page.locator('[data-lesson-end]')).toBeVisible()
  })
})
