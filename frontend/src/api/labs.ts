import { getJSON, sendJSON } from './client'
import type { FlowSpec, TraceResult } from './sim'

export type LabStatus = 'not_started' | 'started' | 'completed' | 'skipped'

export interface LabProgress {
  status: LabStatus
  attempts: number
  hints_used: number
  last_attempt_at: string | null
  completed_at: string | null
}

export interface LabGoal {
  id: string
  title: string
  section_id: string
}

export interface LabSummary {
  id: string
  title: string
  summary: string
  difficulty: number
  estimated_minutes: number
  topology: string
  goals: LabGoal[]
  tasks: number
  progress: LabProgress
}

export interface LabChoice {
  id: string
  text: string
}

export interface LabTask {
  id: string
  type: 'inspect' | 'diagnose' | 'fix' | 'predict'
  prompt: string
  choices: LabChoice[]
}

export interface LabFix {
  id: string
  title: string
  description: string
}

export interface LabDetail extends LabSummary {
  symptom: string
  screens: { label: string; path: string }[]
  flow?: FlowSpec
  relies_on_assumptions: number[]
  task_list: LabTask[]
  fixes: LabFix[]
  hint_count: number
}

export interface LabCheck {
  task_id: string
  correct: boolean
  given: string
  expected: string
  explanation: string
  detail: string
  assumptions: { number: number; text: string }[]
  progress: LabProgress
}

export const fetchLabs = () => getJSON<{ labs: LabSummary[] }>('/api/labs').then((r) => r.labs)
export const fetchLab = (id: string) => getJSON<LabDetail>(`/api/labs/${encodeURIComponent(id)}`)
export const checkLabTask = (labId: string, taskId: string, answer: string, fixes: string[]) =>
  sendJSON<LabCheck>('POST', `/api/labs/${encodeURIComponent(labId)}/tasks/${encodeURIComponent(taskId)}/check`, {
    answer,
    fixes,
  })
export const applyLabFix = (labId: string, fixId: string, applied: string[]) =>
  sendJSON<{ applied: string[]; trace?: TraceResult }>('POST', `/api/labs/${encodeURIComponent(labId)}/fix`, {
    fix_id: fixId,
    applied,
  })
export const revealLabHint = (labId: string, n: number) =>
  sendJSON<{ index: number; total: number; task: string; text: string; progress: LabProgress }>(
    'POST',
    `/api/labs/${encodeURIComponent(labId)}/hints/${n}`,
  )
export const putLabStatus = (labId: string, status: 'started' | 'completed' | 'skipped') =>
  sendJSON<{ lab_id: string; progress: LabProgress }>('PUT', `/api/labs/${encodeURIComponent(labId)}/status`, {
    status,
  })

export interface LabDebriefItem {
  finding: string
  where: string
  unverified: boolean
  note?: string
}

export interface LabDebrief {
  lab_id: string
  available: boolean
  has_debrief: boolean
  items: LabDebriefItem[]
}

/** The debrief names the root causes, so the server sends it only once the lab is completed or skipped. */
export const fetchLabDebrief = (id: string) => getJSON<LabDebrief>(`/api/labs/${encodeURIComponent(id)}/debrief`)
