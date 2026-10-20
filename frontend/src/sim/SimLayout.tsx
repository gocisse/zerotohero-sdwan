import { NavLink, Outlet } from 'react-router-dom'

const nav = (active: boolean) =>
  `rounded-md px-3 py-1.5 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
    active
      ? 'bg-sky-100 font-medium text-sky-900 dark:bg-sky-900/40 dark:text-sky-100'
      : 'text-slate-600 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800'
  }`

/** Wraps every /sim screen with the always-visible "teaching model" banner. */
export default function SimLayout() {
  return (
    <div className="max-w-5xl">
      <div
        role="note"
        data-sim-banner
        className="mb-4 rounded-md border border-orange-300 bg-orange-50 px-3 py-2 text-sm font-medium text-orange-950 dark:border-orange-700 dark:bg-orange-950/40 dark:text-orange-100"
      >
        Teaching model, not an emulator. Items marked 'assumption' are not verified.
      </div>
      <nav aria-label="Simulator" className="mb-3 flex gap-1 text-sm">
        <NavLink to="/sim" end className={({ isActive }) => nav(isActive)}>
          Topologies
        </NavLink>
        <NavLink to="/sim/labs" className={({ isActive }) => nav(isActive)} data-nav-labs>
          Labs
        </NavLink>
      </nav>
      <Outlet />
    </div>
  )
}
