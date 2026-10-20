import type { GlossaryTerm, StudyGuide } from '../api'

export interface GlossaryGroup {
  /** Stable key: the topic id, or "beyond". */
  key: string
  title: string
  /** Terms alphabetical within the group. */
  terms: GlossaryTerm[]
}

export interface GlossarySectionGroup {
  /** Guide section id (G1..) or "beyond". */
  key: string
  title: string
  groups: GlossaryGroup[]
}

export const BEYOND_KEY = 'beyond'

const alpha = (a: GlossaryTerm, b: GlossaryTerm) => a.term.localeCompare(b.term, undefined, { sensitivity: 'base' })

/**
 * The glossary arranged in the study guide's order. A term with several guide topics is listed under each (it is stored
 * once). Terms flagged beyond_guide land in a final group.
 */
export function groupGlossary(guide: StudyGuide, terms: GlossaryTerm[]): GlossarySectionGroup[] {
  const byTopic = new Map<string, GlossaryTerm[]>()
  const beyond: GlossaryTerm[] = []
  for (const t of terms) {
    if (t.beyond_guide || t.guide_topics.length === 0) beyond.push(t)
    for (const id of t.guide_topics) byTopic.set(id, [...(byTopic.get(id) ?? []), t])
  }
  const out: GlossarySectionGroup[] = guide.sections.map((s) => ({
    key: s.id,
    title: s.title,
    groups: s.topics.map((tp) => ({ key: tp.id, title: tp.title, terms: [...(byTopic.get(tp.id) ?? [])].sort(alpha) })),
  }))
  out.push({
    key: BEYOND_KEY,
    title: guide.beyond.title,
    groups: [{ key: BEYOND_KEY, title: guide.beyond.title, terms: [...beyond].sort(alpha) }],
  })
  return out
}

/** The group a term opens in for a deep link: its primary (first) guide topic, or "beyond". */
export const primaryGroup = (t: GlossaryTerm) => t.guide_topics[0] ?? BEYOND_KEY

/** Case-insensitive match on the term or any alias (the same rule as the A to Z view). */
export const termMatches = (t: GlossaryTerm, query: string) => {
  const q = query.trim().toLowerCase()
  return q === '' || [t.term, ...t.aliases].some((n) => n.toLowerCase().includes(q))
}

/** Splits text around case-insensitive matches of query, for highlighting. */
export function splitMatches(text: string, query: string): { text: string; match: boolean }[] {
  const q = query.trim().toLowerCase()
  if (!q) return [{ text, match: false }]
  const out: { text: string; match: boolean }[] = []
  const lower = text.toLowerCase()
  let i = 0
  for (let at = lower.indexOf(q, i); at >= 0; at = lower.indexOf(q, i)) {
    if (at > i) out.push({ text: text.slice(i, at), match: false })
    out.push({ text: text.slice(at, at + q.length), match: true })
    i = at + q.length
  }
  if (i < text.length) out.push({ text: text.slice(i), match: false })
  return out
}
