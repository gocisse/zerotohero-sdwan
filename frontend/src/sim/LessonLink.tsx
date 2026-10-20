import { Link } from 'react-router-dom'
import { useCourseState } from '../course/context'
import { lessonHref } from './routingQuery'

function useLessonTitle(ref: string): string {
  const course = useCourseState()
  if (course.status !== 'ready') return ref
  for (const s of course.course.sections) for (const l of s.lessons) if (l.id === ref) return l.title
  return ref
}

/** A link to the lesson with the given id (the section follows from the id's prefix). */
export function LessonLink({ id, prefix = 'Read' }: { id: string; prefix?: string }) {
  const title = useLessonTitle(id)
  return (
    <Link to={lessonHref(id)} className="font-medium underline" data-lesson-link={id}>
      {prefix}: {title}
    </Link>
  )
}
