import { useEffect, useRef, useState } from 'react'
import { postTunnels, type Conditions, type TunnelSet } from '../api'

export interface TunnelsState {
  data: TunnelSet | null
  error: Error | null
  loading: boolean
}

/** Fetches the tunnels for the conditions; keeps the previous answer on screen while the next one loads. */
export function useTunnels(topologyId: string, conditions: Conditions): TunnelsState {
  const key = `${topologyId}/${JSON.stringify(conditions)}`
  const [state, setState] = useState<{ key: string; data: TunnelSet | null; error: Error | null }>({
    key: '',
    data: null,
    error: null,
  })
  const last = useRef<TunnelSet | null>(null)
  useEffect(() => {
    let cancelled = false
    postTunnels(topologyId, conditions)
      .then((data) => {
        last.current = data
        if (!cancelled) setState({ key, data, error: null })
      })
      .catch((error: Error) => !cancelled && setState({ key, data: last.current, error }))
    return () => {
      cancelled = true
    }
    // The key encodes the topology and the conditions.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])
  return { data: state.data ?? last.current, error: state.key === key ? state.error : null, loading: state.key !== key }
}
