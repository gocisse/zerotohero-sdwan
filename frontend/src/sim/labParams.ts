/** Appends the lab query (?lab=&fixes=) to a simulator path, using ? or & as needed. */
export function withLab(path: string, labSearch: string): string {
  if (!labSearch) return path
  return `${path}${path.includes('?') ? '&' : '?'}${labSearch.slice(1)}`
}

/** New search params that keep the lab and its fixes while changing other keys. */
export function keepLab(current: URLSearchParams, next: Record<string, string>): URLSearchParams {
  const out = new URLSearchParams(next)
  for (const k of ['lab', 'fixes']) {
    const v = current.get(k)
    if (v) out.set(k, v)
  }
  return out
}
