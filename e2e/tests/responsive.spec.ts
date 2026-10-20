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
    await expect(page.locator('[data-dashboard]')).toBeVisible()
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
      for (const l of s.lessons) await expect(nav.locator(`a[href$="/${l.id}"]`), l.title).toBeVisible()
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

test.describe('narrow screen callouts', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, reducedMotion: 'reduce' })

  // The pan code keeps every callout bubble fully inside the diagram box whenever the bubbles fit together (panTarget.ts).
  test('no callout bubble is clipped on any step of any diagram with callouts', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const owner = new Map<string, (typeof used)[number]>()
    for (const l of used) for (const d of l.diagrams) if (!owner.has(d)) owner.set(d, l)
    test.setTimeout(60_000 + 8_000 * owner.size)
    let checked = 0
    for (const [id, lesson] of owner) {
      const d = await getJSON<{ steps: { callouts?: unknown[] }[] }>(request, `/api/diagrams/${id}`)
      if (!d.steps.some((s) => (s.callouts ?? []).length > 0)) continue
      await page.goto(lessonUrl(lesson))
      await page.locator('[data-lesson-end]').waitFor()
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      for (let s = 1; s <= d.steps.length; s++) {
        await page.waitForTimeout(120)
        const clipped = await fig.evaluate((el) => {
          const box = el.querySelector('.overflow-x-auto')
          if (!box) return 0
          const br = box.getBoundingClientRect()
          return [...el.querySelectorAll('[data-callout-bubble]')].filter((b) => {
            const r = b.getBoundingClientRect()
            return r.left < br.left - 1 || r.right > br.right + 1
          }).length
        })
        expect(clipped, `${id} step ${s}: clipped callout bubbles`).toBe(0)
        if (s < d.steps.length) await fig.getByRole('button', { name: 'Next →' }).click()
      }
      checked++
    }
    expect(checked, 'diagrams with callouts checked').toBeGreaterThan(10)
  })
})

// A link label slides along its link to stay off every node's label (placeLinkLabel in calloutPlacement.ts). The detector below is
// checked against a deliberately moved label first, so "no overlap" cannot come from a detector that sees nothing.
const OVERLAPS = `(el) => {
  const vis = (e) => { for (let x = e; x && x !== el; x = x.parentElement) if (x.classList && x.classList.contains('opacity-0')) return false; return true }
  const texts = [...el.querySelectorAll('svg text')].filter(vis)
  const nodeTexts = texts.filter((t) => t.closest('[data-node]'))
  const linkTexts = texts.filter((t) => !t.closest('[data-node]') && !t.closest('[data-packet]') && !t.closest('[data-group]'))
  const out = []
  for (const l of linkTexts) {
    const a = l.getBoundingClientRect()
    for (const n of nodeTexts) {
      const c = n.getBoundingClientRect()
      if (Math.min(a.right, c.right) - Math.max(a.left, c.left) > 2 && Math.min(a.bottom, c.bottom) - Math.max(a.top, c.top) > 2) out.push(l.textContent + ' x ' + n.textContent)
    }
  }
  return out
}`

test.describe('link labels', () => {
  test.use({ reducedMotion: 'reduce' })

  test('the overlap detector sees a label placed on top of a node label', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const lesson = used.find((l) => l.diagrams.includes('svc-workflow'))!
    await page.goto(lessonUrl(lesson))
    await page.locator('[data-lesson-end]').waitFor()
    const fig = page.locator('[data-diagram="svc-workflow"]')
    await fig.scrollIntoViewIfNeeded()
    while (await fig.getByRole('button', { name: 'Next →' }).isEnabled()) await fig.getByRole('button', { name: 'Next →' }).click()
    expect(await fig.evaluate((el, src) => (0, eval)(`(${src})`)(el) as string[], OVERLAPS)).toEqual([])
    // Move the first link label onto the first node label: the detector must now report an overlap.
    const found = await fig.evaluate((el) => {
      const vis = (e: Element) => {
        for (let x: Element | null = e; x && x !== el; x = x.parentElement) if (x.classList.contains('opacity-0')) return false
        return true
      }
      const texts = [...el.querySelectorAll('svg text')].filter(vis) as SVGTextElement[]
      const node = texts.find((t) => t.closest('[data-node]'))
      const label = texts.find((t) => !t.closest('[data-node]') && !t.closest('[data-packet]') && !t.closest('[data-group]'))
      if (!node || !label) return false
      const scale = label.getScreenCTM()?.a ?? 1
      const nb = node.getBoundingClientRect()
      const lb = label.getBoundingClientRect()
      label.setAttribute('transform', `translate(${(nb.left - lb.left) / scale} ${(nb.top - lb.top) / scale})`)
      return true
    })
    expect(found).toBe(true)
    expect((await fig.evaluate((el, src) => (0, eval)(`(${src})`)(el) as string[], OVERLAPS)) as string[]).not.toEqual([])
  })

  test('no link label overlaps a node label on any step of any diagram', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const owner = new Map<string, (typeof used)[number]>()
    for (const l of used) for (const d of l.diagrams) if (!owner.has(d)) owner.set(d, l)
    test.setTimeout(60_000 + 8_000 * owner.size)
    for (const [id, lesson] of owner) {
      await page.goto(lessonUrl(lesson))
      await page.locator('[data-lesson-end]').waitFor()
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      const d = await getJSON<{ steps: unknown[] }>(request, `/api/diagrams/${id}`)
      for (let s = 1; s <= Math.max(1, d.steps.length); s++) {
        await page.waitForTimeout(60)
        expect(await fig.evaluate((el, src) => (0, eval)(`(${src})`)(el) as string[], OVERLAPS), `${id} step ${s}: link label over node text`).toEqual([])
        const next = fig.getByRole('button', { name: 'Next →' })
        if (s < d.steps.length) await next.click()
      }
    }
  })
})
