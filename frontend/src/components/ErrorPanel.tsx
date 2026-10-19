import { ApiError } from '../api'

interface Props {
  title: string
  error?: Error
  children?: React.ReactNode
}

/** Shows an error prominently; lists every content-validation problem when the backend sent them. */
export default function ErrorPanel({ title, error, children }: Props) {
  const details = error instanceof ApiError ? error.details : []
  return (
    <div
      role="alert"
      className="rounded-lg border border-red-300 bg-red-50 p-4 text-red-900 dark:border-red-800 dark:bg-red-950/40 dark:text-red-100"
    >
      <h2 className="font-semibold">{title}</h2>
      {error && <p className="mt-1 text-sm whitespace-pre-wrap">{error.message}</p>}
      {details.length > 0 && (
        <ul className="mt-3 space-y-2 text-sm">
          {details.map((d, i) => (
            <li key={i}>
              <code className="font-mono font-semibold">{d.file}</code>
              <div>{d.message}</div>
            </li>
          ))}
        </ul>
      )}
      {children}
    </div>
  )
}
