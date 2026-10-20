import { expect, test, type APIRequestContext } from '@playwright/test'
import { existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { getJSON } from './support'

// Flow tracer tests (step 14). Topologies, Edges and flows are discovered through the API; nothing is hard-coded.
interface Edge { id: string; name: string; routing?: unknown }
interface Topo { topology: { sites: { edges: Edge[] }[] } }
interface Trace {
  verdict: string
  blocked_stage: string
  summary: string
  stages: { id: string; status: string; not_modelled?: unknown[]; heuristic?: string }[]
  depends_on: { switch: { key: string }; new_outcome: string }[]
  assumptions: { key: string }[]
}

const post = async <T>(request: APIRequestContext, path: string, data: unknown): Promise<T> => {
  const res = await request.post(path, { data })
  expect(res.ok(), `${path} -> ${res.status()}`).toBeTruthy()
  return (await res.json()) as T
}

async function topologies(request: APIRequestContext) {
  const list = (await getJSON<{ topologies: { id: string }[] }>(request, '/api/topologies')).topologies
  const out: { id: string; edges: Edge[] }[] = []
  for (const x of list) out.push({ id: x.id, edges: (await getJSON<Topo>(request, `/api/topologies/${x.id}`)).topology.sites.flatMap((s) => s.edges) })
  return out
}

test.describe('flow tracer', () => {
  test('the trace loads for a flow in every topology and matches the API', async ({ page, request }) => {
    const all = await topologies(request)
    expect(all.length).toBeGreaterThan(0)
    test.setTimeout(60_000 + 20_000 * all.length)
    for (const x of all) {
      const [a, b] = [x.edges[0]!, x.edges[1] ?? x.edges[0]!]
      const flow = { source: a.id, destination: `edge:${b.id}` }
      const api = await post<Trace>(request, `/api/topologies/${x.id}/trace`, { flow })
      await page.goto(`/sim/${x.id}/trace`)
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      await page.selectOption('[data-trace-source]', a.id)
      await page.fill('[data-trace-dest]', flow.destination)
      await page.click('[data-trace-run]')
      const res = page.locator('[data-trace-result]')
      await expect(res, x.id).toHaveAttribute('data-verdict', api.verdict)
      await expect(page.locator('[data-trace-stage]'), x.id).toHaveCount(api.stages.length)
      await expect(page.locator('[data-trace-summary]')).toContainText(api.summary.slice(0, 30))
      const stages = page.locator('[data-trace-stage]')
      for (let i = 0; i < (await stages.count()); i++) await expect(stages.nth(i).locator('[data-basis]').first()).toBeVisible()
      // The 13 assumption switches are listed and marked as not verified.
      await expect(page.locator('[data-assumption]')).toHaveCount(api.assumptions.length)
      await expect(page.locator('[data-assumptions-panel]')).toContainText('not verified')
    }
  })

  test('the stage 8 asymmetry heuristic (assumption 88) is labelled "heuristic, may be wrong" wherever it is applied', async ({ page, request }) => {
    const all = await topologies(request)
    let seen = 0
    for (const x of all) {
      const [a, b] = [x.edges[0]!, x.edges[1] ?? x.edges[0]!]
      const flow = { source: a.id, destination: `edge:${b.id}` }
      const api = await post<Trace>(request, `/api/topologies/${x.id}/trace`, { flow })
      const ret = api.stages.find((st) => st.id === 'return')!
      await page.goto(`/sim/${x.id}/trace`)
      await page.selectOption('[data-trace-source]', a.id)
      await page.fill('[data-trace-dest]', flow.destination)
      await page.click('[data-trace-run]')
      await expect(page.locator('[data-trace-result]')).toBeVisible()
      const label = page.locator('[data-trace-stage="return"] [data-heuristic], [data-trace-stage][data-stage-id="return"] [data-heuristic]')
      if (ret.heuristic) {
        seen++
        expect(ret.heuristic).toContain('Heuristic, may be wrong')
        await expect(label.first(), x.id).toContainText('Heuristic, may be wrong')
        await expect(label.first()).toContainText('assumption 88')
      } else {
        await expect(page.locator('[data-heuristic]')).toHaveCount(0)
      }
    }
    expect(seen, 'at least one topology applies the heuristic').toBeGreaterThan(0)
  })

  test('a not-modelled box is shown and stays distinct', async ({ page, request }) => {
    const x = (await topologies(request))[0]!
    await page.goto(`/sim/${x.id}/trace`)
    await expect(page.locator('[data-trace-result]')).toBeVisible()
    const box = page.locator('[data-trace-stages] [data-not-modelled]').first()
    await expect(box).toBeVisible()
    await expect(box).toContainText('Not modelled')
    await expect(page.locator('[data-trace-not-modelled] [data-not-modelled]').first()).toBeVisible()
  })

  test('flipping an assumption the verdict depends on changes it and shows why', async ({ page, request }) => {
    const all = await topologies(request)
    let found: { id: string; flow: { source: string; destination: string }; trace: Trace } | undefined
    for (const x of all) {
      for (const e of x.edges) {
        const prefixes = x.edges.some((y) => y.routing) ? ['10.20.0.5', '10.40.0.9', '172.16.50.4'] : []
        for (const d of prefixes) {
          const flow = { source: e.id, destination: d }
          const t = await post<Trace>(request, `/api/topologies/${x.id}/trace`, { flow })
          if (t.depends_on.length > 0) { found = { id: x.id, flow, trace: t }; break }
        }
        if (found) break
      }
      if (found) break
    }
    expect(found, 'a routing flow whose verdict depends on an assumption').toBeDefined()
    const { id, flow, trace } = found!
    const dep = trace.depends_on[0]!
    await page.goto(`/sim/${id}/trace`)
    await page.selectOption('[data-trace-source]', flow.source)
    await page.fill('[data-trace-dest]', flow.destination)
    await page.click('[data-trace-run]')
    await expect(page.locator('[data-trace-result]')).toHaveAttribute('data-verdict', trace.verdict)
    await expect(page.locator(`[data-depends=${dep.switch.key}]`)).toBeVisible()
    const input = page.locator(`[data-assumption-input=${dep.switch.key}]`)
    if ((await input.evaluate((el) => el.tagName)) === 'SELECT') {
      const cur = await input.inputValue()
      const other = await input.evaluate((el, c) => [...(el as HTMLSelectElement).options].map((o) => o.value).find((v) => v !== c)!, cur)
      await input.selectOption(other)
    } else {
      await input.click()
    }
    await expect(page.locator('[data-trace-result]')).toHaveAttribute('data-verdict', dep.new_outcome)
    // Nothing is stored.
    await page.reload()
    await expect(page.locator(`[data-assumption=${dep.switch.key}]`)).toHaveAttribute('data-assumption-changed', 'false')
  })

  test('a blocked trace ends at the blocking stage and later stages are not reached', async ({ page, request }) => {
    const all = await topologies(request)
    let found: { id: string; flow: { source: string; destination: string }; trace: Trace } | undefined
    outer: for (const x of all) {
      for (const e of x.edges) {
        for (const o of x.edges) {
          if (o.id === e.id) continue
          const flow = { source: e.id, destination: `edge:${o.id}` }
          const t = await post<Trace>(request, `/api/topologies/${x.id}/trace`, { flow, assumptions: { fw_default_action: 'drop' } })
          if (t.verdict === 'blocked') { found = { id: x.id, flow, trace: t }; break outer }
        }
      }
    }
    expect(found).toBeDefined()
    const { id, flow, trace } = found!
    await page.goto(`/sim/${id}/trace`)
    await page.selectOption('[data-trace-source]', flow.source)
    await page.fill('[data-trace-dest]', flow.destination)
    await page.click('[data-trace-run]')
    await page.selectOption('[data-assumption-input=fw_default_action]', 'drop')
    await expect(page.locator('[data-trace-result]')).toHaveAttribute('data-blocked-stage', trace.blocked_stage)
    await expect(page.locator('[data-trace-stage][data-blocked-here=true]')).toHaveAttribute('data-trace-stage', trace.blocked_stage)
    await expect(page.locator('[data-trace-stage][data-status=not-reached]').first()).toBeVisible()
  })

  test('the trace screens do not overflow at 390px in dark mode', async ({ browser, request }) => {
    const x = (await topologies(request))[0]!
    const ctx = await browser.newContext({ viewport: { width: 390, height: 900 }, colorScheme: 'dark', hasTouch: true })
    const page = await ctx.newPage()
    for (const path of [`/sim/${x.id}/trace`, '/sim/labs']) {
      await page.goto(path)
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      await page.waitForTimeout(800)
      expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth), path).toBeLessThanOrEqual(0)
    }
    await ctx.close()
  })
})

test('a broken lab file produces the validation error panel instead of crashing, and the app recovers', async ({ page, request }) => {
  test.setTimeout(120_000)
  const dir = join(process.cwd(), '..', 'content', 'labs')
  const file = join(dir, 'zz-broken-lab-e2e.yaml')
  const first = (await getJSON<{ labs: { id: string }[] }>(request, '/api/labs')).labs[0]!.id
  let text = readFileSync(join(dir, `${first}.yaml`), 'utf8')
  text = text.replace(/^id: .*$/m, 'id: zz-broken-lab-e2e').replace(/^goals: \[/m, 'goals: [no-such-lesson, ')
  try {
    writeFileSync(file, text)
    await expect.poll(async () => (await request.get('/api/labs')).status(), { timeout: 20_000 }).toBe(503)
    const body = await (await request.get('/api/labs')).json()
    const msgs = (body.details as { file: string; message: string }[]).filter((d) => d.file === 'labs/zz-broken-lab-e2e.yaml').map((d) => d.message)
    expect(msgs.some((m) => m.includes('unknown lesson "no-such-lesson"'))).toBe(true)
    await page.goto('/sim/labs')
    await expect(page.getByRole('alert').first()).toContainText('zz-broken-lab-e2e.yaml')
    await expect(page.locator('[data-sim-banner]')).toBeVisible()
  } finally {
    if (existsSync(file)) rmSync(file)
  }
  await expect.poll(async () => (await request.get('/api/labs')).status(), { timeout: 20_000 }).toBe(200)
  await page.goto('/sim/labs')
  await expect(page.locator('[data-labs-page]')).toBeVisible()
})
