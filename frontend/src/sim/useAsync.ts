import { useEffect, useState } from 'react'

export type Async<T> = { status: 'loading' } | { status: 'ready'; data: T } | { status: 'error'; error: Error }

/** Runs `load` whenever `key` changes and tracks loading, ready and error states. */
export function useAsync<T>(load: () => Promise<T>, key: string): Async<T> {
  const [state, setState] = useState<{ key: string; value: Async<T> }>({ key, value: { status: 'loading' } })
  useEffect(() => {
    let cancelled = false
    load()
      .then((data) => !cancelled && setState({ key, value: { status: 'ready', data } }))
      .catch((error: Error) => !cancelled && setState({ key, value: { status: 'error', error } }))
    return () => {
      cancelled = true
    }
    // `load` is recreated on every render; `key` identifies what is being loaded.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])
  return state.key === key ? state.value : { status: 'loading' }
}
