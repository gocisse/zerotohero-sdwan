import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { fetchProgress, fetchReviewQueue } from '../api'
import { ProgressContext, type ProgressContextValue } from './context'

type Snapshot = Pick<ProgressContextValue, 'progress' | 'review'>

/** Progress is a convenience layer: a failed fetch yields undefined and the previous value is kept. */
async function load(): Promise<{ [K in keyof Snapshot]?: NonNullable<Snapshot[K]> }> {
  const [p, r] = await Promise.allSettled([fetchProgress(), fetchReviewQueue()])
  return {
    progress: p.status === 'fulfilled' ? p.value : undefined,
    review:
      r.status === 'fulfilled'
        ? { due_count: r.value.due_count, upcoming_count: r.value.upcoming_count, next_due_at: r.value.next_due_at }
        : undefined,
  }
}

/** Holds the app-wide progress summary and review-due count (sidebar badges, dashboard). */
export default function ProgressProvider({ children }: { children: ReactNode }) {
  const [snap, setSnap] = useState<Snapshot>({ progress: null, review: null })

  const apply = useCallback((next: Awaited<ReturnType<typeof load>>) => {
    setSnap((prev) => ({ progress: next.progress ?? prev.progress, review: next.review ?? prev.review }))
  }, [])

  const refresh = useCallback(async () => apply(await load()), [apply])

  useEffect(() => {
    let cancelled = false
    void load().then((next) => !cancelled && apply(next))
    return () => {
      cancelled = true
    }
  }, [apply])

  const value = useMemo(() => ({ ...snap, refresh }), [snap, refresh])
  return <ProgressContext.Provider value={value}>{children}</ProgressContext.Provider>
}
