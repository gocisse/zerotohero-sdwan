import { expect, test, type APIRequestContext } from '@playwright/test'
import { existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { getJSON } from './support'

// Routing tests (step 13). Topologies, Edges, prefixes and answers are discovered through the API; nothing is hard-coded.
interface Edge { id: string; name: string; routing?: unknown }
interface Topo { topology: { sites: { edges: Edge[] }[] } }
interface Route { prefix: string; selected: boolean; reachable: boolean; source: string; edge?: string }
interface RouteSet {
  edges: { edge: string; name: string; segments: { segment: string; routes: Route[] }[]; lan_advertisements: unknown[] }[]
  assumptions: { key: string; used: boolean; assumption: number; value: boolean | string; changed: boolean }[]
}
interface Reach {
  outcome: string
  first_failing: string
  reason: string
  lesson_ref: string
  prefix: string
  stages: { id: string; status: string }[]
  depends_on: { switch: { key: string }; new_outcome: string }[]
}

const post = async <T>(request: APIRequestContext, path: string, data: unknown): Promise<T> => {
  const res = await request.post(path, { data })
  expect(res.ok(), `${path} -> ${res.status()}`).toBeTruthy()
  return (await res.json()) as T
}

async function routingTopologies(request: APIRequestContext) {
  const list = (await getJSON<{ topologies: { id: string }[] }>(request, '/api/topologies')).topologies
  const out: { id: string; edges: Edge[]; routes: RouteSet }[] = []
  for (const x of list) {
    const t = (await getJSON<Topo>(request, `/api/topologies/${x.id}`)).topology
    const edges = t.sites.flatMap((s) => s.edges)
    if (!edges.some((e) => e.routing)) continue
    out.push({ id: x.id, edges, routes: await post<RouteSet>(request, `/api/topologies/${x.id}/routes`, {}) })
  }
  return out
}

const prefixesOf = (rs: RouteSet) => [...new Set(rs.edges.flatMap((e) => e.segments.flatMap((s) => s.routes.map((r) => r.prefix))))]

/** The first question (source Edge, prefix) whose answer matches `want`. */
async function findQuery(request: APIRequestContext, id: string, edges: Edge[], rs: RouteSet, want: (r: Reach) => boolean) {
  for (const e of edges)
    for (const p of prefixesOf(rs).slice(0, 12)) {
      const r = await post<Reach>(request, `/api/topologies/${id}/reachability`, { query: { source: e.id, destination: p } })
      if (want(r)) return { edge: e, prefix: p, result: r }
    }
  return undefined
}

test.describe('routing', () => {
  test('the route table loads for every topology with routing data and matches the API', async ({ page, request }) => {
    const all = await routingTopologies(request)
    expect(all.length).toBeGreaterThan(0)
    test.setTimeout(60_000 + 20_000 * all.length)
    for (const x of all) {
      await page.goto(`/sim/${x.id}/routes`)
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      await expect(page.locator('[data-route-table]')).toBeVisible()
      for (const er of x.routes.edges.slice(0, 3)) {
        await page.selectOption('[data-routes-edge]', er.edge)
        const seg = er.segments[0]!
        await expect(page.locator('[data-route-row]'), `${x.id}/${er.edge}`).toHaveCount(seg.routes.length)
        const selected = seg.routes.filter((r) => r.selected).length
        await expect(page.locator('[data-route-row][data-selected=true]')).toHaveCount(selected)
        // Every row carries a basis badge.
        const rows = page.locator('[data-route-row]')
        for (let i = 0; i < Math.min(await rows.count(), 6); i++) await expect(rows.nth(i).locator('[data-basis]').first()).toBeVisible()
      }
      // The assumptions panel lists every switch the API returns, each marked as not verified.
      await expect(page.locator('[data-assumption]')).toHaveCount(x.routes.assumptions.length)
      await expect(page.locator('[data-assumptions-panel]')).toContainText('not verified')
    }
  })

  test('toggling an assumption changes the result and shows the assumption', async ({ page, request }) => {
    const all = await routingTopologies(request)
    let found: { id: string; q: NonNullable<Awaited<ReturnType<typeof findQuery>>> } | undefined
    for (const x of all) {
      const q = await findQuery(request, x.id, x.edges, x.routes, (r) => r.outcome !== 'not-modelled' && r.depends_on.length > 0)
      if (q) {
        found = { id: x.id, q }
        break
      }
    }
    test.skip(!found, 'no answer in any topology depends on an assumption')
    const { id, q } = found!
    const dep = q.result.depends_on[0]!
    await page.goto(`/sim/${id}/reachability`)
    await page.selectOption('[data-reach-source]', q.edge.id)
    await page.fill('[data-reach-dest]', q.prefix)
    await page.click('[data-reach-run]')
    const result = page.locator('[data-reach-result]')
    await expect(result).toHaveAttribute('data-outcome', q.result.outcome)
    await expect(page.locator('[data-reach-depends]')).toContainText('assumption')
    await expect(page.locator(`[data-depends=${dep.switch.key}]`)).toBeVisible()
    const input = page.locator(`[data-assumption-input=${dep.switch.key}]`)
    await expect(page.locator(`[data-assumption=${dep.switch.key}] [data-assumption-used]`)).toBeVisible()
    if ((await input.evaluate((el) => el.tagName)) === 'SELECT') {
      const other = await input.evaluate((el, cur) => [...(el as HTMLSelectElement).options].map((o) => o.value).find((v) => v !== cur)!, await input.inputValue())
      await input.selectOption(other)
    } else {
      await input.click()
    }
    await expect(result).toHaveAttribute('data-outcome', dep.new_outcome)
    await expect(page.locator(`[data-assumption=${dep.switch.key}]`)).toHaveAttribute('data-assumption-changed', 'true')
    // Nothing is stored: a reload starts from the documented defaults again.
    await page.reload()
    await expect(page.locator(`[data-assumption=${dep.switch.key}]`)).toHaveAttribute('data-assumption-changed', 'false')
  })

  test('the reachability explainer reports a first failing stage with its reason and a lesson link', async ({ page, request }) => {
    const all = await routingTopologies(request)
    let found: { id: string; q: NonNullable<Awaited<ReturnType<typeof findQuery>>> } | undefined
    for (const x of all) {
      const q = await findQuery(request, x.id, x.edges, x.routes, (r) => r.outcome === 'unreachable')
      if (q) {
        found = { id: x.id, q }
        break
      }
    }
    expect(found, 'at least one routing topology has an unreachable prefix under the defaults').toBeDefined()
    const { id, q } = found!
    await page.goto(`/sim/${id}/reachability`)
    await page.selectOption('[data-reach-source]', q.edge.id)
    await page.fill('[data-reach-dest]', q.prefix)
    await page.click('[data-reach-run]')
    await expect(page.locator('[data-reach-result]')).toHaveAttribute('data-first-failing', q.result.first_failing)
    await expect(page.locator('[data-reach-reason]')).toContainText('First failing stage')
    await expect(page.locator(`[data-stage=${q.result.first_failing}]`)).toHaveAttribute('data-first-failing', 'true')
    await expect(page.locator('[data-stage]')).toHaveCount(q.result.stages.length)
    await expect(page.locator(`[data-reach-result] [data-lesson-link=${q.result.lesson_ref}]`).first()).toBeVisible()
    // And a reachable one says so.
    const ok = await findQuery(request, id, found!.q.edge ? [q.edge] : [], all.find((x) => x.id === id)!.routes, (r) => r.outcome === 'reachable')
    if (ok) {
      await page.fill('[data-reach-dest]', ok.prefix)
      await page.click('[data-reach-run]')
      await expect(page.locator('[data-reach-result]')).toHaveAttribute('data-outcome', 'reachable')
    }
  })

  test('the routing challenge reveals correct and incorrect answers and never blocks', async ({ page, request }) => {
    const all = await routingTopologies(request)
    let target: (typeof all)[number] | undefined
    for (const x of all) if (await findQuery(request, x.id, x.edges, x.routes, (r) => r.outcome === 'unreachable')) target = x
    expect(target, 'a routing topology with an unreachable prefix').toBeDefined()
    const x = target!
    await page.goto(`/sim/${x.id}/routing-challenge`)
    await expect(page.locator('[data-rc-question]')).toBeVisible()
    const answerFor = async () => {
      const text = (await page.locator('[data-rc-question]').innerText()).trim()
      const m = /^Why can't (.+) reach (\S+)\?$/.exec(text)!
      const edge = x.edges.find((e) => e.name === m[1])!
      const r = await post<Reach>(request, `/api/topologies/${x.id}/reachability`, { query: { source: edge.id, destination: m[2] } })
      return r.first_failing
    }
    // Correct answer.
    let correct = await answerFor()
    await page.locator(`[data-rc-choice=${correct}]`).check()
    await expect(page.locator('[data-rc-result]')).toHaveAttribute('data-rc-result', 'correct')
    await expect(page.locator('[data-rc-explanation]')).toContainText(correct)
    await expect(page.locator('[data-rc-depends], [data-rc-nodepends]').first()).toBeVisible()
    // Incorrect answer on the next question; the next button is never disabled.
    await page.click('[data-rc-next]')
    correct = await answerFor()
    const wrong = await page.locator('[data-rc-choice]').evaluateAll((els, c) => els.map((e) => (e as HTMLElement).dataset.rcChoice!).find((v) => v !== c)!, correct)
    await page.locator(`[data-rc-choice=${wrong}]`).check()
    await expect(page.locator('[data-rc-result]')).toHaveAttribute('data-rc-result', 'incorrect')
    await expect(page.locator('[data-rc-explanation]')).toContainText(correct)
    await expect(page.locator('[data-rc-next]')).toBeEnabled()
  })

  test('the Section 6 lessons link into the simulator', async ({ page, request }) => {
    const all = await routingTopologies(request)
    const id = all[0]!.id
    for (const [lesson, path] of [['rt-big-picture', 'routes'], ['rt-redistribution', 'reachability'], ['rt-diagnosis', 'reachability']] as const) {
      await page.goto(`/course/routing/${lesson}`)
      const link = page.getByRole('link', { name: 'Try it in the simulator' })
      await expect(link).toBeVisible()
      await link.click()
      await expect(page).toHaveURL(new RegExp(`/sim/[^/]+/${path}$`))
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      expect(page.url()).toContain('/sim/')
    }
    expect(id).toBeTruthy()
  })

  test('an invalid routing topology shows the validation error panel and the app recovers', async ({ page, request }) => {
    test.setTimeout(120_000)
    const dir = join(process.cwd(), '..', 'content', 'topologies')
    const file = join(dir, 'zz-broken-routing-e2e.yaml')
    let text = readFileSync(join(dir, 'routing-demo.yaml'), 'utf8')
    text = text.replace('id: routing-demo', 'id: zz-broken-routing-e2e')
    // a duplicated neighbor, OSPF full mode on a switched interface, a filter that references an unknown neighbor filter
    text = text.replace("- { id: lan-r1, ip: 10.30.255.2, asn: 65002, inbound_filter: from-lan }", "- { id: lan-r1, ip: 10.30.255.2, asn: 65002, inbound_filter: ghost-filter }\n              - { id: lan-r2, ip: 10.30.255.2, asn: 65002 }")
    text = text.replace('- { id: ge3, kind: routed, address: 10.20.255.1/30 }', '- { id: ge3, kind: lan, vlan: 10, address: 10.20.255.1/30 }')
    try {
      writeFileSync(file, text)
      await expect.poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 }).toBe(503)
      const body = await (await request.get('/api/topologies')).json()
      const msgs = (body.details as { file: string; message: string }[]).filter((d) => d.file === 'topologies/zz-broken-routing-e2e.yaml').map((d) => d.message)
      expect(msgs.some((m) => m.includes('duplicate neighbor'))).toBe(true)
      expect(msgs.some((m) => m.includes('supports OSPF only as passive'))).toBe(true)
      expect(msgs.some((m) => m.includes('unknown filter "ghost-filter"'))).toBe(true)
      for (const path of ['/sim', '/sim/three-site/routes']) {
        await page.goto(path)
        await expect(page.getByRole('alert').first()).toContainText('zz-broken-routing-e2e.yaml')
        await expect(page.locator('[data-sim-banner]')).toBeVisible()
      }
    } finally {
      if (existsSync(file)) rmSync(file)
    }
    await expect.poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 }).toBe(200)
    await page.goto('/sim/routing-demo/routes')
    await expect(page.locator('[data-route-table]')).toBeVisible()
  })

  test('the routing screens do not overflow at 390px in dark mode', async ({ browser, request }) => {
    const all = await routingTopologies(request)
    const ctx = await browser.newContext({ viewport: { width: 390, height: 900 }, colorScheme: 'dark', hasTouch: true })
    const page = await ctx.newPage()
    for (const screen of ['routes', 'reachability', 'routing-challenge']) {
      await page.goto(`/sim/${all[0]!.id}/${screen}`)
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
      await page.waitForTimeout(800)
      const over = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(over, `${screen}: horizontal page overflow`).toBeLessThanOrEqual(0)
    }
    await ctx.close()
  })
})
