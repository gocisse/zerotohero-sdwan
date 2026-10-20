import { getJSON, sendJSON } from './client'
import type { QuizQuestion } from './course'
import type { Submission } from './progress'
import type { LabChoice } from './labs'

export interface SelfCheckQuestion extends QuizQuestion {
  section_id: string
  section_title: string
}

export interface SelfCheckScreen {
  id: string
  prompt: string
  choices: LabChoice[]
  screen: { label: string; path: string }
  topology: string
}

export interface SelfCheck {
  questions: SelfCheckQuestion[]
  screens: SelfCheckScreen[]
  /** The pseudo-lab whose state the screen tasks read (open the screens with ?lab=<id>). */
  lab: string
  symptom: string
}

export interface SelfCheckGrade {
  score: number
  total: number
  results: {
    question_id: string
    topic: string
    answered: boolean
    correct: boolean
    partial_score: number
    given: unknown
    correct_answer: unknown
    explanation: string
    option_explanations: Record<string, string>
    lesson_ref?: string
  }[]
}

export interface SelfCheckScreenResult {
  correct: boolean
  given: string
  expected: string
  explanation: string
  detail: string
  assumptions: { number: number; text: string }[]
}

/** The self-check is ungraded: the server stores nothing for any of these calls. */
export const fetchSelfCheck = () => getJSON<SelfCheck>('/api/self-check')
export const gradeSelfCheck = (answers: Submission[]) =>
  sendJSON<SelfCheckGrade>('POST', '/api/self-check/grade', { answers })
export const checkSelfCheckScreen = (taskId: string, answer: string) =>
  sendJSON<SelfCheckScreenResult>('POST', `/api/self-check/screens/${encodeURIComponent(taskId)}/check`, { answer })
