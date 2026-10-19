import { useEffect, useState, type ReactNode } from 'react'
import { fetchGlossary } from '../api'
import { GlossaryContext, type GlossaryState } from './context'

/** Fetches the glossary once and caches it for every <Term> and the glossary page. */
export default function GlossaryProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<GlossaryState>({ status: 'loading' })

  useEffect(() => {
    let cancelled = false
    fetchGlossary()
      .then(
        ({ terms }) => !cancelled && setState({ status: 'ready', terms, byId: new Map(terms.map((t) => [t.id, t])) }),
      )
      .catch((error: Error) => !cancelled && setState({ status: 'error', error }))
    return () => {
      cancelled = true
    }
  }, [])

  return <GlossaryContext.Provider value={state}>{children}</GlossaryContext.Provider>
}
