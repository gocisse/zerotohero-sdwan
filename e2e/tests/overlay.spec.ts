import { expect, test, type APIRequestContext } from '@playwright/test'
import { existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { getJSON } from './support'

// Overlay tests (step 12). Everything is discovered through the API: topologies, Edges, links, Hubs, tunnels.
interface Edge { id: string; name: string; profile: string; interfaces: { id: string; kind: string; link?: string }[] }
interface Topo {
  topology: {
    gateways: { id: string }[]
    links: { id: string }[]
    profiles: { id: string; cloud_vpn?: { branch_to_hub?: { hubs?: string[] }; branch_to_branch?: { enabled: boolean } } }[]
    sites: { edges: Edge[] }[]
    clusters?: { id: string; members: string[] }[]
    non_sdwan_destinations?: { id: string }[]
  }
}
interface Tun { id: string; type: string; state: string; reason_code: string; a: { id: string; name: string }; b: { id: string; name: string }; paths: unknown[] }
interface TunnelSet { tunnels: Tun[] }
interface Decision { options: { kind: string; usable: boolean; chosen: boolean; reason_code: string; reason: string }[]; chosen?: { label: string }; mtu?: unknown }

const post = async <T>(request: APIRequestContext, path: string, data: unknown): Promise<T> => {
  const res = await request.post(path, { data })
  expect(res.ok(), `${path} -> ${res.status()}`).toBeTruthy()
  return (await res.json()) as T
}
const tunnelsOf = (request: APIRequestContext, id: string, cond: unknown = {}) => post<TunnelSet>(request, `/api/topologies/${id}/tunnels`, cond)

async function allTopologies(request: APIRequestContext) {
  const list = (await getJSON<{ topologies: { id: string }[] }>(request, '/api/topologies')).topologies
  const out: { id: string; t: Topo['topology']; tunnels: Tun[] }[] = []
  for (const x of list) {
    const t = (await getJSON<Topo>(request, `/api/topologies/${x.id}`)).topology
    out.push({ id: x.id, t, tunnels: (await tunnelsOf(request, x.id)).tunnels })
  }
  return out
}
const existing = (t: Tun) => t.state !== 'not-established' && t.state !== 'torn-down'

test.describe('overlay', () => {
  test('the map and the tunnel list load for every topology that has tunnels', async ({ page, request }) => {
    const all = (await allTopologies(request)).filter((x) => x.tunnels.length > 0)
    expect(all.length).toBeGreaterThan(0)
    test.setTimeout(60_000 + 20_000 * all.length)
    for (const x of all) {
      await page.goto(`/sim/${x.id}/map`)
      await expect(page.locator('[data-tunnel-map]')).toBeVisible()
      const nodes = x.t.sites.flatMap((s) => s.edges).length + x.t.gateways.length + (x.t.non_sdwan_destinations?.length ?? 0)
      await expect(page.locator('[data-map-node]'), `${x.id}: nodes`).toHaveCount(nodes)
      const n = x.tunnels.filter(existing).length
      await expect(page.locator('[data-tunnel-count]')).toHaveText(`(${n})`)
      await expect(page.locator('[data-tunnel-row]')).toHaveCount(n)
      // Every tunnel row carries a basis badge.
      const rows = page.locator('[data-tunnel-row]')
      for (let i = 0; i < Math.min(await rows.count(), 12); i++) await expect(rows.nth(i).locator('[data-basis]').first()).toBeVisible()
      await expect(page.locator('[data-map-legend]')).toBeVisible()
      await expect(page.locator('[data-sim-banner]')).toBeVisible()
    }
  })

  test('toggling a link or a Hub condition changes the tunnel list', async ({ page, request }) => {
    const all = (await allTopologies(request)).filter((x) => x.tunnels.length > 0)
    let linkDone = false
    let hubDone = false
    for (const x of all) {
      // A link whose failure changes at least one tunnel's state.
      if (!linkDone) {
        for (const l of x.t.links) {
          const after = (await tunnelsOf(request, x.id, { links_down: [l.id] })).tunnels
          const changed = after.filter((t) => x.tunnels.find((b) => b.id === t.id)?.state !== t.state)
          if (changed.length === 0) continue
          await page.goto(`/sim/${x.id}/map`)
          await page.locator('[data-list-existing]').uncheck()
          await expect(page.locator('[data-tunnel-row]')).toHaveCount(x.tunnels.length)
          await page.selectOption(`[data-cond-link="${l.id}"]`, 'down')
          // Every tunnel row now shows the state the API computes for that condition.
          for (const t of after.slice(0, 30)) await expect(page.locator(`[data-tunnel-row="${t.id}"]`)).toHaveAttribute('data-state', t.state)
          await page.locator('[data-cond-reset]').click()
          await expect(page.locator(`[data-cond-link="${l.id}"]`)).toHaveValue('up')
          for (const t of x.tunnels.slice(0, 30)) await expect(page.locator(`[data-tunnel-row="${t.id}"]`)).toHaveAttribute('data-state', t.state)
          linkDone = true
          break
        }
      }
      // A Hub whose failure takes tunnels down.
      if (!hubDone) {
        const hubs = new Set<string>()
        for (const e of x.tunnels.filter((t) => t.type === 'hub')) hubs.add(e.b.id)
        const hub = [...hubs][0]
        if (hub) {
          const after = (await tunnelsOf(request, x.id, { hubs_unreachable: [hub] })).tunnels
          const down = after.filter((t) => t.state === 'down' && t.b.id === hub)
          expect(down.length).toBeGreaterThan(0)
          await page.goto(`/sim/${x.id}/map`)
          await page.locator(`[data-cond-hub="${hub}"]`).check()
          for (const t of down) await expect(page.locator(`[data-tunnel-row="${t.id}"]`)).toHaveAttribute('data-state', 'down')
          await expect(page.locator(`[data-map-node="edge:${hub}"]`)).toHaveAttribute('data-unreachable', 'true')
          await expect(page.locator(`[data-map-tunnel="${down[0]!.id}"] text`).first()).toHaveText('✕')
          hubDone = true
        }
      }
    }
    expect(linkDone, 'a topology with a link whose failure changes a tunnel').toBe(true)
    expect(hubDone, 'a topology with a Hub').toBe(true)
  })

  test('Orchestrator unreachable shows unknown tunnel states and a not-modelled note', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.tunnels.length > 0)!
    await page.goto(`/sim/${x.id}/map`)
    await page.locator('[data-cond-orchestrator]').check()
    await expect(page.locator('[data-cond-orchestrator-note]')).toContainText('Not modelled')
    await expect(page.locator('[data-tunnel-row]').first()).toHaveAttribute('data-state', 'unknown')
    await expect(page.locator('[data-not-modelled]').first()).toContainText('Not modelled')
  })

  test('Monitor > Edges has the Links and Paths tabs with the documented names and basis badges', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.tunnels.length > 0)!
    const edge = x.t.sites.flatMap((s) => s.edges)[0]!
    await page.goto(`/sim/${x.id}/monitor?edge=${edge.id}`)
    await expect(page.getByRole('tab', { name: 'Links' })).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('tab', { name: 'Paths' })).toBeVisible()
    // No invented tabs: only the two whose contents are in the fact sheets.
    await expect(page.getByRole('tab')).toHaveCount(2)
    const wan = edge.interfaces.filter((i) => i.kind === 'wan').length
    await expect(page.locator('[data-link-row]')).toHaveCount(wan)
    await page.locator('[data-links-groups]').check()
    await expect(page.locator('[data-links-group]').first()).toBeVisible()
    await page.keyboard.press('Tab')
    await page.getByRole('tab', { name: 'Links' }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('tab', { name: 'Paths' })).toHaveAttribute('aria-selected', 'true')
    const mine = x.tunnels.filter((t) => (t.a.id === edge.id || t.b.id === edge.id) && t.paths.length > 0)
    await expect(page.locator('[data-path-peer]')).toHaveCount(mine.length)
    await expect(page.locator('[data-path-row]').first()).toBeVisible()
    await expect(page.locator('[data-paths-list] [data-basis]').first()).toBeVisible()
  })

  test('the path explainer returns a decision with rejected options for every topology', async ({ page, request }) => {
    const all = (await allTopologies(request)).filter((x) => x.t.sites.flatMap((s) => s.edges).length >= 2)
    test.setTimeout(60_000 + 20_000 * all.length)
    for (const x of all) {
      const edges = x.t.sites.flatMap((s) => s.edges)
      const [a, b] = [edges[0]!, edges[1]!]
      const api = await post<Decision>(request, `/api/topologies/${x.id}/paths`, { flow: { source: a.id, destination: { kind: 'edge', id: b.id } }, conditions: {} })
      await page.goto(`/sim/${x.id}/paths`)
      await page.selectOption('[data-path-source]', a.id)
      await page.selectOption('[data-path-dest]', `edge:${b.id}`)
      await page.click('[data-path-run]')
      await expect(page.locator('[data-path-result]')).toBeVisible()
      await expect(page.locator('[data-reachability-assumed]')).toContainText('assumed')
      await expect(page.locator('[data-option]'), `${x.id}: options`).toHaveCount(api.options.length)
      await expect(page.locator('[data-option][data-chosen=true]')).toHaveCount(api.chosen ? 1 : 0)
      const rejected = api.options.filter((o) => !o.usable)
      await expect(page.locator('[data-option][data-usable=false]')).toHaveCount(rejected.length)
      const reasons = page.locator('[data-option][data-usable=false] [data-option-reason]')
      for (let i = 0; i < (await reasons.count()); i++) await expect(reasons.nth(i)).not.toHaveText('')
      if (api.chosen) await expect(page.locator('[data-path-why]')).toContainText(api.chosen.label)
    }
  })

  test('the path explainer follows a condition, steering and the MTU', async ({ page, request }) => {
    // Find a topology where two Edges send traffic through Hubs for VPN.
    for (const x of await allTopologies(request)) {
      const prof = x.t.profiles.find((p) => (p.cloud_vpn?.branch_to_hub?.hubs?.length ?? 0) >= 2)
      if (!prof) continue
      const spokes = x.t.sites.flatMap((s) => s.edges).filter((e) => e.profile === prof.id)
      if (spokes.length < 2) continue
      const [a, b] = [spokes[0]!, spokes[1]!]
      const hub = prof.cloud_vpn!.branch_to_hub!.hubs![0]!
      const flow = { source: a.id, destination: { kind: 'edge', id: b.id } }
      const base = await post<Decision>(request, `/api/topologies/${x.id}/paths`, { flow, conditions: {} })
      const failed = await post<Decision>(request, `/api/topologies/${x.id}/paths`, { flow, conditions: { hubs_unreachable: [hub] } })
      expect(failed.chosen?.label).not.toEqual(base.chosen?.label)
      await page.goto(`/sim/${x.id}/paths`)
      await page.selectOption('[data-path-source]', a.id)
      await page.selectOption('[data-path-dest]', `edge:${b.id}`)
      await page.click('[data-path-run]')
      await expect(page.locator('[data-path-why]')).toContainText(base.chosen!.label)
      await page.locator(`[data-cond-hub="${hub}"]`).check()
      await page.click('[data-path-run]')
      await expect(page.locator('[data-path-why]')).toContainText(failed.chosen!.label)
      await expect(page.locator('[data-steering]')).toBeVisible()
      await expect(page.locator('[data-mtu-range]')).toContainText('bytes')
      return
    }
    throw new Error('no topology with a Spoke Profile that has two Hubs')
  })

  test('the timeline player advances and ends with the dynamic tunnel torn down', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.t.profiles.some((p) => p.cloud_vpn?.branch_to_branch?.enabled))
    expect(x, 'a topology with Branch to Branch VPN').toBeTruthy()
    await page.goto(`/sim/${x!.id}/timeline`)
    await expect(page.locator('[data-timeline]')).toBeVisible()
    const stepText = async () => (await page.locator('[data-tl-step]').innerText()).trim()
    const total = Number((await stepText()).match(/of (\d+)/)![1])
    expect(total).toBeGreaterThanOrEqual(3)
    expect(await stepText()).toContain(`Frame 1 of ${total}`)
    await expect(page.locator('[data-tl-tunnel]').first()).toHaveAttribute('data-state', 'not-established')
    await page.locator('[data-tl-next]').click()
    await expect(page.locator('[data-tl-first-packet]')).toContainText('standing path')
    await expect(page.locator('[data-tl-tunnel]').first()).toHaveAttribute('data-state', 'up')
    // Play runs to the end by itself.
    await page.locator('[data-tl-reset]').click()
    await page.locator('[data-tl-play]').click()
    await expect(page.locator('[data-tl-step]')).toContainText(`Frame ${total} of ${total}`, { timeout: 20_000 })
    await expect(page.locator('[data-tl-tunnel]').first()).toHaveAttribute('data-state', 'torn-down')
    await expect(page.locator('[data-tl-changes]')).toContainText('torn down')
    // Previous goes back; Reset returns to the start.
    await page.locator('[data-tl-prev]').click()
    await expect(page.locator('[data-tl-tunnel]').first()).toHaveAttribute('data-state', 'up')
  })

  test('the timeline respects prefers-reduced-motion', async ({ page, request }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    const x = (await allTopologies(request)).find((y) => y.t.profiles.some((p) => p.cloud_vpn?.branch_to_branch?.enabled))!
    await page.goto(`/sim/${x.id}/timeline`)
    await expect(page.locator('[data-tl-marker]')).toBeVisible()
    expect(await page.locator('[data-tl-marker]').getAttribute('style')).toBeNull()
    await page.locator('[data-tl-next]').click()
    expect(await page.locator('[data-tl-marker]').getAttribute('style')).toBeNull()
  })

  test('Hub failover: local flows move to the secondary Hub, peer flows are not moved back until flushed', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.t.profiles.some((p) => (p.cloud_vpn?.branch_to_hub?.hubs?.length ?? 0) >= 2))
    expect(x, 'a topology with two Hubs').toBeTruthy()
    await page.goto(`/sim/${x!.id}/timeline`)
    await page.selectOption('[data-tl-scenario]', 'failover')
    await expect(page.locator('[data-tl-step]')).toContainText('of 4')
    await page.locator('[data-tl-next]').click() // hub down
    const local = await page.locator('[data-tl-local]').innerText()
    const peer = await page.locator('[data-tl-peer]').innerText()
    expect(local).toEqual(peer)
    await page.locator('[data-tl-next]').click() // hub returns
    await expect(page.locator('[data-tl-peer]')).toHaveText(peer) // not moved back
    await expect(page.locator('[data-tl-failovers]')).toContainText('not moved back')
    await expect(page.locator('[data-tl-failovers]')).toContainText('not documented')
    await page.locator('[data-tl-next]').click() // flush
    await expect(page.locator('[data-tl-peer]')).not.toHaveText(peer)
  })

  test('the overlay challenge reveals correct and incorrect answers (no direct tunnel)', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.t.profiles.some((p) => p.cloud_vpn?.branch_to_branch?.enabled))!
    const edges = x.t.sites.flatMap((s) => s.edges)
    const byName = new Map(edges.map((e) => [e.name, e.id]))
    const ago = (c: string) => (c.includes('Nobody') ? null : c.includes('200') ? 200 : 5)
    await page.goto(`/sim/${x.id}/overlay-challenge`)
    await expect(page.locator('[data-oc-question]')).toBeVisible()
    const seen = new Set<string>()
    for (let round = 0; round < 2; round++) {
      const prompt = await page.locator('[data-oc-question]').innerText()
      const m = /between (.+) and (.+)\?$/.exec(prompt.trim())!
      const [ia, ib] = [byName.get(m[1]!)!, byName.get(m[2]!)!]
      const v = ago(await page.locator('[data-oc-context]').innerText())
      const cond = v === null ? {} : { traffic: edges.flatMap((p, i) => edges.slice(i + 1).map((q) => ({ a: p.id, b: q.id, first_packet_ago_s: v + 60, last_packet_ago_s: v }))) }
      const ts = await tunnelsOf(request, x.id, cond)
      const tun = ts.tunnels.find((t) => t.type === 'dynamic' && [t.a.id, t.b.id].sort().join() === [ia, ib].sort().join())!
      const choices = await page.locator('[data-oc-choice]').evaluateAll((els) => els.map((e) => e.getAttribute('data-oc-choice')!))
      expect(choices).toContain(tun.reason_code)
      const pick = round === 0 ? tun.reason_code : choices.find((c) => c !== tun.reason_code)!
      await page.locator(`[data-oc-choice="${pick}"]`).check()
      const result = page.locator('[data-oc-result]')
      await expect(result).toHaveAttribute('data-oc-result', pick === tun.reason_code ? 'correct' : 'incorrect')
      await expect(page.locator('[data-oc-explanation]')).not.toHaveText('')
      await expect(result.locator('[data-basis]').first()).toBeVisible()
      seen.add(pick === tun.reason_code ? 'correct' : 'incorrect')
      await page.locator('[data-oc-next]').click()
    }
    expect([...seen].sort()).toEqual(['correct', 'incorrect'])
    await expect(page.locator('main')).not.toContainText(/\block(ed)?\b|must pass|score/i)
  })

  test('the overlay challenge asks why a flow goes through the Hub', async ({ page, request }) => {
    const x = (await allTopologies(request)).find((y) => y.t.profiles.some((p) => (p.cloud_vpn?.branch_to_hub?.hubs?.length ?? 0) >= 2))!
    await page.goto(`/sim/${x.id}/overlay-challenge`)
    await page.selectOption('[data-oc-kind]', 'hub')
    await expect(page.locator('[data-oc-question]')).toContainText('goes through')
    const choices = await page.locator('[data-oc-choice]').evaluateAll((els) => els.map((e) => e.getAttribute('data-oc-choice')!))
    expect(choices.length).toBe(4)
    await page.locator('[data-oc-choice]').first().check()
    await expect(page.locator('[data-oc-result]')).toBeVisible()
    await expect(page.locator('[data-oc-explanation]')).not.toHaveText('')
  })

  test('an invalid overlay topology shows the validation error panel and the app recovers', async ({ page, request }) => {
    test.setTimeout(120_000)
    const dir = join(process.cwd(), '..', 'content', 'topologies')
    const file = join(dir, 'zz-broken-overlay-e2e.yaml')
    let text = readFileSync(join(dir, 'hub-and-spoke-demo.yaml'), 'utf8')
    text = text.replace('id: hub-and-spoke-demo', 'id: zz-broken-overlay-e2e')
    text = text.replace('hubs: [hub-p-edge, hub-s-edge]', 'hubs: [hub-p-edge, hub-p-edge]') // Hub order duplicated
    text = text.replace('members: [hub-c1, hub-c2]', 'members: [hub-c1, azure-vhub]') // cluster member that is not an Edge
    text = text.replace('gateway: gw-1\n  - id: aws-vpn', 'gateway: gw-missing\n  - id: aws-vpn') // NSD pointing at a missing Gateway
    try {
      writeFileSync(file, text)
      await expect.poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 }).toBe(503)
      const body = await (await request.get('/api/topologies')).json()
      const msgs = (body.details as { file: string; message: string }[]).filter((d) => d.file === 'topologies/zz-broken-overlay-e2e.yaml').map((d) => d.message)
      expect(msgs.some((m) => m.includes('Hub order must be unambiguous'))).toBe(true)
      expect(msgs.some((m) => m.includes('is not an Edge of this topology'))).toBe(true)
      expect(msgs.some((m) => m.includes('unknown Gateway "gw-missing"'))).toBe(true)
      for (const path of ['/sim', '/sim/three-site/map']) {
        await page.goto(path)
        await expect(page.getByRole('alert').first()).toContainText('zz-broken-overlay-e2e.yaml')
        await expect(page.locator('[data-sim-banner]')).toBeVisible()
      }
    } finally {
      if (existsSync(file)) rmSync(file)
    }
    await expect.poll(async () => (await request.get('/api/topologies')).status(), { timeout: 20_000 }).toBe(200)
    await page.goto('/sim/hub-and-spoke-demo/map')
    await expect(page.locator('[data-tunnel-map]')).toBeVisible()
  })
})
