import { Link } from 'react-router-dom'

/**
 * Shown at the end of the last course section. The capstone is practice that ties the sections together; it is never
 * required, and nothing in the course is locked behind it.
 */
export default function CapstoneCard() {
  return (
    <section
      aria-labelledby="capstone-card"
      className="mt-10 rounded-xl border-2 border-violet-300 bg-violet-50 p-6 dark:border-violet-800 dark:bg-violet-950/30"
      data-capstone-card
    >
      <h2 id="capstone-card" className="text-xl font-semibold">
        Capstone (optional)
      </h2>
      <p className="mt-1 text-slate-700 dark:text-slate-300">
        Put the whole course to work on one practice network: three connected scenarios (a Monday-morning ticket, a Hub
        incident and a change request) that you diagnose with the flow tracer and the mock screens. Afterwards, an
        ungraded self-check shows what you can explain and what to revisit. Neither is required.
      </p>
      <div className="mt-4 flex flex-wrap gap-3">
        <Link
          to="/sim/labs#capstone"
          className="rounded-md bg-violet-700 px-5 py-2.5 text-sm font-medium text-white hover:bg-violet-800 focus-visible:ring-2 focus-visible:ring-violet-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950"
          data-capstone-open
        >
          Open the capstone
        </Link>
        <Link
          to="/sim/self-check"
          className="rounded-md border border-slate-400 px-5 py-2.5 text-sm hover:border-violet-500 focus-visible:ring-2 focus-visible:ring-violet-500 focus-visible:outline-none dark:border-slate-600"
          data-self-check-open
        >
          Take the self-check
        </Link>
      </div>
    </section>
  )
}
