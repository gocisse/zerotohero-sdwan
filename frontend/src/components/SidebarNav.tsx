import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import type { LessonStatus, SectionSummary } from '../api'
import { useCourseState } from '../course/context'
import { pct } from '../lib/course'
import { useProgress } from '../progress/context'

const navLink = (active: boolean) =>
  `block rounded-md px-3 py-1.5 text-sm focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
    active
      ? 'bg-sky-100 font-medium text-sky-900 dark:bg-sky-900/40 dark:text-sky-100'
      : 'text-slate-600 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800'
  }`

const statusIcon: Record<LessonStatus, { icon: string; label: string; cls: string }> = {
  not_started: { icon: '○', label: 'Not started', cls: 'text-slate-400 dark:text-slate-500' },
  visited: { icon: '◐', label: 'In progress', cls: 'text-sky-600 dark:text-sky-400' },
  completed: { icon: '✓', label: 'Completed', cls: 'text-emerald-600 dark:text-emerald-400' },
}

function StatusIcon({ status }: { status: LessonStatus }) {
  const s = statusIcon[status]
  return (
    <span className={`w-4 shrink-0 text-center ${s.cls}`} data-status={status}>
      <span aria-hidden>{s.icon}</span>
      <span className="sr-only">{s.label}</span>
    </span>
  )
}

function SectionNav({
  section,
  collapsed,
  onToggle,
  onNavigate,
}: {
  section: SectionSummary
  collapsed: boolean
  onToggle: () => void
  onNavigate?: () => void
}) {
  const { progress } = useProgress()
  const statusOf = (id: string): LessonStatus =>
    progress?.lessons.find((l) => l.lesson_id === id)?.status ?? 'not_started'
  const sp = progress?.sections.find((s) => s.section_id === section.id)

  return (
    <div>
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={!collapsed}
        className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-200 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:text-slate-200 dark:hover:bg-slate-800"
      >
        <span aria-hidden className={`text-xs transition-transform ${collapsed ? '' : 'rotate-90'}`}>
          ▶
        </span>
        <span className="flex-1">{section.title}</span>
        {sp?.best_score_pct != null && (
          <span
            className="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200"
            title="Best quiz score"
            data-best-score
          >
            {pct(sp.best_score_pct)}
          </span>
        )}
      </button>
      {!collapsed && (
        <ul className="mt-1 mb-2 ml-4 space-y-0.5 border-l border-slate-200 pl-2 dark:border-slate-800">
          <li>
            <NavLink
              to={`/course/${section.id}`}
              end
              onClick={onNavigate}
              className={({ isActive }) => navLink(isActive)}
            >
              Overview
            </NavLink>
          </li>
          {section.lessons.map((l) => (
            <li key={l.id}>
              <NavLink
                to={`/course/${section.id}/${l.id}`}
                onClick={onNavigate}
                className={({ isActive }) => `${navLink(isActive)} flex items-start gap-2`}
              >
                <StatusIcon status={statusOf(l.id)} />
                <span>{l.title}</span>
              </NavLink>
            </li>
          ))}
          {section.question_count > 0 && (
            <li>
              <NavLink
                to={`/course/${section.id}/quiz`}
                onClick={onNavigate}
                className={({ isActive }) => navLink(isActive)}
              >
                Section quiz
                {sp && sp.quiz_attempts === 0 && sp.quiz_skipped && (
                  <span className="ml-2 text-xs text-slate-400 dark:text-slate-500">skipped</span>
                )}
              </NavLink>
            </li>
          )}
        </ul>
      )}
    </div>
  )
}

/** Course navigation shared by the desktop sidebar and the mobile drawer. */
export default function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const state = useCourseState()
  const { review } = useProgress()
  // Sections are expanded by default; the set holds the ones the user collapsed.
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())
  const toggle = (id: string) =>
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (!next.delete(id)) next.add(id)
      return next
    })

  return (
    <nav aria-label="Course" className="flex flex-col gap-1 px-3 pb-6">
      <NavLink to="/" end onClick={onNavigate} className={({ isActive }) => navLink(isActive)}>
        Home
      </NavLink>
      <NavLink to="/progress" onClick={onNavigate} className={({ isActive }) => navLink(isActive)}>
        Progress
      </NavLink>
      <NavLink
        to="/review"
        onClick={onNavigate}
        className={({ isActive }) => `${navLink(isActive)} flex items-center justify-between`}
      >
        <span>Review</span>
        {review && review.due_count > 0 && (
          <span
            className="rounded-full bg-amber-500 px-2 py-0.5 text-xs font-semibold text-white"
            data-review-badge
            aria-label={`${review.due_count} questions due`}
          >
            {review.due_count}
          </span>
        )}
      </NavLink>
      <NavLink to="/glossary" onClick={onNavigate} className={({ isActive }) => navLink(isActive)}>
        Glossary
      </NavLink>
      <div className="my-2 border-t border-slate-200 dark:border-slate-800" />
      {state.status === 'loading' && <p className="px-3 py-2 text-sm text-slate-500">Loading course…</p>}
      {state.status === 'error' && (
        <p className="px-3 py-2 text-sm text-red-600 dark:text-red-400">Course unavailable: {state.error.message}</p>
      )}
      {state.status === 'ready' &&
        state.course.sections.map((s) => (
          <SectionNav
            key={s.id}
            section={s}
            collapsed={collapsed.has(s.id)}
            onToggle={() => toggle(s.id)}
            onNavigate={onNavigate}
          />
        ))}
    </nav>
  )
}
