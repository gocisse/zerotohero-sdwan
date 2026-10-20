import type { Progress, TopicMastery } from '../api'

/**
 * The four things the course is for. Goal coverage is INFORMATIONAL: it is derived from lesson progress, quiz scores,
 * weak topics, labs and the last self-check (kept in this browser only), and nothing in the app reads it to restrict
 * anything. Section ids, quiz topic tags and lab ids come from the content; e2e/tests/self-check.spec.ts fails if one
 * of them no longer exists.
 */
export interface Goal {
  id: string
  title: string
  blurb: string
  sections: string[]
  /** Quiz topic tags; an entry ending in "-" matches every tag with that prefix. */
  topics: string[]
  labs: string[]
  explain: { id: string; text: string }[]
  find: { id: string; text: string }[]
}

export const GOALS: Goal[] = [
  {
    id: 'read-config',
    title: "Read an Edge's configuration",
    blurb: 'Open an Edge and tell which settings come from the Profile, which from the Edge, and which rule decides.',
    sections: ['configuration', 'architecture'],
    topics: ['config-', 'edge-role'],
    labs: ['video-wrong-link', 'capstone-3-change-request', 'capstone-1-monday-morning'],
    explain: [
      { id: 'cfg-inherit', text: 'how an Edge inherits a setting from its Profile and what an Override changes' },
      { id: 'cfg-rules', text: "how an Edge's own business rules and the Profile's rules fit together" },
      { id: 'cfg-segments', text: 'what a Segment is and why a Profile has to include it' },
    ],
    find: [
      { id: 'cfg-find-bp', text: "an Edge's Business Policy tab, and tell Rules From Profile from Edge Overrides" },
      { id: 'cfg-find-override', text: 'the Override check box beside an inherited setting on the Edge Device tab' },
    ],
  },
  {
    id: 'traffic-flow',
    title: 'How traffic flows',
    blurb: 'Follow one flow: which rule applies, which link and path it takes, and what happens when a link degrades.',
    sections: ['dmpo', 'configuration', 'architecture'],
    topics: ['dmpo-', 'config-business-policy', 'config-firewall', 'config-apps-qos', 'mtu-overhead'],
    labs: ['video-wrong-link', 'mtu-ipsec', 'capstone-1-monday-morning', 'capstone-2-hub-incident'],
    explain: [
      { id: 'flow-stages', text: 'the stages the flow tracer walks through (a teaching order, not a documented one)' },
      { id: 'flow-steering', text: 'what link steering does and how Mandatory differs from a preference' },
      { id: 'flow-dmpo', text: 'what DMPO measures and what it can do about an outage or a brownout' },
      { id: 'flow-mtu', text: 'why the tunnel overhead lowers the packet size that fits' },
    ],
    find: [
      { id: 'flow-find-links', text: 'the Links and Paths tabs of an Edge under Monitor > Edges' },
      { id: 'flow-find-logs', text: 'the firewall log that records traffic via a Hub' },
    ],
  },
  {
    id: 'tunnels',
    title: 'How tunnels form',
    blurb:
      'Know which tunnels exist and why: Gateway, Hub, dynamic branch-to-branch, and what changes when a peer is down.',
    sections: ['architecture', 'overlay'],
    topics: ['tunnel-architecture', 'underlay-overlay', 'gateway-role', 'overlay-'],
    labs: ['no-direct-tunnel', 'hub-down', 'capstone-2-hub-incident'],
    explain: [
      { id: 'tun-kinds', text: 'the difference between a Gateway tunnel, a Hub tunnel and a dynamic tunnel' },
      { id: 'tun-cloudvpn', text: 'what the Cloud VPN settings of a Profile switch on' },
      { id: 'tun-dynamic', text: 'when a dynamic branch-to-branch tunnel appears and when it goes away' },
      { id: 'tun-hub', text: 'what a Spoke does when its primary Hub does not answer' },
    ],
    find: [
      { id: 'tun-find-paths', text: 'the Paths tab of an Edge, and the List Paths test under Remote Diagnostics' },
      { id: 'tun-find-vpn', text: 'the Cloud VPN settings of a Profile and their Override on an Edge' },
    ],
  },
  {
    id: 'routes',
    title: 'How routes flow',
    blurb: 'Follow a prefix from a LAN router to another Edge, and say where it stops and why.',
    sections: ['routing'],
    topics: ['rt-'],
    labs: ['branch-b-cannot-reach', 'which-route-wins', 'capstone-1-monday-morning'],
    explain: [
      { id: 'rt-stages', text: 'the stages a route goes through: learned, advertised, redistributed, preferred' },
      { id: 'rt-redist', text: 'why a prefix can be known in the overlay and still not reach a LAN router' },
      { id: 'rt-pref', text: 'how route preference decides between two routes (lower wins)' },
    ],
    find: [
      { id: 'rt-find-dump', text: 'the Route Table Dump and List Routes per Prefix tests under Remote Diagnostics' },
      { id: 'rt-find-ofc', text: 'Configure > Overlay Flow Control and its Routes List' },
    ],
  },
]

export const topicMatches = (goal: Goal, topic: string) =>
  goal.topics.some((t) => (t.endsWith('-') ? topic.startsWith(t) : topic === t))

/** What one browser remembers about the last self-check (ungraded; never sent anywhere). */
export interface SelfCheckMemory {
  at: string
  perSection: Record<string, { correct: number; total: number }>
  screens: { correct: number; total: number }
}

const MEMORY_KEY = 'vca.selfcheck.v1'
const CHECKLIST_KEY = 'vca.checklist.v1'

function read<T>(key: string, fallback: T): T {
  try {
    const raw = window.localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

function write(key: string, value: unknown) {
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Storage can be unavailable (private window, blocked data); the page works without it.
  }
}

export const loadSelfCheck = (): SelfCheckMemory | null => read<SelfCheckMemory | null>(MEMORY_KEY, null)
export const saveSelfCheck = (m: SelfCheckMemory) => write(MEMORY_KEY, m)
export const loadChecklist = (): Record<string, boolean> => read<Record<string, boolean>>(CHECKLIST_KEY, {})
export const saveChecklist = (c: Record<string, boolean>) => write(CHECKLIST_KEY, c)

export interface GoalCoverage {
  goal: Goal
  lessons: number | null
  quiz: number | null
  labs: number | null
  selfCheck: number | null
  weakTopics: string[]
  checklist: { ticked: number; total: number }
  /** Mean of the parts that have data; null when nothing has been done yet. */
  overall: number | null
}

const mean = (xs: number[]) => (xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : null)

export function goalCoverage(
  goal: Goal,
  progress: Progress,
  topics: TopicMastery[] | null,
  memory: SelfCheckMemory | null,
  checklist: Record<string, boolean>,
): GoalCoverage {
  const secs = progress.sections.filter((s) => goal.sections.includes(s.section_id))
  const lessonsTotal = secs.reduce((n, s) => n + s.lessons_total, 0)
  const lessons = lessonsTotal ? (secs.reduce((n, s) => n + s.lessons_completed, 0) / lessonsTotal) * 100 : null
  const quiz = mean(secs.filter((s) => s.best_score_pct != null).map((s) => s.best_score_pct as number))
  const labItems = progress.labs.items.filter((l) => goal.labs.includes(l.lab_id))
  const labs = labItems.length
    ? (labItems.filter((l) => l.status === 'completed').length / labItems.length) * 100
    : null
  let selfCheck: number | null = null
  if (memory) {
    const parts = goal.sections
      .map((s) => memory.perSection[s])
      .filter((x): x is { correct: number; total: number } => !!x && x.total > 0)
    const total = parts.reduce((n, p) => n + p.total, 0)
    if (total) selfCheck = (parts.reduce((n, p) => n + p.correct, 0) / total) * 100
  }
  const items = [...goal.explain, ...goal.find]
  const parts = [lessons, quiz, labs, selfCheck].filter((x): x is number => x != null)
  return {
    goal,
    lessons,
    quiz,
    labs,
    selfCheck,
    weakTopics: (topics ?? []).filter((t) => t.weak && topicMatches(goal, t.topic)).map((t) => t.topic),
    checklist: { ticked: items.filter((i) => checklist[i.id]).length, total: items.length },
    overall: mean(parts),
  }
}
