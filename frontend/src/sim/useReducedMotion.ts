import { useEffect, useState } from 'react'

/** True when the user asks for reduced motion (prefers-reduced-motion). */
export function useReducedMotion(): boolean {
  const query = '(prefers-reduced-motion: reduce)'
  const [reduced, setReduced] = useState(
    () => (typeof window !== 'undefined' && window.matchMedia?.(query).matches) || false,
  )
  useEffect(() => {
    const mq = window.matchMedia?.(query)
    if (!mq) return
    const on = () => setReduced(mq.matches)
    mq.addEventListener('change', on)
    return () => mq.removeEventListener('change', on)
  }, [])
  return reduced
}
