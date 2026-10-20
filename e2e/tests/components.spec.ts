import { expect, test, type Page } from '@playwright/test'
import { getJSON, openLesson, scanCourse, type Diagram, type GlossaryTerm } from './support'

const posOf = (page: Page, diagramId: string) =>
  page.evaluate((d) => {
    const r = document.querySelector(`[data-diagram="${d}"] [data-packet]`)?.getBoundingClientRect()
    return r ? [r.x, r.y] : null
  }, diagramId)

async function diagramsInUse(request: Parameters<typeof scanCourse>[0]) {
  const { used } = await scanCourse(request)
  const map = new Map<string, (typeof used)[number]>()
  for (const l of used) for (const d of l.diagrams) if (!map.has(d)) map.set(d, l)
  return map
}

test.describe('diagrams', () => {
  test('every diagram steps through and its packets move', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    expect(map.size).toBeGreaterThan(0)
    test.setTimeout(30_000 + 10_000 * map.size) // grows with the number of diagrams
    for (const [id, lesson] of map) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()

      // Legend shows exactly the link styles the diagram uses.
      const styles = [...new Set(d.links.map((l) => l.style))].sort()
      const legend = await fig.locator('[data-legend-style]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.legendStyle!).sort())
      expect(legend, `${id}: legend`).toEqual(styles)

      if (d.steps.length === 0) {
        await expect(fig.locator('[data-step-indicator]'), `${id}: static has no player`).toHaveCount(0)
        continue
      }
      let moved = 0
      let packetSteps = 0
      for (let s = 1; s <= d.steps.length; s++) {
        await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step ${s} of ${d.steps.length}`)
        const expected = d.steps[s - 1]!.packets.length
        if (expected > 0) {
          packetSteps++
          await expect(fig.locator('[data-packet]'), `${id} step ${s}: packet elements`).toHaveCount(expected)
          await page.waitForTimeout(200)
          const a = await posOf(page, id)
          await page.waitForTimeout(250)
          const b = await posOf(page, id)
          if (a && b && (Math.abs(a[0]! - b[0]!) > 1 || Math.abs(a[1]! - b[1]!) > 1)) moved++
        } else {
          await expect(fig.locator('[data-packet]')).toHaveCount(0)
        }
        if (s < d.steps.length) await fig.getByRole('button', { name: 'Next →' }).click()
      }
      if (packetSteps > 0) expect(moved, `${id}: packets animate`).toBeGreaterThan(0)
      await expect(fig.getByRole('button', { name: 'Next →' })).toBeDisabled()
      await fig.getByRole('button', { name: '← Previous' }).click()
      await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step ${d.steps.length - 1} of ${d.steps.length}`)
    }
  })

  test('arrow keys step a diagram, also with a node focused', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    for (const [id, lesson] of map) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      if (d.steps.length < 2) continue
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      await fig.getByRole('group', { name: /Step player/ }).focus()
      await page.keyboard.press('ArrowRight')
      await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step 2 of ${d.steps.length}`)
      await page.keyboard.press('ArrowLeft')
      await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step 1 of ${d.steps.length}`)
      await fig.locator('[data-node]').first().focus()
      await page.keyboard.press('ArrowRight')
      await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step 2 of ${d.steps.length}`)
      return
    }
    test.skip(true, 'no stepped diagram in use')
  })

  test('node tooltip appears on hover and on keyboard focus', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    const [id, lesson] = [...map][0]!
    const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
    await openLesson(page, lesson)
    const fig = page.locator(`[data-diagram="${id}"]`)
    await fig.scrollIntoViewIfNeeded()
    // Use the last step so progressively revealed nodes are visible.
    for (let i = 1; i < d.steps.length; i++) await fig.getByRole('button', { name: 'Next →' }).click()
    const first = d.nodes[0]!
    const node = fig.locator(`[data-node="${first.id}"]`)
    await node.hover()
    await expect(fig.getByRole('tooltip')).toContainText(first.label)
    await page.mouse.move(2, 2)
    await expect(fig.getByRole('tooltip')).toHaveCount(0)
    await node.focus()
    await expect(fig.getByRole('tooltip')).toContainText(first.label)
    await expect(node).toHaveAttribute('aria-label', new RegExp(first.label))
  })

  test('step player can play and pause', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    for (const [id, lesson] of map) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      if (d.steps.length < 2) continue
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      await fig.getByRole('button', { name: '▶ Play' }).click()
      await expect(fig.getByRole('button', { name: '⏸ Pause' })).toBeVisible()
      await expect(fig.locator('[data-step-indicator]')).toHaveText(`Step 2 of ${d.steps.length}`, { timeout: 15_000 })
      await fig.getByRole('button', { name: '⏸ Pause' }).click()
      const frozen = await fig.locator('[data-step-indicator]').innerText()
      await page.waitForTimeout(4800)
      expect(await fig.locator('[data-step-indicator]').innerText()).toBe(frozen)
      return
    }
    test.skip(true, 'no stepped diagram in use')
  })
})

test.describe('reduced motion', () => {
  test.use({ reducedMotion: 'reduce' })
  test('packets are shown at their destination with no movement', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    for (const [id, lesson] of map) {
      const d = await getJSON<Diagram>(request, `/api/diagrams/${id}`)
      const idx = d.steps.findIndex((s) => s.packets.length > 0)
      if (idx < 0) continue
      await openLesson(page, lesson)
      const fig = page.locator(`[data-diagram="${id}"]`)
      await fig.scrollIntoViewIfNeeded()
      for (let i = 0; i < idx; i++) await fig.getByRole('button', { name: 'Next →' }).click()
      await expect(fig.locator('[data-packet]').first()).toHaveAttribute('data-state', 'arrived')
      const a = await posOf(page, id)
      await page.waitForTimeout(400)
      expect(await posOf(page, id)).toEqual(a)
      return
    }
    test.skip(true, 'no diagram with packets in use')
  })
})

test.describe('Predict', () => {
  test('right answer, wrong answer and "I\'m not sure" each reveal correctly and reset', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const withPredict = used.filter((l) => l.predicts.length > 0)
    expect(withPredict.length).toBeGreaterThan(0)
    for (const l of withPredict) {
      await openLesson(page, l)
      for (let i = 0; i < l.predicts.length; i++) {
        const info = l.predicts[i]!
        const p = page.locator('[data-predict]').nth(i)
        await p.scrollIntoViewIfNeeded()
        await expect(p.locator('[data-predict-result]')).not.toContainText('The answer')
        await p.locator(`[data-choice="${info.answer}"]`).click()
        await expect(p.locator('[data-predict-result]')).toContainText('Nice prediction')
        await expect(p.locator(`[data-choice="${info.answer}"][data-state=correct]`)).toHaveCount(1)
        await p.getByRole('button', { name: /Try again/ }).click()
        const wrong = [...Array(info.choices).keys()].find((c) => c !== info.answer)!
        await p.locator(`[data-choice="${wrong}"]`).click()
        await expect(p.locator('[data-predict-result]')).toContainText('Not quite')
        await expect(p.locator(`[data-choice="${wrong}"][data-state=wrong]`)).toHaveCount(1)
        await p.getByRole('button', { name: /Try again/ }).click()
        await p.locator('[data-unsure]').click()
        await expect(p.locator('[data-predict-result]')).toContainText('No problem')
        await expect(p.locator('[data-predict-result]')).not.toContainText('Not quite')
        await p.getByRole('button', { name: /Try again/ }).click()
      }
    }
  })

  test('is keyboard operable and not persisted', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const l = used.find((x) => x.predicts.length > 0)!
    await openLesson(page, l)
    const p = page.locator('[data-predict]').first()
    await p.scrollIntoViewIfNeeded()
    await p.locator('[data-choice="0"]').focus()
    await page.keyboard.press('Enter')
    await expect(p.locator('[data-predict-result]')).toContainText('The answer')
    await page.reload()
    await expect(page.locator('[data-predict]').first().locator('[data-predict-result]')).not.toContainText('The answer')
  })
})

test.describe('Term popover', () => {
  test('opens on hover, on keyboard focus and closes on Escape', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const l = used.find((x) => x.terms.length > 0)!
    await openLesson(page, l)
    const t = page.locator(`[data-term="${l.terms[0]}"]`).first()
    await t.scrollIntoViewIfNeeded()
    await expect(page.locator('[data-term-popover]')).toHaveCount(0)
    await t.hover()
    const pop = page.locator('[data-term-popover]')
    await expect(pop).toBeVisible()
    await expect(pop.getByRole('link', { name: /Full glossary entry/ })).toHaveCount(1)
    await page.mouse.move(2, 2)
    await expect(pop).toHaveCount(0)
    await t.focus()
    await expect(pop).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(pop).toHaveCount(0)
  })
})

test.describe('Term popover (touch)', () => {
  test.use({ hasTouch: true, viewport: { width: 390, height: 844 } })
  test('opens on tap and closes on a tap elsewhere', async ({ page, request }) => {
    const { used } = await scanCourse(request)
    const l = used.find((x) => x.terms.length > 0)!
    await openLesson(page, l)
    const t = page.locator(`[data-term="${l.terms[0]}"]`).first()
    await t.scrollIntoViewIfNeeded()
    await t.tap()
    await expect(page.locator('[data-term-popover]')).toBeVisible()
    await page.locator('h1').tap()
    await expect(page.locator('[data-term-popover]')).toHaveCount(0)
  })
})

test.describe('glossary page', () => {
  test('lists every term alphabetically and search finds terms by name and alias', async ({ page, request }) => {
    const { terms } = await getJSON<{ terms: GlossaryTerm[] }>(request, '/api/glossary')
    expect(terms.length).toBeGreaterThan(0)
    await page.goto('/')
    await page.getByRole('link', { name: 'Glossary', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Glossary' })).toBeVisible()
    await page.getByRole('button', { name: 'A-Z' }).click()
    const sorted = [...terms].sort((a, b) => a.term.localeCompare(b.term, undefined, { sensitivity: 'base' })).map((t) => t.id)
    const ids = await page.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
    expect(ids).toEqual(sorted)
    await expect(page.locator('[data-glossary-count]')).toContainText(`${terms.length} of ${terms.length}`)

    const withAlias = terms.find((t) => t.aliases.length > 0)!
    const alias = withAlias.aliases[0]!
    await page.getByRole('searchbox').fill(alias)
    await expect(page.locator(`[data-glossary-term="${withAlias.id}"]`)).toHaveCount(1)
    const q = alias.toLowerCase()
    const expected = terms.filter((t) => [t.term, ...t.aliases].some((n) => n.toLowerCase().includes(q))).map((t) => t.id)
    await expect(page.locator('[data-glossary-term]')).toHaveCount(expected.length)
    await page.getByRole('searchbox').fill(withAlias.term.toUpperCase())
    await expect(page.locator(`[data-glossary-term="${withAlias.id}"]`)).toHaveCount(1)
    await page.getByRole('searchbox').fill('zzzz-no-such-term-zzzz')
    await expect(page.getByText('No terms match')).toBeVisible()
  })

  test('related links, deep links and "where this is taught" work', async ({ page, request }) => {
    const { terms } = await getJSON<{ terms: GlossaryTerm[] & { related: string[]; lesson_ref?: string }[] }>(request, '/api/glossary')
    const withRelated = (terms as unknown as { id: string; term: string; related: string[]; lesson_ref?: string }[]).find((t) => t.related.length > 0)!
    await page.goto(`/glossary#term-${withRelated.id}`)
    const entry = page.locator(`[data-glossary-term="${withRelated.id}"]`)
    await expect(entry).toBeVisible()
    await expect(entry).toHaveClass(/violet/)
    const rel = (terms as unknown as { id: string; term: string }[]).find((t) => t.id === withRelated.related[0])!
    await entry.getByRole('link', { name: rel.term, exact: true }).click()
    expect(page.url().endsWith(`#term-${rel.id}`)).toBeTruthy()
    const withRef = (terms as unknown as { id: string; lesson_ref?: string }[]).find((t) => t.lesson_ref)!
    await page.goto('/glossary')
    await page.getByRole('button', { name: 'A-Z' }).click()
    await page.locator(`[data-glossary-term="${withRef.id}"]`).getByRole('link', { name: 'Where this is taught →' }).click()
    await expect(page.locator('h1')).toBeVisible()
    expect(page.url()).toContain('/course/')
  })
})

test.describe('error handling', () => {
  test('a diagram that fails to load shows an error panel and the page still works', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    const [id, lesson] = [...map][0]!
    await page.route(`**/api/diagrams/${id}`, (r) => r.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"internal","message":"simulated failure"}' }))
    await page.goto(`/course/${lesson.sectionId}/${lesson.id}`)
    await expect(page.getByRole('alert').first()).toContainText('could not be loaded')
    await expect(page.locator('h1').first()).toBeVisible()
  })

  test('malformed diagram JSON is caught by the render boundary', async ({ page, request }) => {
    const map = await diagramsInUse(request)
    const [id, lesson] = [...map][0]!
    await page.route(`**/api/diagrams/${id}`, (r) => r.fulfill({ status: 200, contentType: 'application/json', body: `{"id":"${id}","title":"x"}` }))
    await page.goto(`/course/${lesson.sectionId}/${lesson.id}`)
    await expect(page.getByText('failed to render')).toBeVisible()
    await expect(page.locator('h1').first()).toBeVisible()
  })
})
