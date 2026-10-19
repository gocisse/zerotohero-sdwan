import type { Course } from '../api'
import { flattenLessons } from '../course/context'

/** Path to a lesson by id (the section is looked up), or undefined if it is not in the course. */
export function lessonPath(course: Course, lessonId: string): string | undefined {
  const l = flattenLessons(course).find((x) => x.id === lessonId)
  return l && `/course/${l.sectionId}/${l.id}`
}

/** Where "continue" leads after a section: the first lesson of the next section, else the progress page. */
export function nextSectionPath(course: Course, sectionId: string): { to: string; isLast: boolean } {
  const i = course.sections.findIndex((s) => s.id === sectionId)
  const next = course.sections[i + 1]
  const first = next?.lessons[0]
  return next && first ? { to: `/course/${next.id}/${first.id}`, isLast: false } : { to: '/progress', isLast: true }
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function pct(n: number | null | undefined): string {
  return n == null ? '—' : `${Math.round(n)}%`
}
