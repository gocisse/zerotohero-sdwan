import { Outlet } from 'react-router-dom'

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
      <Outlet />
    </div>
  )
}
