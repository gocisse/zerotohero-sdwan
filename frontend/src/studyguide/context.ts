import { createContext, useContext } from 'react'
import type { StudyGuide } from '../api'

export type StudyGuideState =
  | { status: 'loading' }
  | { status: 'ready'; guide: StudyGuide; refresh: () => Promise<void> }
  | { status: 'unavailable'; refresh: () => Promise<void> }

export const StudyGuideContext = createContext<StudyGuideState>({ status: 'loading' })

export const useStudyGuide = () => useContext(StudyGuideContext)
