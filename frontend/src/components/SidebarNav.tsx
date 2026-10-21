import { useMemo, useState } from 'react'
import { NavLink } from 'react-router-dom'
import type { LessonStatus, SectionSummary } from '../api'
import { useCourseState } from '../course/context'
import { pct } from '../lib/course'
import { useProgress } from '../progress/context'
import { useStudyGuide } from '../studyguide/context'
import { sectionLessons } from '../studyguide/derive'

type NavOrder = 'course' | 'guide'
const ORDER_KEY = 'vca.sidebar-order'

/** The remembered sidebar order. localStorage can be unavailable (private window), so every access is guarded. */
function readOrder(): NavOrder {
  try {
    return localStorage.getItem(ORDER_KEY) === 'guide' ? 'guide' : 'course'
  } catch {
    return 'course'
  }
}

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

/** Lessons grouped by study guide section, in guide order. Only the grouping changes; the course order is untouched. */
function GuideOrderNav({
  collapsed,
  toggle,
  onNavigate,
}: {
  collapsed: Set<string>
  toggle: (id: string) => void
  onNavigate?: () => void
}) {
  const sg = useStudyGuide()
  const course = useCourseState()
  const { progress } = useProgress()
  const statusOf = (id: string): LessonStatus =>
    progress?.lessons.find((l) => l.lesson_id === id)?.status ?? 'not_started'

  const groups = useMemo(() => {
    if (sg.status !== 'ready' || course.status !== 'ready') return []
    const sectionOf = new Map(course.course.sections.flatMap((s) => s.lessons.map((l) => [l.id, s.id] as const)))
    const seen = new Set<string>()
    const take = (l: { id: string; title: string }) => {
      if (seen.has(l.id) || !sectionOf.has(l.id)) return null
      seen.add(l.id)
      return { id: l.id, title: l.title, sectionId: sectionOf.get(l.id)! }
    }
    const out = sg.guide.sections.map((s) => ({
      key: s.id,
      title: `${s.id} · ${s.title}`,
      empty: s.topics.every((t) => t.coverage === 'none'),
      lessons: sectionLessons(s).flatMap((l) => take(l) ?? []),
    }))
    const rest = course.course.sections.flatMap((s) => s.lessons).flatMap((l) => take(l) ?? [])
    out.push({ key: 'beyond', title: sg.guide.beyond.title, empty: false, lessons: rest })
    return out
  }, [sg, course])

  if (sg.status !== 'ready' || course.status !== 'ready') return null
  return (
    <div data-sidebar-order="guide">
      {groups.map((g) => (
        <div key={g.key} data-guide-nav={g.key}>
          <button
            type="button"
            onClick={() => toggle(`guide:${g.key}`)}
            aria-expanded={!collapsed.has(`guide:${g.key}`)}
            className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-200 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:text-slate-200 dark:hover:bg-slate-800"
          >
            <span
              aria-hidden
              className={`text-xs transition-transform ${collapsed.has(`guide:${g.key}`) ? '' : 'rotate-90'}`}
            >
              ▶
            </span>
            <span className="flex-1">{g.title}</span>
          </button>
          {!collapsed.has(`guide:${g.key}`) && (
            <ul className="mt-1 mb-2 ml-4 space-y-0.5 border-l border-slate-200 pl-2 dark:border-slate-800">
              {g.lessons.map((l) => (
                <li key={l.id}>
                  <NavLink
                    to={`/course/${l.sectionId}/${l.id}`}
                    onClick={onNavigate}
                    className={({ isActive }) => `${navLink(isActive)} flex items-start gap-2`}
                  >
                    <StatusIcon status={statusOf(l.id)} />
                    <span>{l.title}</span>
                  </NavLink>
                </li>
              ))}
              {g.lessons.length === 0 && (
                <li className="px-3 py-1.5 text-sm text-slate-500 dark:text-slate-400">
                  {g.empty ? 'Not covered in this course yet' : 'No lessons'}
                </li>
              )}
            </ul>
          )}
        </div>
      ))}
      <div data-guide-nav="quizzes">
        <button
          type="button"
          onClick={() => toggle('guide:quizzes')}
          aria-expanded={!collapsed.has('guide:quizzes')}
          className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-200 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none dark:text-slate-200 dark:hover:bg-slate-800"
        >
          <span
            aria-hidden
            className={`text-xs transition-transform ${collapsed.has('guide:quizzes') ? '' : 'rotate-90'}`}
          >
            ▶
          </span>
          <span className="flex-1">Section quizzes</span>
        </button>
        {!collapsed.has('guide:quizzes') && (
          <ul className="mt-1 mb-2 ml-4 space-y-0.5 border-l border-slate-200 pl-2 dark:border-slate-800">
            {course.course.sections
              .filter((s) => s.question_count > 0)
              .map((s) => (
                <li key={s.id}>
                  <NavLink
                    to={`/course/${s.id}/quiz`}
                    onClick={onNavigate}
                    className={({ isActive }) => navLink(isActive)}
                  >
                    {s.title}
                  </NavLink>
                </li>
              ))}
          </ul>
        )}
      </div>
    </div>
  )
}

/** Course navigation shared by the desktop sidebar and the mobile drawer. */
export default function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const state = useCourseState()
  const { review } = useProgress()
  const sg = useStudyGuide()
  const [order, setOrder] = useState<NavOrder>(readOrder)
  const chooseOrder = (o: NavOrder) => {
    setOrder(o)
    try {
      localStorage.setItem(ORDER_KEY, o)
    } catch {
      // The choice still applies for this visit.
    }
  }
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
      <NavLink to="/troubleshooting" onClick={onNavigate} className={({ isActive }) => navLink(isActive)}>
        Troubleshooting
      </NavLink>
      <NavLink to="/sim" onClick={onNavigate} className={({ isActive }) => navLink(isActive)}>
        Simulator
      </NavLink>
      <div className="my-2 border-t border-slate-200 dark:border-slate-800" />
      {state.status === 'loading' && <p className="px-3 py-2 text-sm text-slate-500">Loading course…</p>}
      {state.status === 'error' && (
        <p className="px-3 py-2 text-sm text-red-600 dark:text-red-400">Course unavailable: {state.error.message}</p>
      )}
      {sg.status === 'ready' && (
        <div
          role="group"
          aria-label="Lesson order"
          className="mb-1 flex overflow-hidden rounded-md border border-slate-300 text-xs dark:border-slate-700"
        >
          {(
            [
              ['course', 'Course order'],
              ['guide', 'Study guide order'],
            ] as const
          ).map(([o, label]) => (
            <button
              key={o}
              type="button"
              aria-pressed={order === o}
              onClick={() => chooseOrder(o)}
              data-order-toggle={o}
              className={`flex-1 px-2 py-1.5 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:outline-none ${
                order === o ? 'bg-sky-600 text-white dark:bg-sky-500' : 'hover:bg-slate-200 dark:hover:bg-slate-800'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      )}
      {state.status === 'ready' && order === 'guide' && sg.status === 'ready' && (
        <GuideOrderNav collapsed={collapsed} toggle={toggle} onNavigate={onNavigate} />
      )}
      {state.status === 'ready' &&
        (order === 'course' || sg.status !== 'ready') &&
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
