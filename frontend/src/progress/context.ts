import { createContext, useContext } from 'react'
import type { Progress, ReviewQueue } from '../api'

export interface ProgressContextValue {
  /** null until the first load finishes (or if the backend is unreachable). */
  progress: Progress | null
  review: Pick<ReviewQueue, 'due_count' | 'upcoming_count' | 'next_due_at'> | null
  /** Re-fetch progress and the review count; call after anything that changes them. */
  refresh: () => Promise<void>
}

export const ProgressContext = createContext<ProgressContextValue>({
  progress: null,
  review: null,
  refresh: async () => {},
})

export const useProgress = () => useContext(ProgressContext)
