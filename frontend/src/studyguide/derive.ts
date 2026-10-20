import type { Activity, GuideSection, GuideTopic, LessonStatus, StudyGuide } from '../api'

/** A topic the course offers something for. "none" topics are shown but never counted against the learner. */
export const isCovered = (t: GuideTopic) => t.coverage !== 'none'

/** A covered topic is done when every lesson mapped to it is completed. */
export const isTopicDone = (t: GuideTopic) => t.lessons.length > 0 && t.lessons.every((l) => l.status === 'completed')

export interface CoverageCounts {
  full: number
  partial: number
  none: number
}

export interface SectionStats {
  topics: number
  covered: number
  done: number
  byCoverage: CoverageCounts
  lessonsCompleted: number
  lessonsTotal: number
  /** Mean of the best quiz score of each course section the guide section's lessons belong to; null if none taken. */
  avgBestQuizPct: number | null
  labsCompleted: number
  labsTotal: number
}

/** Unique lessons of a guide section, in guide order. */
export function sectionLessons(s: GuideSection) {
  const seen = new Set<string>()
  const out = []
  for (const t of s.topics) {
    for (const l of t.lessons) {
      if (!seen.has(l.id)) {
        seen.add(l.id)
        out.push(l)
      }
    }
  }
  return out
}

export function sectionStats(s: GuideSection): SectionStats {
  const byCoverage: CoverageCounts = { full: 0, partial: 0, none: 0 }
  let done = 0
  for (const t of s.topics) {
    byCoverage[t.coverage]++
    if (isCovered(t) && isTopicDone(t)) done++
  }
  const lessons = sectionLessons(s)
  // The best score belongs to a course section, so average each distinct one once.
  const best = new Map<string, number>()
  for (const t of s.topics) {
    if (t.progress.best_section_score_pct == null) continue
    for (const l of t.lessons)
      best.set(l.section_id, Math.max(best.get(l.section_id) ?? 0, t.progress.best_section_score_pct))
  }
  const labs = new Map<string, boolean>()
  for (const t of s.topics)
    for (const l of t.labs) labs.set(l.id, (labs.get(l.id) ?? false) || l.status === 'completed')
  return {
    topics: s.topics.length,
    covered: s.topics.length - byCoverage.none,
    done,
    byCoverage,
    lessonsCompleted: lessons.filter((l) => l.status === 'completed').length,
    lessonsTotal: lessons.length,
    avgBestQuizPct: best.size === 0 ? null : [...best.values()].reduce((a, b) => a + b, 0) / best.size,
    labsCompleted: [...labs.values()].filter(Boolean).length,
    labsTotal: labs.size,
  }
}

export interface OverallStats {
  covered: number
  done: number
  notCovered: number
  /** done / covered, 0..1; 0 when nothing is covered. */
  fraction: number
}

/** Progress is computed over covered topics only; topics the course does not cover are reported separately. */
export function overallStats(g: StudyGuide): OverallStats {
  let covered = 0
  let done = 0
  let notCovered = 0
  for (const s of g.sections)
    for (const t of s.topics) {
      if (!isCovered(t)) notCovered++
      else {
        covered++
        if (isTopicDone(t)) done++
      }
    }
  return { covered, done, notCovered, fraction: covered === 0 ? 0 : done / covered }
}

/** Every lesson once, in guide order (the first topic that lists it wins), then the unmapped ones. */
export function guideLessonOrder(
  g: StudyGuide,
): { lesson: { id: string; title: string; section_id: string; status: LessonStatus }; guideSection: string | null }[] {
  const seen = new Set<string>()
  const out: ReturnType<typeof guideLessonOrder> = []
  for (const s of g.sections)
    for (const t of s.topics)
      for (const l of t.lessons)
        if (!seen.has(l.id)) {
          seen.add(l.id)
          out.push({ lesson: l, guideSection: s.id })
        }
  for (const l of g.unmapped_lessons)
    if (!seen.has(l.id)) {
      seen.add(l.id)
      out.push({ lesson: l, guideSection: null })
    }
  return out
}

/** The first lesson in guide order that is not completed. Informational: nothing is locked behind it. */
export function nextRecommended(g: StudyGuide) {
  return guideLessonOrder(g).find((x) => x.guideSection !== null && x.lesson.status !== 'completed')?.lesson ?? null
}

/** The guide topics a quiz topic tag belongs to, in guide order. */
export function topicsForQuizTag(g: StudyGuide, tag: string): { section: GuideSection; topic: GuideTopic }[] {
  const out: { section: GuideSection; topic: GuideTopic }[] = []
  for (const s of g.sections)
    for (const t of s.topics) if (t.quiz_topics.some((q) => q.topic === tag)) out.push({ section: s, topic: t })
  return out
}

// ---- activity ----

export const HEATMAP_WEEKS = 12

export type ActivityDay = { date: string; counts: Record<string, number>; total: number; future: boolean }

const pad = (n: number) => String(n).padStart(2, '0')
/** Local calendar date as YYYY-MM-DD. */
export const localDate = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`

/**
 * The heatmap grid: HEATMAP_WEEKS columns of 7 days, weeks starting on Monday, the last column containing `now`.
 * Days are the learner's local days. Days after `now` are marked future and never counted.
 */
export function heatmapWeeks(events: Activity['events'], now: Date): ActivityDay[][] {
  const byDay = new Map<string, Record<string, number>>()
  for (const e of events) {
    const k = localDate(new Date(e.at))
    const c = byDay.get(k) ?? {}
    c[e.kind] = (c[e.kind] ?? 0) + 1
    byDay.set(k, c)
  }
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const mondayOffset = (today.getDay() + 6) % 7
  const start = new Date(today)
  start.setDate(today.getDate() - mondayOffset - 7 * (HEATMAP_WEEKS - 1))
  const weeks: ActivityDay[][] = []
  for (let w = 0; w < HEATMAP_WEEKS; w++) {
    const week: ActivityDay[] = []
    for (let d = 0; d < 7; d++) {
      const day = new Date(start)
      day.setDate(start.getDate() + w * 7 + d)
      const key = localDate(day)
      const future = day > today
      const counts = future ? {} : (byDay.get(key) ?? {})
      week.push({ date: key, counts, total: Object.values(counts).reduce((a, b) => a + b, 0), future })
    }
    weeks.push(week)
  }
  return weeks
}

/** Intensity bucket 0..4 for a day's event count. */
export const intensity = (total: number) => (total <= 0 ? 0 : total <= 1 ? 1 : total <= 3 ? 2 : total <= 6 ? 3 : 4)

/** Events in the last 7 days grouped into sessions: a gap longer than `gapMinutes` starts a new one. */
export function sessionsThisWeek(events: Activity['events'], now: Date, gapMinutes = 30): number {
  const from = now.getTime() - 7 * 24 * 3600 * 1000
  const times = events
    .map((e) => new Date(e.at).getTime())
    .filter((t) => t >= from && t <= now.getTime())
    .sort((a, b) => a - b)
  let sessions = 0
  let last = -Infinity
  for (const t of times) {
    if (t - last > gapMinutes * 60 * 1000) sessions++
    last = t
  }
  return sessions
}

/** Polyline points for a sparkline of 0..100 values in a width x height box (one value gives a flat line). */
export function sparkPoints(values: number[], width: number, height: number, pad = 3): string {
  if (values.length === 0) return ''
  const x = (i: number) => (values.length === 1 ? width / 2 : pad + (i * (width - 2 * pad)) / (values.length - 1))
  const y = (v: number) => height - pad - (Math.max(0, Math.min(100, v)) / 100) * (height - 2 * pad)
  return values.map((v, i) => `${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(' ')
}
