import assert from 'node:assert/strict'
import { test } from 'node:test'
import { layoutTopology } from './layout.ts'
import type { Topology } from '../api'

const edge = (id: string, profile: string, role: 'branch' | 'hub' = 'branch') => ({
  id,
  name: id,
  role,
  profile,
  interfaces: [],
  overrides: {},
})

const topo = (): Topology =>
  ({
    version: 1,
    meta: { id: 't', title: 'T', summary: 's', teaching: true },
    customer: { id: 'c', name: 'C' },
    segments: [],
    gateways: [
      { id: 'gw-1', region: 'r' },
      { id: 'gw-2', region: 'r' },
    ],
    links: [],
    profiles: [
      {
        id: 'spoke',
        name: 'Spoke',
        segments: ['global'],
        cloud_vpn: { branch_to_hub: { enabled: true, hubs: ['hub-a', 'cl'] } },
      },
      { id: 'hub', name: 'Hub', segments: ['global'] },
    ],
    sites: [
      { id: 's1', name: 'S1', region: 'r', edges: [edge('s-1', 'spoke'), edge('s-2', 'spoke'), edge('s-3', 'spoke')] },
      {
        id: 's2',
        name: 'S2',
        region: 'r',
        edges: [edge('hub-a', 'hub', 'hub'), edge('m-1', 'hub'), edge('m-2', 'hub')],
      },
    ],
    clusters: [{ id: 'cl', members: ['m-1', 'm-2'] }],
    non_sdwan_destinations: [
      { id: 'n-gw', via: 'gateway', type: 'x', gateway: 'gw-1' },
      { id: 'n-edge', via: 'edge', type: 'x', edge: 's-2' },
    ],
  }) as Topology

test('every node is placed once, inside the canvas, at a distinct position', () => {
  const l = layoutTopology(topo())
  const keys = new Set(l.nodes.map((n) => n.key))
  assert.equal(keys.size, l.nodes.length)
  const spots = new Set(l.nodes.map((n) => `${n.x},${n.y}`))
  assert.equal(spots.size, l.nodes.length)
  for (const n of l.nodes) {
    assert.ok(n.x > 0 && n.x < l.width, `${n.key} x`)
    assert.ok(n.y > 0 && n.y < l.height, `${n.key} y`)
  }
})

test('gateways are above hubs and hubs above the other Edges', () => {
  const l = layoutTopology(topo())
  const y = (k: string) => l.nodes.find((n) => n.key === k)!.y
  assert.ok(y('gateway:gw-1') < y('edge:hub-a'))
  assert.ok(y('edge:hub-a') < y('edge:s-1'))
  assert.equal(y('edge:hub-a'), y('edge:m-1'))
})

test('Hub order is kept and cluster members stay together', () => {
  const l = layoutTopology(topo())
  const x = (k: string) => l.nodes.find((n) => n.key === k)!.x
  assert.ok(x('edge:hub-a') < x('edge:m-1'))
  assert.ok(x('edge:m-1') < x('edge:m-2'))
})

test('the cluster box encloses its members', () => {
  const l = layoutTopology(topo())
  const box = l.boxes.find((b) => b.id === 'cl')!
  for (const id of ['m-1', 'm-2']) {
    const n = l.nodes.find((m) => m.key === `edge:${id}`)!
    assert.ok(n.x > box.x && n.x < box.x + box.width)
    assert.ok(n.y > box.y && n.y < box.y + box.height)
  }
})

test('a destination built by an Edge sits under that Edge; one built by a Gateway sits in the top row', () => {
  const l = layoutTopology(topo())
  const n = (k: string) => l.nodes.find((m) => m.key === k)!
  assert.equal(n('nsd:n-edge').x, n('edge:s-2').x)
  assert.ok(n('nsd:n-edge').y > n('edge:s-2').y)
  assert.equal(n('nsd:n-gw').y, n('gateway:gw-1').y)
})

test('many Edges wrap onto several rows instead of overflowing', () => {
  const t = topo()
  t.sites[0]!.edges = Array.from({ length: 14 }, (_, i) => edge(`b-${i}`, 'spoke'))
  const l = layoutTopology(t)
  const ys = new Set(l.nodes.filter((n) => n.id.startsWith('b-')).map((n) => n.y))
  assert.ok(ys.size >= 2)
  for (const n of l.nodes) assert.ok(n.y < l.height)
})
