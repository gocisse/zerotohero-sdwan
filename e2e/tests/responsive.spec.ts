import { expect, test, type Page } from '@playwright/test'
import { getCourse, getJSON, scanCourse, lessonUrl } from './support'

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)

async function routes(request: Parameters<typeof scanCourse>[0]) {
  const { course, used } = await scanCourse(request)
  const lesson = used.find((l) => l.diagrams.length > 0) ?? used[0]!
  const quizSection = course.sections.find((s) => s.question_count > 0)!
  return ['/', lessonUrl(lesson), `/course/${quizSection.id}`, `/course/${quizSection.id}/quiz`, '/progress', '/glossary', '/review']
}

test.describe('dark mode', () => {
  test.use({ colorScheme: 'dark' })
  test('key pages render without horizontal overflow and with a dark background', async ({ page, request }) => {
    for (const r of await routes(request)) {
      await page.goto(r)
      await page.waitForTimeout(500)
      expect(await overflow(page), `${r}: overflow`).toBeLessThanOrEqual(1)
      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
      const [rr, gg, bb] = (bg.match(/\d+/g) ?? ['255', '255', '255']).map(Number)
      expect((rr! + gg! + bb!) / 3, `${r}: dark background`).toBeLessThan(80)
    }
  })
})

test.describe('narrow screen', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true })

  test('no horizontal page scroll on key pages', async ({ page, request }) => {
    for (const r of await routes(request)) {
      await page.goto(r)
      await page.waitForTimeout(700)
      expect(await overflow(page), `${r}: overflow`).toBeLessThanOrEqual(1)
    }
  })

  test('menu drawer opens, navigates, closes and handles Escape', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('button', { name: 'Open menu' })).toBeVisible()
    await page.getByRole('button', { name: 'Open menu' }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).toHaveCount(0)
    await page.getByRole('button', { name: 'Open menu' }).click()
    await page.getByRole('dialog').getByRole('link', { name: 'Progress', exact: true }).click()
    await expect(page.getByTestId('overall-pct')).toBeVisible()
    await expect(page.getByRole('dialog')).toHaveCount(0)
  })

  test('a wide diagram scrolls inside its own box with a pan hint and follows the steps', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const lesson = used.find((l) => l.diagrams.length > 0)
    test.skip(!lesson, 'no diagram in use')
    await page.goto(lessonUrl(lesson!))
    const fig = page.locator(`[data-diagram="${lesson!.diagrams[0]}"]`)
    await fig.scrollIntoViewIfNeeded()
    const scroller = fig.locator('.overflow-x-auto').first()
    const wide = await scroller.evaluate((el) => el.scrollWidth > el.clientWidth + 4)
    if (wide) {
      await expect(fig.locator('[data-pan-hint]')).toBeVisible()
      const next = fig.getByRole('button', { name: 'Next →' })
      if (await next.count()) {
        await next.click()
        await next.click().catch(() => {})
      }
    }
    expect(await overflow(page)).toBeLessThanOrEqual(1)
  })
})

test.describe('sidebar', () => {
  test('lists every section and lesson, collapses sections, and shows the review badge', async ({ page, request }) => {
    const course = await getCourse(request)
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: 'Course' })
    for (const s of course.sections) {
      await expect(nav.getByRole('button', { name: new RegExp(s.title.slice(0, 12)) })).toBeVisible()
      for (const l of s.lessons) await expect(nav.getByRole('link', { name: l.title })).toBeVisible()
    }
    const s0 = course.sections[0]!
    const toggle = nav.getByRole('button', { name: new RegExp(s0.title.slice(0, 12)) }).first()
    await toggle.click()
    await expect(nav.getByRole('link', { name: s0.lessons[0]!.title })).toHaveCount(0)
    await toggle.click()
    await expect(nav.getByRole('link', { name: s0.lessons[0]!.title })).toBeVisible()
    const due = (await getJSON<{ due_count: number }>(request, '/api/review/queue')).due_count
    if (due > 0) await expect(page.locator('[data-review-badge]')).toHaveText(String(due))
    else await expect(page.locator('[data-review-badge]')).toHaveCount(0)
  })

  test('Home shows backend and database OK and a "Start learning" link to the first lesson', async ({ page, request }) => {
    const course = await getCourse(request)
    const first = course.sections[0]!.lessons[0]!
    await page.goto('/')
    await expect(page.getByText('OK')).toHaveCount(2)
    await page.getByRole('link', { name: /Start learning/ }).click()
    await page.waitForURL(`**/course/${course.sections[0]!.id}/${first.id}`)
  })
})
