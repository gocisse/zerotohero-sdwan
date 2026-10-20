import { Link } from 'react-router-dom'
import { fetchLabs, type LabSummary } from '../api'
import ErrorPanel from '../components/ErrorPanel'
import { LessonLink } from './LessonLink'
import { LAB_STATUS_STYLE, LAB_STATUS_TEXT, difficultyText } from './labText'
import { useAsync } from './useAsync'

/** /sim/labs: practice scenarios. Nothing is required and nothing is locked. */
export default function LabsPage() {
  const labs = useAsync(() => fetchLabs(), 'labs')
  if (labs.status === 'loading') return <p className="text-slate-500">Loading…</p>
  if (labs.status === 'error') return <ErrorPanel title="Labs are unavailable" error={labs.error} />
  const capstone = labs.data.filter((l) => l.id.startsWith('capstone-'))
  return (
    <div data-labs-page>
      <p className="text-sm text-slate-500 dark:text-slate-400">
        <Link to="/sim" className="underline">
          Simulator
        </Link>{' '}
        / Labs
      </p>
      <h1 className="mt-1 text-2xl font-semibold">Labs</h1>
      <p className="mt-1 max-w-3xl text-sm text-slate-600 dark:text-slate-300">
        Practice diagnosing a broken teaching network from the simulator's mock screens, then fix it. Labs are never
        required, nothing is locked, and a wrong answer only shows feedback. No command-line knowledge is needed.
      </p>
      <ul className="mt-4 grid gap-4 md:grid-cols-2">
        {labs.data
          .filter((l) => !l.id.startsWith('capstone-'))
          .map((l) => (
            <LabCard key={l.id} l={l} />
          ))}
      </ul>
      {capstone.length > 0 && (
        <section
          id="capstone"
          className="mt-8 scroll-mt-4 rounded-xl border-2 border-violet-300 p-4 dark:border-violet-800"
          data-capstone-section
        >
          <h2 className="text-xl font-semibold">Capstone</h2>
          <p className="mt-1 max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            Three connected scenarios on one practice network (a data center with two Hubs and three branches running
            OSPF, BGP and a static route). Each is a multi-part scenario, so work through it in a sensible order with
            the flow tracer and the mock screens. Each ends with a debrief that says where you would look in the real
            VCO. Never required; when you are done, try the{' '}
            <Link to="/sim/self-check" className="underline" data-self-check-link>
              self-check
            </Link>
            .
          </p>
          <ul className="mt-3 grid gap-4 md:grid-cols-2">
            {capstone.map((l) => (
              <LabCard key={l.id} l={l} />
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}

function LabCard({ l }: { l: LabSummary }) {
  return (
    <li className="rounded-lg border border-slate-200 p-4 dark:border-slate-800" data-lab-card={l.id}>
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-base font-semibold">
          <Link to={`/sim/labs/${l.id}`} className="underline" data-lab-open={l.id}>
            {l.title}
          </Link>
        </h2>
        <span
          className={`rounded-full border px-2 py-0.5 text-xs ${LAB_STATUS_STYLE[l.progress.status]}`}
          data-lab-status={l.progress.status}
        >
          {LAB_STATUS_TEXT[l.progress.status]}
        </span>
      </div>
      <p className="mt-1 text-sm text-slate-700 dark:text-slate-200">{l.summary}</p>
      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
        Difficulty: {difficultyText(l.difficulty)} ({l.difficulty} of 3) · about {l.estimated_minutes} min · {l.tasks}{' '}
        tasks
      </p>
      <p className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs">
        {l.goals.map((g) => (
          <LessonLink key={g.id} id={g.id} prefix="Lesson" />
        ))}
      </p>
    </li>
  )
}
