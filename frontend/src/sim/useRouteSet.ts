import { useEffect, useState } from 'react'
import { postRoutes, type RouteSet } from '../api'
import { useTopologyCtx } from './topologyContext'

/** Computes the route tables for the current conditions and assumptions; keeps the previous result while reloading. */
export function useRouteSet(): { data: RouteSet | null; error: Error | null; busy: boolean } {
  const { topologyId, conditions, assumptions } = useTopologyCtx()
  const key = JSON.stringify([topologyId, conditions, assumptions])
  const [state, setState] = useState<{ key: string; data: RouteSet | null; error: Error | null }>({
    key: '',
    data: null,
    error: null,
  })
  useEffect(() => {
    let cancelled = false
    postRoutes(topologyId, conditions, assumptions)
      .then((data) => !cancelled && setState({ key, data, error: null }))
      .catch((error: Error) => !cancelled && setState((s) => ({ ...s, key, error })))
    return () => {
      cancelled = true
    }
    // `key` stands for the three inputs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])
  return { data: state.data, error: state.error, busy: state.key !== key }
}
