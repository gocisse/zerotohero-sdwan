import { getJSON } from './client'

export type RunbookKind = 'checklist' | 'cheatsheet' | 'template' | 'reference'

export interface RunbookSummary {
  id: string
  title: string
  summary: string
  kind: RunbookKind
  estimated_minutes: number
}

export interface Runbook extends RunbookSummary {
  /** Raw MDX body (frontmatter removed). */
  source: string
}

export const fetchRunbooks = () => getJSON<{ runbooks: RunbookSummary[] }>('/api/runbooks')
export const fetchRunbook = (id: string) => getJSON<Runbook>(`/api/runbooks/${encodeURIComponent(id)}`)
