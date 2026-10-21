import { getJSON } from './client'
import type { LessonStatus } from './progress'

export type Coverage = 'full' | 'partial' | 'none'
export type TopicKind = 'concept' | 'config' | 'topology'

export interface GuideLesson {
  id: string
  title: string
  section_id: string
  status: LessonStatus
}

export interface GuideQuizTopic {
  topic: string
  answers: number
  /** null until the topic has been answered. */
  accuracy_pct: number | null
  weak: boolean
  due: number
}

export interface GuideLab {
  id: string
  title: string
  status: string
}

export interface SimScreen {
  id: string
  label: string
  /** Contains "{topology}" for screens that live inside a topology. */
  path: string
}

export interface GuideProgress {
  lessons_completed: number
  lessons_total: number
  best_section_score_pct: number | null
  topic_accuracy_pct: number | null
  quiz_answers: number
  labs_completed: number
  labs_total: number
  review_due: number
}

export interface GuideTopic {
  id: string
  title: string
  kind: TopicKind
  coverage: Coverage
  note: string
  lessons: GuideLesson[]
  quiz_topics: GuideQuizTopic[]
  diagrams: { id: string; title: string }[]
  labs: GuideLab[]
  sim_screens: SimScreen[]
  progress: GuideProgress
}

export interface GuideSection {
  id: string
  title: string
  /** True for a section the course added beyond the learner's original outline (G6). */
  added_by_course: boolean
  topics: GuideTopic[]
}

export interface StudyGuide {
  sections: GuideSection[]
  beyond: { title: string; reason: string }
  unmapped_lessons: GuideLesson[]
  warnings: { file: string; message: string }[]
  sim_screens: SimScreen[]
}

export type ActivityKind =
  'lesson_visited' | 'lesson_completed' | 'quiz_attempt' | 'review_answer' | 'lab_attempt' | 'lab_completed'

export interface Activity {
  since: string
  events: { kind: ActivityKind; at: string }[]
  last_lesson: { lesson_id: string; section_id: string; title: string; status: LessonStatus; at: string } | null
  quiz: { section_id: string; attempts: { score: number; total: number; created_at: string }[] }[]
  /** What the stored timestamps cannot show; the screen repeats it. */
  limits: string[]
}

export const fetchStudyGuide = () => getJSON<StudyGuide>('/api/study-guide')
export const fetchActivity = () => getJSON<Activity>('/api/activity')
