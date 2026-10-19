import { useEffect, useRef, useState } from 'react'
import { Link, Outlet } from 'react-router-dom'
import { useCourseState } from '../course/context'
import SidebarNav from './SidebarNav'

export default function Layout() {
  const state = useCourseState()
  const [menuOpen, setMenuOpen] = useState(false)
  const closeRef = useRef<HTMLButtonElement>(null)

  // Mobile drawer: Escape closes, and focus moves into the drawer when it opens.
  useEffect(() => {
    if (!menuOpen) return
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMenuOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [menuOpen])

  return (
    <div className="flex h-dvh overflow-hidden">
      <aside className="hidden w-72 shrink-0 flex-col overflow-y-auto border-r border-slate-200 bg-slate-50 md:flex dark:border-slate-800 dark:bg-slate-900">
        <Link to="/" className="px-5 py-4 text-lg font-semibold">
          VeloCloud Academy
        </Link>
        <SidebarNav />
      </aside>

      {menuOpen && (
        <div className="fixed inset-0 z-40 md:hidden" role="dialog" aria-modal="true" aria-label="Course menu">
          <button
            type="button"
            aria-label="Close menu"
            tabIndex={-1}
            className="absolute inset-0 bg-black/50"
            onClick={() => setMenuOpen(false)}
          />
          <div className="absolute inset-y-0 left-0 flex w-80 max-w-[85vw] flex-col overflow-y-auto bg-slate-50 shadow-xl dark:bg-slate-900">
            <div className="flex items-center justify-between px-5 py-4">
              <span className="text-lg font-semibold">VeloCloud Academy</span>
              <button
                ref={closeRef}
                type="button"
                onClick={() => setMenuOpen(false)}
                className="rounded-md px-2 py-1 text-sm hover:bg-slate-200 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:hover:bg-slate-800"
              >
                Close ✕
              </button>
            </div>
            <SidebarNav onNavigate={() => setMenuOpen(false)} />
          </div>
        </div>
      )}

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center gap-3 border-b border-slate-200 px-4 md:px-6 dark:border-slate-800">
          <button
            type="button"
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-sky-400 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none md:hidden dark:border-slate-700"
            aria-label="Open menu"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen(true)}
          >
            ☰ Menu
          </button>
          <span className="min-w-0 flex-1 truncate text-sm text-slate-500 dark:text-slate-400">
            {state.status === 'ready' ? state.course.title : 'VeloCloud Academy'}
          </span>
          <span className="hidden text-xs text-slate-400 sm:inline dark:text-slate-500">local profile</span>
        </header>
        <main className="flex-1 overflow-y-auto p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
