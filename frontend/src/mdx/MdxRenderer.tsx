import { evaluate } from '@mdx-js/mdx'
import 'highlight.js/styles/github-dark.css'
import { Component, useEffect, useState, type ComponentType, type ErrorInfo, type ReactNode } from 'react'
import * as runtime from 'react/jsx-runtime'
import rehypeHighlight from 'rehype-highlight'
import remarkGfm from 'remark-gfm'
import ErrorPanel from '../components/ErrorPanel'
import { mdxComponents } from './registry'

type Compiled =
  | { kind: 'pending' }
  | { kind: 'ok'; Content: ComponentType<{ components?: typeof mdxComponents }> }
  | { kind: 'error'; message: string }

/** Compile MDX source in the browser and render it with the component registry. Mount with `key={source}`. */
export default function MdxRenderer({ source, onRendered }: { source: string; onRendered?: () => void }) {
  const [compiled, setCompiled] = useState<Compiled>({ kind: 'pending' })

  useEffect(() => {
    let cancelled = false
    evaluate(source, {
      Fragment: runtime.Fragment,
      jsx: runtime.jsx,
      jsxs: runtime.jsxs,
      remarkPlugins: [remarkGfm],
      rehypePlugins: [rehypeHighlight],
    })
      .then((mod) => {
        if (cancelled) return
        setCompiled({ kind: 'ok', Content: mod.default })
        onRendered?.() // batched with the state update above, so listeners see the rendered content
      })
      .catch((err: unknown) => !cancelled && setCompiled({ kind: 'error', message: describe(err) }))
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source])

  if (compiled.kind === 'pending') return <p className="text-slate-500">Rendering…</p>
  if (compiled.kind === 'error') {
    return (
      <ErrorPanel title="This lesson could not be compiled">
        <pre className="mt-2 overflow-x-auto text-sm whitespace-pre-wrap">{compiled.message}</pre>
      </ErrorPanel>
    )
  }
  const { Content } = compiled
  return (
    <RenderBoundary>
      <Content components={mdxComponents} />
    </RenderBoundary>
  )
}

/** vfile messages carry line/column; surface them when present. */
function describe(err: unknown): string {
  if (err instanceof Error) {
    const pos = (err as { line?: number; column?: number }).line
      ? ` (line ${(err as { line?: number }).line}, column ${(err as { column?: number }).column})`
      : ''
    return err.message + pos
  }
  return String(err)
}

/** Catches errors thrown while rendering compiled MDX (e.g. an undefined component). */
class RenderBoundary extends Component<{ children: ReactNode }, { error?: Error }> {
  state: { error?: Error } = {}

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('MDX render error', error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <ErrorPanel title="This lesson failed to render">
          <pre className="mt-2 overflow-x-auto text-sm whitespace-pre-wrap">{this.state.error.message}</pre>
        </ErrorPanel>
      )
    }
    return this.props.children
  }
}
