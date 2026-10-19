/** One problem reported by the backend's content validator. */
export interface ContentProblem {
  file: string
  message: string
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: string
  readonly details: ContentProblem[]

  constructor(status: number, message: string, code?: string, details: ContentProblem[] = []) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

interface ErrorBody {
  error?: string
  message?: string
  details?: ContentProblem[]
}

async function request<T>(method: string, path: string, body: unknown, okStatuses: number[]): Promise<T> {
  const res = await fetch(path, {
    method,
    headers:
      body === undefined
        ? { Accept: 'application/json' }
        : { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok && !okStatuses.includes(res.status)) {
    let parsed: ErrorBody = {}
    try {
      parsed = (await res.json()) as ErrorBody
    } catch {
      // Non-JSON error body (e.g. a proxy error page); fall back to the status line.
    }
    throw new ApiError(res.status, parsed.message ?? `${res.status} ${res.statusText}`, parsed.error, parsed.details)
  }
  return (await res.json()) as T
}

/** GET JSON from the backend. Non-2xx responses throw an ApiError, except those listed in `okStatuses`. */
export const getJSON = <T>(path: string, okStatuses: number[] = []) => request<T>('GET', path, undefined, okStatuses)

/** Send a JSON body (POST/PUT) and parse the JSON response. */
export const sendJSON = <T>(method: 'POST' | 'PUT', path: string, body: unknown = {}) =>
  request<T>(method, path, body, [])
