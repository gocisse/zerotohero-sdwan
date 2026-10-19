import { getJSON } from './client'

export interface LessonSummary {
  id: string
  title: string
  summary: string
  estimated_minutes: number
}

export interface SectionSummary {
  id: string
  title: string
  summary: string
  estimated_minutes: number
  placeholder: boolean
  question_count: number
  lessons: LessonSummary[]
}

export interface Course {
  title: string
  description: string
  sections: SectionSummary[]
}

export interface Lesson {
  id: string
  section_id: string
  title: string
  summary: string
  estimated_minutes: number
  topics: string[]
  prerequisites: string[]
  /** Raw MDX body (frontmatter already removed). */
  source: string
}

export type QuestionType = 'single_choice' | 'multi_choice' | 'true_false' | 'ordering'

export interface QuizOption {
  id: string
  text: string
}

/** Quiz questions as served: no correct answers or explanations (grading is server-side). */
export interface QuizQuestion {
  id: string
  topic: string
  difficulty: 1 | 2 | 3
  type: QuestionType
  prompt: string
  options: QuizOption[]
}

export interface Quiz {
  section_id: string
  questions: QuizQuestion[]
}

export const fetchCourse = () => getJSON<Course>('/api/course')
export const fetchLesson = (id: string) => getJSON<Lesson>(`/api/lessons/${encodeURIComponent(id)}`)
export const fetchQuiz = (sectionId: string) => getJSON<Quiz>(`/api/sections/${encodeURIComponent(sectionId)}/quiz`)
