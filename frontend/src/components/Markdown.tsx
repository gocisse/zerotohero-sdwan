import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props {
  children: string
  className?: string
  /** Render as a span with no paragraph wrappers, so it can sit inside buttons and sentences. */
  inline?: boolean
}

/** Renders authored Markdown (quiz prompts, explanations, narration, glossary entries). */
export default function Markdown({ children, className = '', inline }: Props) {
  if (inline) {
    return (
      <span className={className}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={{ p: ({ children: c }) => <>{c}</> }}>
          {children}
        </ReactMarkdown>
      </span>
    )
  }
  return (
    <div
      className={`prose prose-slate max-w-none dark:prose-invert prose-p:my-1 prose-code:before:content-none prose-code:after:content-none ${className}`}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{children}</ReactMarkdown>
    </div>
  )
}
