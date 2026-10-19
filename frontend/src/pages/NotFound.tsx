import { Link } from 'react-router-dom'

export default function NotFound() {
  return (
    <div>
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <Link to="/" className="mt-4 inline-block text-sky-600 hover:underline dark:text-sky-400">
        Back to Home
      </Link>
    </div>
  )
}
