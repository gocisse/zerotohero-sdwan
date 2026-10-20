import { getJSON, sendJSON } from './client'
import type { QuizQuestion } from './course'

/** single_choice / true_false: option id; multi_choice / ordering: option ids (ordering: in the chosen order). */
export type AnswerValue = string | string[]

export interface Submission {
  question_id: string
  answer: AnswerValue | null
}

export interface QuestionResult {
  question_id: string
  topic: string
  answered: boolean
  /** Exact match only. */
  correct: boolean
  /** 0..1; fractional only for multi_choice partial credit. */
  partial_score: number
  given: AnswerValue | null
  correct_answer: AnswerValue
  explanation: string
  /** Explanations for wrongly picked options only. */
  option_explanations: Record<string, string>
  lesson_ref?: string
}

export interface AttemptResult {
  attempt_id: number
  section_id: string
  score: number
  total: number
  percentage: number
  /** Informational only (>= 70%). Must never be used to lock or hide anything. */
  passed: boolean
  created_at: string
  results: QuestionResult[]
}

export interface AttemptSummary {
  id: number
  score: number
  total: number
  created_at: string
}

export type LessonStatus = 'not_started' | 'visited' | 'completed'

export interface SectionProgress {
  section_id: string
  lessons_completed: number
  lessons_total: number
  quiz_attempts: number
  best_score_pct: number | null
  latest_score_pct: number | null
  quiz_skipped: boolean
  quiz_skipped_at: string | null
  last_visited_at: string | null
}

export interface Progress {
  overall_percentage: number
  lessons_completed: number
  lessons_total: number
  lessons: { lesson_id: string; section_id: string; status: LessonStatus }[]
  sections: SectionProgress[]
  /** Informational: no lab is ever required. */
  labs: {
    total: number
    started: number
    completed: number
    skipped: number
    items: { lab_id: string; title: string; status: string; attempts: number; hints_used: number }[]
  }
}

export interface TopicMastery {
  topic: string
  answers: number
  accuracy_pct: number
  weak: boolean
  lessons: { id: string; title: string; section_id: string }[]
}

export interface ReviewQueue {
  due_count: number
  upcoming_count: number
  next_due_at: string | null
  session_size: number
  questions: QuizQuestion[]
}

export interface ReviewAnswersResponse {
  results: QuestionResult[]
  due_count: number
  upcoming_count: number
  next_due_at: string | null
}

const sec = (id: string) => `/api/sections/${encodeURIComponent(id)}`

export const submitQuizAttempt = (sectionId: string, answers: Submission[]) =>
  sendJSON<AttemptResult>('POST', `${sec(sectionId)}/quiz/attempts`, { answers })
export const fetchAttempts = (sectionId: string) =>
  getJSON<{ section_id: string; attempts: AttemptSummary[] }>(`${sec(sectionId)}/quiz/attempts`)
export const skipQuiz = (sectionId: string) =>
  sendJSON<{ section_id: string; quiz_skipped_at: string }>('POST', `${sec(sectionId)}/quiz/skip`)
export const putLessonProgress = (lessonId: string, status: 'visited' | 'completed') =>
  sendJSON<{ lesson_id: string; status: LessonStatus; completed_at: string | null }>(
    'PUT',
    `/api/lessons/${encodeURIComponent(lessonId)}/progress`,
    { status },
  )
export const fetchProgress = () => getJSON<Progress>('/api/progress')
export const fetchTopics = () => getJSON<{ topics: TopicMastery[] }>('/api/progress/topics')
export const fetchReviewQueue = () => getJSON<ReviewQueue>('/api/review/queue')
export const submitReviewAnswers = (answers: Submission[]) =>
  sendJSON<ReviewAnswersResponse>('POST', '/api/review/answers', { answers })
