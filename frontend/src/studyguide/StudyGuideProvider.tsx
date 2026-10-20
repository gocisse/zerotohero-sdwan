import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { fetchStudyGuide } from '../api'
import { StudyGuideContext } from './context'
import type { StudyGuide } from '../api'

type Loaded = { guide: StudyGuide } | { guide: null } | null

/**
 * Fetches the study guide (structure plus progress derived from stored data). The guide is a navigation layer: if it
 * cannot be loaded the rest of the app keeps working and the screens that use it say so.
 */
export default function StudyGuideProvider({ children }: { children: ReactNode }) {
  const [loaded, setLoaded] = useState<Loaded>(null)

  const refresh = useCallback(async () => {
    try {
      setLoaded({ guide: await fetchStudyGuide() })
    } catch {
      setLoaded((prev) => prev ?? { guide: null })
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    fetchStudyGuide()
      .then((guide) => !cancelled && setLoaded({ guide }))
      .catch(() => !cancelled && setLoaded({ guide: null }))
    return () => {
      cancelled = true
    }
  }, [])

  const value =
    loaded === null
      ? { status: 'loading' as const }
      : loaded.guide
        ? { status: 'ready' as const, guide: loaded.guide, refresh }
        : { status: 'unavailable' as const, refresh }
  return <StudyGuideContext.Provider value={value}>{children}</StudyGuideContext.Provider>
}
