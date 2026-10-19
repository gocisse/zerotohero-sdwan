import { getJSON } from './client'

export type ServiceStatus = 'ok' | 'error'

export interface Health {
  status: ServiceStatus
  service: string
  database: { status: ServiceStatus; error?: string }
}

/** The backend answers 503 with a full body when the DB is down; that is still a valid report. */
export function fetchHealth(): Promise<Health> {
  return getJSON<Health>('/api/health', [503])
}
