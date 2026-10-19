import { createContext, useContext } from 'react'
import type { ApiError, Course, LessonSummary } from '../api'

export type CourseState =
  { status: 'loading' } | { status: 'ready'; course: Course } | { status: 'error'; error: ApiError | Error }

export const CourseContext = createContext<CourseState>({ status: 'loading' })

export const useCourseState = () => useContext(CourseContext)

export interface FlatLesson extends LessonSummary {
  sectionId: string
}

/** All lessons in course order. */
export function flattenLessons(course: Course): FlatLesson[] {
  return course.sections.flatMap((s) => s.lessons.map((l) => ({ ...l, sectionId: s.id })))
}
