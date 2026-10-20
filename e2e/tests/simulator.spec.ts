import { expect, test } from '@playwright/test'
import { existsSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { getJSON } from './support'

// Simulator tests. Everything is discovered through the API (topologies, edges, settings), so adding a topology
// file is covered automatically.
interface TopologySummary { id: string; title: string; edges: number }
interface Topo {
  topology: { profiles: { id: string; name: string }[]; sites: { edges: { id: string; name: string; profile: string }[] }[] }
  settings: { id: string; label: string; kind: string }[]
}
interface EffectiveDevice { id: string; label: string; value: unknown; source: { kind: string } }
interface Effective { device: EffectiveDevice[]; edge: { name: string } }

const BANNER = "Teaching model, not an emulator. Items marked 'assumption' are not verified."

async function topologies(request: import('@playwright/test').APIRequestContext) {
  const list = (await getJSON<{ topologies: TopologySummary[] }>(request, '/api/topologies')).topologies
  expect(list.length).toBeGreaterThan(0)
  return list
}

test.describe('simulator', () => {
  test('the picker lists every topology and the banner is on every screen', async ({ page, request }) => {
    const list = await topologies(request)
    await page.goto('/sim')
    await expect(page.locator('[data-sim-banner]')).toHaveText(BANNER)
    for (const t of list) await expect(page.locator(`[data-topology="${t.id}"]`)).toContainText(t.title)
    for (const t of list) {
      for (const path of ['', '/what-if', '/challenge']) {
        await page.goto(`/sim/${t.id}${path}`)
        await expect(page.locator('[data-sim-banner]'), `${t.id}${path}: banner`).toHaveText(BANNER)
      }
    }
  })

  test('every topology loads, lists its Edges, and every Edge shows source badges on all three tabs', async ({ page, request }) => {
    test.setTimeout(60_000 + 40_000 * (await topologies(request)).length)
    for (const t of await topologies(request)) {
      const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
      const edges = detail.topology.sites.flatMap((s) => s.edges)
      await page.goto(`/sim/${t.id}`)
      await expect(page.locator('[data-edge-list] [data-edge]')).toHaveCount(edges.length)
      for (const e of edges) {
        const eff = await getJSON<Effective>(request, `/api/topologies/${t.id}/edges/${e.id}/effective-config`)
        await page.goto(`/sim/${t.id}/edges/${e.id}`)
        await expect(page.locator('[data-edge-title]')).toHaveText(e.name)
        // Device tab: one row per setting, each with a source badge and a basis badge.
        await expect(page.locator('[data-setting]'), `${e.id}: settings`).toHaveCount(eff.device.filter((d) => d.id !== 'firewall.stateful_firewall').length)
        const rows = page.locator('[data-setting]')
        for (let i = 0; i < (await rows.count()); i++) {
          await expect(rows.nth(i).locator('[data-source]')).toHaveCount(1)
          await expect(rows.nth(i).locator('[data-basis]').first()).toBeVisible()
        }
        // Business Policy: the two real sections and an explicit not-modelled marker for the order.
        await page.getByRole('tab', { name: 'Business Policy' }).click()
        await expect(page.locator('[data-bp-section=edge-overrides]')).toBeVisible()
        await expect(page.locator('[data-bp-section=from-profile]')).toBeVisible()
        await expect(page.locator('[data-not-modelled]').first()).toContainText('Not modelled')
        // Firewall: Stateful Firewall with its source, and the groups.
        await page.getByRole('tab', { name: 'Firewall' }).click()
        await expect(page.locator('[data-setting="firewall.stateful_firewall"] [data-source]')).toHaveCount(1)
        await expect(page.locator('[data-fw-group-section=edge-override]')).toBeVisible()
      }
    }
  })

  test('badges show the documented sources: overridden, from Profile and default', async ({ page, request }) => {
    for (const t of await topologies(request)) {
      const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
      for (const e of detail.topology.sites.flatMap((s) => s.edges)) {
        const eff = await getJSON<Effective>(request, `/api/topologies/${t.id}/edges/${e.id}/effective-config`)
        for (const d of eff.device.filter((x) => x.id !== 'firewall.stateful_firewall')) {
          if (d.source.kind === 'edge-override') {
            await page.goto(`/sim/${t.id}/edges/${e.id}`)
            await expect(page.locator(`[data-setting="${d.id}"] [data-source=edge-override]`)).toContainText('overridden on this Edge')
            await expect(page.locator(`[data-setting="${d.id}"] input[type=checkbox]`)).toBeChecked()
            return
          }
        }
      }
    }
    throw new Error('no topology has an Edge override to check')
  })

  test('a basis badge tooltip names the fact sheet or the assumption', async ({ page, request }) => {
    const t = (await topologies(request))[0]!
    const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
    await page.goto(`/sim/${t.id}/edges/${detail.topology.sites[0]!.edges[0]!.id}`)
    await page.locator('[data-basis=verified]').first().focus()
    await expect(page.getByRole('tooltip')).toContainText('content/reference/')
    await page.keyboard.press('Escape')
    await expect(page.getByRole('tooltip')).toHaveCount(0)
    await page.locator('[data-basis=assumption]').first().focus()
    await expect(page.getByRole('tooltip')).toContainText('docs/SIMULATOR-MODEL.md')
  })

  test('the what-if panel returns and shows a diff for every Edge', async ({ page, request }) => {
    for (const t of await topologies(request)) {
      const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
      const edges = detail.topology.sites.flatMap((s) => s.edges)
      const profile = detail.topology.profiles[0]!
      await page.goto(`/sim/${t.id}/what-if`)
      await page.selectOption('[data-whatif-type]', 'set_profile_setting')
      await page.selectOption('[data-whatif-profile]', profile.id)
      await page.selectOption('[data-whatif-setting]', 'connectivity.arp_timeouts')
      await page.fill('[data-whatif-value]', '777')
      await page.click('[data-whatif-run]')
      await expect(page.locator('[data-whatif-result]')).toBeVisible()
      await expect(page.locator('[data-diff-edge]')).toHaveCount(edges.length)
      const statuses = await page.locator('[data-diff-edge]').evaluateAll((els) => els.map((e) => e.getAttribute('data-status')))
      expect(statuses.every((s) => ['changed', 'kept-override', 'unchanged'].includes(s ?? ''))).toBe(true)
      // Edges on the changed Profile either change or keep their override; others are untouched.
      for (const e of edges) {
        const eff = await getJSON<Effective>(request, `/api/topologies/${t.id}/edges/${e.id}/effective-config`)
        const arp = eff.device.find((d) => d.id === 'connectivity.arp_timeouts')!
        const want = e.profile !== profile.id ? 'unchanged' : arp.source.kind === 'edge-override' ? 'kept-override' : 'changed'
        await expect(page.locator(`[data-diff-edge="${e.id}"]`), `${t.id}/${e.id}`).toHaveAttribute('data-status', want)
      }
    }
  })

  test('moving an Edge to another Profile reports the side effects with their evidence', async ({ page, request }) => {
    const list = await topologies(request)
    let done = false
    for (const t of list) {
      const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
      if (detail.topology.profiles.length < 2) continue
      const edge = detail.topology.sites.flatMap((s) => s.edges)[0]!
      const target = detail.topology.profiles.find((p) => p.id !== edge.profile)!
      await page.goto(`/sim/${t.id}/what-if`)
      await page.selectOption('[data-whatif-type]', 'assign_profile')
      await page.selectOption('[data-whatif-edge]', edge.id)
      await page.selectOption('[data-whatif-profile]', target.id)
      await page.click('[data-whatif-run]')
      await expect(page.locator(`[data-diff-edge="${edge.id}"][data-status=changed]`)).toBeVisible()
      await expect(page.locator('[data-not-modelled]').first()).toContainText('Not modelled')
      await expect(page.locator(`[data-diff-edge="${edge.id}"] [data-diff-notes] [data-basis=not-modelled]`).first()).toBeVisible()
      done = true
      break
    }
    expect(done, 'a topology with two Profiles').toBe(true)
  })

  test('an invalid change shows an error instead of a diff', async ({ page, request }) => {
    const t = (await topologies(request))[0]!
    const res = await request.post(`/api/topologies/${t.id}/what-if`, { data: { type: 'set_profile_setting', profile: 'nope', setting: 'routing.bfd', value: true } })
    expect(res.status()).toBe(400)
    expect((await res.json()).error).toBe('invalid_change')
    void page
  })

  test('challenge mode reveals correct and incorrect answers and never blocks', async ({ page, request }) => {
    const t = (await topologies(request))[0]!
    const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
    const edges = detail.topology.sites.flatMap((s) => s.edges)
    const effective = new Map<string, Effective>()
    for (const e of edges) effective.set(e.name, await getJSON<Effective>(request, `/api/topologies/${t.id}/edges/${e.id}/effective-config`))
    await page.goto(`/sim/${t.id}/challenge`)
    const seen = new Set<string>()
    for (let i = 0; i < 24 && seen.size < 2; i++) {
      const q = (await page.locator('[data-challenge-question]').innerText()).trim()
      const m = /^Why does (.+?) have (.+?) = /.exec(q)!
      const eff = effective.get(m[1]!)!
      const setting = eff.device.find((d) => d.label === m[2])!
      const correct = setting.source.kind
      const options = ['profile', 'edge-override', 'default']
      const pick = seen.has('correct') || i % 2 === 1 ? options.find((o) => o !== correct)! : correct
      await page.locator(`[data-choice="${pick}"]`).check()
      const result = page.locator('[data-challenge-result]')
      await expect(result).toHaveAttribute('data-challenge-result', pick === correct ? 'correct' : 'incorrect')
      await expect(result).toContainText(/overrid|inherits|default/i)
      seen.add(pick === correct ? 'correct' : 'incorrect')
      await page.locator('[data-challenge-next]').click()
    }
    expect([...seen].sort()).toEqual(['correct', 'incorrect'])
    await expect(page.locator('main')).not.toContainText(/\block(ed)?\b|must pass|score/i)
  })

  test('the Edge page is keyboard operable (arrow keys move between tabs)', async ({ page, request }) => {
    const t = (await topologies(request))[0]!
    const detail = await getJSON<Topo>(request, `/api/topologies/${t.id}`)
    await page.goto(`/sim/${t.id}/edges/${detail.topology.sites[0]!.edges[0]!.id}`)
    await page.getByRole('tab', { name: 'Device' }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('tab', { name: 'Business Policy' })).toHaveAttribute('aria-selected', 'true')
    await page.keyboard.press('End')
    await expect(page.getByRole('tab', { name: 'Firewall' })).toHaveAttribute('aria-selected', 'true')
  })

  test('an invalid topology shows the validation error panel and the app recovers', async ({ page, request }) => {
    test.setTimeout(120_000)
    const file = join(process.cwd(), '..', 'content', 'topologies', 'zz-broken-e2e.yaml')
    const broken = `version: 1
meta: { id: zz-broken-e2e, title: Broken, summary: temporary file made by the e2e suite, teaching: true }
customer: { id: c, name: C }
links:
  - { id: l1, type: public, medium: wired, up_mbps: 1, down_mbps: 1 }
profiles:
  - { id: p1, name: P1, segments: [global] }
sites:
  - id: s1
    name: S1
    region: r
    edges:
      - id: e1
        name: E1
        role: branch
        profile: ghost
        interfaces: [{ id: ge1, kind: wan, link: l1 }]
      - id: e1
        name: E1 again
        role: branch
        profile: p1
        interfaces: [{ id: ge2, kind: wan, link: l1 }]
        overrides: { device: { made.up: 1 } }
`
    try {
      writeFileSync(file, broken)
      await expect
        .poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 })
        .toBe(503)
      const body = await (await request.get('/api/topologies')).json()
      const messages = (body.details as { file: string; message: string }[]).filter((d) => d.file === 'topologies/zz-broken-e2e.yaml').map((d) => d.message)
      expect(messages.some((m) => m.includes('unknown profile "ghost"'))).toBe(true)
      expect(messages.some((m) => m.includes('duplicate edge id'))).toBe(true)
      expect(messages.some((m) => m.includes('unknown setting'))).toBe(true)
      await page.goto('/sim')
      const panel = page.getByRole('alert').first()
      await expect(panel).toContainText('zz-broken-e2e.yaml')
      await expect(panel).toContainText('unknown profile')
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
    } finally {
      if (existsSync(file)) rmSync(file)
    }
    await expect.poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 }).toBe(200)
    await page.goto('/sim')
    await expect(page.locator('[data-topology-list]')).toBeVisible()
  })
})
