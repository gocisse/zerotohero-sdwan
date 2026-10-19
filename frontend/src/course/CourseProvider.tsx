import { useEffect, useState, type ReactNode } from 'react'
import { fetchCourse } from '../api'
import { CourseContext, type CourseState } from './context'

export default function CourseProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<CourseState>({ status: 'loading' })

  useEffect(() => {
    let cancelled = false
    fetchCourse()
      .then((course) => !cancelled && setState({ status: 'ready', course }))
      .catch((error: Error) => !cancelled && setState({ status: 'error', error }))
    return () => {
      cancelled = true
    }
  }, [])

  return <CourseContext.Provider value={state}>{children}</CourseContext.Provider>
}
