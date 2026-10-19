import { createContext, useContext } from 'react'
import type { GlossaryTerm } from '../api'

export type GlossaryState =
  | { status: 'loading' }
  | { status: 'ready'; terms: GlossaryTerm[]; byId: Map<string, GlossaryTerm> }
  | { status: 'error'; error: Error }

export const GlossaryContext = createContext<GlossaryState>({ status: 'loading' })

export const useGlossary = () => useContext(GlossaryContext)
