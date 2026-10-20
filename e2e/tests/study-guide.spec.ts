import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { getJSON, getQuiz, openLesson, type AttemptResult, type LessonRef } from './support'

// Study guide, glossary by guide and the study dashboard (step 16). Everything is read from the API: no topic, term or number
// is hard-coded, so editing content/study-guide.yaml or the glossary does not break these tests.

interface GLesson { id: string; title: string; section_id: string; status: string }
interface GTopic {
  id: string
  title: string
  coverage: 'full' | 'partial' | 'none'
  note: string
  lessons: GLesson[]
  labs: { id: string; title: string; status: string }[]
  quiz_topics: { topic: string }[]
  progress: { lessons_completed: number; lessons_total: number; best_section_score_pct: number | null; labs_completed: number; labs_total: number }
}
interface GSection { id: string; title: string; topics: GTopic[] }
interface Guide { sections: GSection[]; beyond: { title: string; reason: string }; unmapped_lessons: GLesson[] }
interface Term { id: string; term: string; aliases: string[]; guide_topics: string[]; beyond_guide: boolean }
interface Progress {
  lessons: { lesson_id: string; status: string }[]
  lessons_completed: number
  sections: { section_id: string; best_score_pct: number | null; quiz_attempts: number }[]
  labs: { completed: number }
}
interface Activity { events: { kind: string; at: string }[]; last_lesson: { lesson_id: string; section_id: string; title: string } | null; quiz: { section_id: string; attempts: unknown[] }[]; limits: string[] }

const getGuide = (r: APIRequestContext) => getJSON<Guide>(r, '/api/study-guide')
const getTerms = async (r: APIRequestContext) => (await getJSON<{ terms: Term[] }>(r, '/api/glossary')).terms
const alpha = (a: Term, b: Term) => a.term.localeCompare(b.term, undefined, { sensitivity: 'base' })
const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
const topicsOf = (g: Guide) => g.sections.flatMap((s) => s.topics)
const noLockWords = /\b(locked|lock|unlock|streak|behind|missed|failed|overdue)\b/i
/** The text of the dashboard parts written for this step (the step 15 goal coverage says "nothing is locked", which is fine). */
const studyText = async (page: Page) =>
  (await page.locator('[data-dashboard-header], [data-guide-progress], [data-weak-topics], [data-activity]').allInnerTexts()).join('\n')

const pad = (n: number) => String(n).padStart(2, '0')
const localDate = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`

/** What the dashboard computes from the guide: a covered topic is done when all its lessons are completed. */
function expected(g: Guide) {
  const topics = topicsOf(g)
  const covered = topics.filter((t) => t.coverage !== 'none')
  const done = covered.filter((t) => t.lessons.length > 0 && t.lessons.every((l) => l.status === 'completed'))
  return { covered: covered.length, done: done.length, notCovered: topics.length - covered.length }
}

test.describe.configure({ mode: 'serial' })

test.describe('study guide API', () => {
  test('is served with every topic, none-coverage topics carry no lessons and ids resolve', async ({ request }) => {
    const g = await getGuide(request)
    expect(g.sections.length).toBeGreaterThan(0)
    const course = await getJSON<{ sections: { lessons: { id: string }[] }[] }>(request, '/api/course')
    const lessonIds = new Set(course.sections.flatMap((s) => s.lessons.map((l) => l.id)))
    const seen = new Set<string>()
    for (const t of topicsOf(g)) {
      expect(seen.has(t.id), `duplicate topic ${t.id}`).toBeFalsy()
      seen.add(t.id)
      expect(t.note.length, `${t.id} has a note`).toBeGreaterThan(0)
      if (t.coverage === 'none') expect(t.lessons, `${t.id}: none means no lessons`).toHaveLength(0)
      else expect(t.lessons.length, `${t.id}: ${t.coverage} needs a lesson`).toBeGreaterThan(0)
      for (const l of t.lessons) expect(lessonIds.has(l.id), `${t.id}: lesson ${l.id}`).toBeTruthy()
      expect(t.progress.lessons_total).toBe(t.lessons.length)
    }
    // Every course lesson is mapped or explicitly unmapped.
    const mapped = new Set([...topicsOf(g).flatMap((t) => t.lessons.map((l) => l.id)), ...g.unmapped_lessons.map((l) => l.id)])
    for (const id of lessonIds) expect(mapped.has(id), `lesson ${id} is neither mapped nor listed as unmapped`).toBeTruthy()
  })

  test('every glossary term has guide topics that exist, or is flagged beyond the guide', async ({ request }) => {
    const g = await getGuide(request)
    const ids = new Set(topicsOf(g).map((t) => t.id))
    for (const t of await getTerms(request)) {
      if (t.beyond_guide) expect(t.guide_topics, `${t.id}`).toHaveLength(0)
      else {
        expect(t.guide_topics.length, `${t.id} needs guide topics`).toBeGreaterThan(0)
        for (const id of t.guide_topics) expect(ids.has(id), `${t.id}: topic ${id}`).toBeTruthy()
      }
    }
  })
})

test.describe('glossary by study guide', () => {
  test('groups, order and counts match the API', async ({ page, request }) => {
    const [g, terms] = await Promise.all([getGuide(request), getTerms(request)])
    await page.goto('/glossary')
    await expect(page.getByRole('button', { name: 'By study guide' })).toHaveAttribute('aria-pressed', 'true')
    await page.getByRole('button', { name: 'Expand all' }).click()

    const sectionKeys = await page.locator('[data-guide-section]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.guideSection!))
    expect(sectionKeys).toEqual([...g.sections.map((s) => s.id), 'beyond'])
    const topicKeys = await page.locator('[data-guide-topic]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.guideTopic!))
    expect(topicKeys).toEqual([...topicsOf(g).map((t) => t.id), 'beyond'])

    const shownEverywhere = new Set<string>()
    const groups: [string, Term[]][] = [
      ...topicsOf(g).map((t): [string, Term[]] => [t.id, terms.filter((x) => x.guide_topics.includes(t.id)).sort(alpha)]),
      ['beyond', terms.filter((x) => x.beyond_guide).sort(alpha)],
    ]
    for (const [key, want] of groups) {
      const group = page.locator(`[data-guide-topic="${key}"]`)
      const ids = await group.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
      expect(ids, `group ${key}`).toEqual(want.map((t) => t.id))
      await expect(group.locator('[data-group-count]')).toHaveText(`${want.length} term${want.length === 1 ? '' : 's'}`)
      ids.forEach((id) => shownEverywhere.add(id))
    }
    // Stored once, shown under each topic: every term appears, and a multi-topic term appears once per topic.
    expect(shownEverywhere.size).toBe(terms.length)
    const multi = terms.find((t) => t.guide_topics.length > 1)
    if (multi) await expect(page.locator(`[data-glossary-term="${multi.id}"]`)).toHaveCount(multi.guide_topics.length)
    // Topics with no terms say so instead of hiding.
    const empty = groups.find(([, w]) => w.length === 0)
    if (empty) await expect(page.locator(`[data-guide-topic="${empty[0]}"]`)).toContainText('No glossary terms for this topic yet')
  })

  test('search expands the matching groups, hides the rest and highlights the match', async ({ page, request }) => {
    const [g, terms] = await Promise.all([getGuide(request), getTerms(request)])
    const term = terms.find((t) => t.guide_topics.length > 1)!
    await page.goto('/glossary')
    await page.getByRole('searchbox').fill(term.term)
    const q = term.term.toLowerCase()
    const hits = terms.filter((t) => [t.term, ...t.aliases].some((n) => n.toLowerCase().includes(q)))
    const expectedGroups = new Set(hits.flatMap((t) => (t.beyond_guide ? ['beyond'] : t.guide_topics)))
    const shown = await page.locator('[data-guide-topic]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.guideTopic!))
    expect(new Set(shown)).toEqual(expectedGroups)
    for (const key of expectedGroups) await expect(page.locator(`[data-guide-topic="${key}"] button[aria-expanded]`)).toHaveAttribute('aria-expanded', 'true')
    await expect(page.locator('mark').first()).toBeVisible()
    await expect(page.locator('mark').first()).toHaveText(new RegExp(term.term.slice(0, 3), 'i'))
    await expect(page.locator('[data-glossary-count]')).toContainText(`${hits.length} of ${terms.length}`)
    // Clearing the search restores the collapsed view.
    await page.getByRole('searchbox').fill('')
    await expect(page.locator(`[data-guide-topic="${topicsOf(g)[0]!.id}"] button[aria-expanded]`)).toHaveAttribute('aria-expanded', 'false')
    await page.getByRole('searchbox').fill('zzzz-no-such-term-zzzz')
    await expect(page.getByText('No terms match')).toBeVisible()
  })

  test('a deep link opens the right group (and only that one for the term)', async ({ page, request }) => {
    const terms = await getTerms(request)
    const multi = terms.find((t) => t.guide_topics.length > 1)!
    await page.goto(`/glossary#term-${multi.id}`)
    const primary = multi.guide_topics[0]!
    await expect(page.locator(`[data-guide-topic="${primary}"] button[aria-expanded]`)).toHaveAttribute('aria-expanded', 'true')
    await expect(page.locator(`#term-${multi.id}`)).toBeVisible()
    await expect(page.locator(`#term-${multi.id}`)).toBeInViewport()
    for (const other of multi.guide_topics.slice(1)) await expect(page.locator(`[data-guide-topic="${other}"] button[aria-expanded]`)).toHaveAttribute('aria-expanded', 'false')
    // A term beyond the guide opens the final group.
    const beyond = terms.find((t) => t.beyond_guide)
    if (beyond) {
      await page.goto(`/glossary#term-${beyond.id}`)
      await expect(page.locator('[data-guide-topic="beyond"] button[aria-expanded]')).toHaveAttribute('aria-expanded', 'true')
      await expect(page.locator(`#term-${beyond.id}`)).toBeVisible()
    }
  })

  test('topic chips filter the view to that topic', async ({ page, request }) => {
    const [g, terms] = await Promise.all([getGuide(request), getTerms(request)])
    const term = terms.find((t) => t.guide_topics.length > 1)!
    await page.goto(`/glossary#term-${term.id}`)
    const second = term.guide_topics[1]!
    const title = topicsOf(g).find((t) => t.id === second)!.title
    await page.locator(`#term-${term.id}`).locator('[data-term-chips]').getByRole('button', { name: title, exact: true }).click()
    await expect(page.locator('[data-topic-filter]')).toHaveAttribute('data-topic-filter', second)
    await expect(page.locator('[data-guide-topic]')).toHaveCount(1)
    await expect(page.locator(`[data-guide-topic="${second}"] button[aria-expanded]`)).toHaveAttribute('aria-expanded', 'true')
    await page.getByRole('button', { name: /Show all topics/ }).click()
    expect(await page.locator('[data-guide-topic]').count()).toBeGreaterThan(1)
  })

  test('the sticky section index lists every section and jumps to it', async ({ page, request }) => {
    const g = await getGuide(request)
    await page.goto('/glossary')
    const links = page.locator('[data-glossary-index] a')
    await expect(links).toHaveCount(g.sections.length + 1)
    await links.nth(1).click()
    await expect(page).toHaveURL(new RegExp(`#guide-${g.sections[1]!.id}$`))
    await expect(page.locator(`#guide-${g.sections[1]!.id}`)).toBeInViewport()
  })

  test('the A-Z view is still available and lists every term once', async ({ page, request }) => {
    const terms = await getTerms(request)
    await page.goto('/glossary')
    await page.getByRole('button', { name: 'A-Z' }).click()
    const ids = await page.locator('[data-glossary-term]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.glossaryTerm!))
    expect(ids).toEqual([...terms].sort(alpha).map((t) => t.id))
  })
})

/** The dashboard's API responses with all progress removed, built from the real guide, to see the empty state anywhere. */
async function mockEmpty(page: Page, request: APIRequestContext) {
  const guide = await getGuide(request)
  for (const t of topicsOf(guide)) {
    t.lessons.forEach((l) => (l.status = 'not_started'))
    t.labs.forEach((l) => (l.status = 'not_started'))
    t.progress = { ...t.progress, lessons_completed: 0, best_section_score_pct: null, labs_completed: 0 }
  }
  const progress = await getJSON<Record<string, unknown> & { lessons: { status: string }[]; sections: Record<string, unknown>[]; labs: { items: { status: string }[] } }>(request, '/api/progress')
  progress.overall_percentage = 0
  progress.lessons_completed = 0
  progress.lessons.forEach((l) => (l.status = 'not_started'))
  progress.sections = progress.sections.map((s) => ({ ...s, lessons_completed: 0, quiz_attempts: 0, best_score_pct: null, latest_score_pct: null, quiz_skipped: false }))
  progress.labs = { total: progress.labs.items.length, started: 0, completed: 0, skipped: 0, items: progress.labs.items.map((i) => ({ ...i, status: 'not_started' })) } as typeof progress.labs
  const activity = await getJSON<Activity & { since: string }>(request, '/api/activity')
  const body = (b: unknown) => ({ status: 200, contentType: 'application/json', body: JSON.stringify(b) })
  await page.route('**/api/study-guide', (r) => r.fulfill(body(guide)))
  await page.route('**/api/progress', (r) => r.fulfill(body(progress)))
  await page.route('**/api/progress/topics', (r) => r.fulfill(body({ topics: [] })))
  await page.route('**/api/review/queue', (r) => r.fulfill(body({ due_count: 0, upcoming_count: 0, next_due_at: null, session_size: 15, questions: [] })))
  await page.route('**/api/activity', (r) => r.fulfill(body({ ...activity, events: [], last_lesson: null, quiz: activity.quiz.map((q) => ({ ...q, attempts: [] })) })))
  return guide
}

async function assertEmptyDashboard(page: Page, guide: Guide) {
  const ex = expected(guide)
  await expect(page.locator('[data-continue]')).toContainText('Begin here')
  await expect(page.locator('[data-next-step]')).toContainText('Open lesson')
  await expect(page.locator('[data-overall-line]')).toHaveText(`0 of ${ex.covered} covered topics done`)
  await expect(page.locator('[data-overall-uncovered]')).toHaveText(`${ex.notCovered} topic${ex.notCovered === 1 ? '' : 's'} not yet in the course`)
  await expect(page.locator('[data-ring]')).toHaveAttribute('aria-label', `0 of ${ex.covered} covered topics done`)
  await expect(page.locator('[data-guide-section]')).toHaveCount(guide.sections.length)
  await expect(page.locator('[data-activity-empty]')).toBeVisible()
  await expect(page.locator('[data-sessions]')).toContainText('0 sessions this week')
  await expect(page.locator('[data-weak-empty]')).toBeVisible()
  await expect(page.locator('[data-quiz-best]').first()).toHaveText('—')
  expect(await studyText(page)).not.toMatch(noLockWords)
  expect(await overflow(page)).toBeLessThanOrEqual(1)
}

test.describe('dashboard on an empty database', () => {
  test('renders the friendly empty state (API responses emptied from the real guide)', async ({ page, request }) => {
    const guide = await mockEmpty(page, request)
    await page.goto('/progress')
    await expect(page.locator('[data-ring]')).toBeVisible()
    await assertEmptyDashboard(page, guide)
  })

  test('renders the empty state on a really empty database (skipped once something has been studied)', async ({ page, request }) => {
    const p = await getJSON<Progress>(request, '/api/progress')
    const a = await getJSON<Activity>(request, '/api/activity')
    const used = p.lessons_completed > 0 || a.events.length > 0 || p.sections.some((s) => s.quiz_attempts > 0)
    test.skip(used, 'the e2e database already holds progress (a KEEP_DB run); the mocked test above covers the empty state')
    await page.goto('/progress')
    await expect(page.locator('[data-ring]')).toBeVisible()
    await assertEmptyDashboard(page, await getGuide(request))
  })
})

test.describe('dashboard with progress', () => {
  test('after a lesson, a quiz and a lab, every number matches the API', async ({ page, request }) => {
    const g0 = await getGuide(request)
    const course = await getJSON<{ sections: { id: string; title: string; question_count: number; lessons: { id: string; title: string }[] }[] }>(request, '/api/course')

    // A lesson: the first lesson of the first covered topic that is not completed yet (done through the UI).
    const target = topicsOf(g0).flatMap((t) => t.lessons).find((l) => l.status !== 'completed') ?? topicsOf(g0)[0]!.lessons[0]!
    const lessonRef: LessonRef = { sectionId: target.section_id, id: target.id, title: target.title, placeholder: false }
    await openLesson(page, lessonRef)
    await page.getByRole('button', { name: 'Mark as complete' }).click()
    await expect.poll(async () => (await getJSON<Progress>(request, '/api/progress')).lessons.find((l) => l.lesson_id === target.id)?.status).toBe('completed')

    // A quiz: a blank attempt reveals the answers, then a second attempt answers everything.
    const quizSection = course.sections.find((s) => s.id === target.section_id && s.question_count > 0) ?? course.sections.find((s) => s.question_count > 0)!
    const blank = await request.post(`/api/sections/${quizSection.id}/quiz/attempts`, { data: { answers: [] } })
    expect(blank.ok()).toBeTruthy()
    const key = ((await blank.json()) as AttemptResult).results
    const questions = await getQuiz(request, quizSection.id)
    const second = await request.post(`/api/sections/${quizSection.id}/quiz/attempts`, {
      data: { answers: questions.map((q) => ({ question_id: q.id, answer: key.find((r) => r.question_id === q.id)!.correct_answer })) },
    })
    expect(second.ok()).toBeTruthy()
    expect(((await second.json()) as AttemptResult).percentage).toBe(100)

    // A lab (the lab flow itself is covered by labs.spec; here the status API records the completion).
    const lab = topicsOf(g0).flatMap((t) => t.labs)[0]
    if (lab) expect((await request.put(`/api/labs/${lab.id}/status`, { data: { status: 'completed' } })).ok()).toBeTruthy()

    const [g, progress, activity] = await Promise.all([getGuide(request), getJSON<Progress>(request, '/api/progress'), getJSON<Activity>(request, '/api/activity')])
    await page.goto('/progress')
    await expect(page.locator('[data-ring]')).toBeVisible()

    // Overall: progress over covered topics only, with the uncovered ones reported separately.
    const ex = expected(g)
    expect(ex.done).toBeGreaterThan(0)
    await expect(page.locator('[data-overall-line]')).toHaveText(`${ex.done} of ${ex.covered} covered topics done`)
    await expect(page.locator('[data-overall-uncovered]')).toHaveText(`${ex.notCovered} topic${ex.notCovered === 1 ? '' : 's'} not yet in the course`)
    await expect(page.locator('[data-ring]')).toHaveAttribute('aria-label', `${ex.done} of ${ex.covered} covered topics done`)

    // Each section card: counts, lessons and labs derived from the guide response.
    for (const s of g.sections) {
      const card = page.locator(`[data-guide-section="${s.id}"]`)
      const covered = s.topics.filter((t) => t.coverage !== 'none')
      const done = covered.filter((t) => t.lessons.length > 0 && t.lessons.every((l) => l.status === 'completed')).length
      const lessons = new Map(s.topics.flatMap((t) => t.lessons).map((l) => [l.id, l.status]))
      const labs = new Map(s.topics.flatMap((t) => t.labs).map((l) => [l.id, l.status]))
      await expect(card.locator('[data-section-summary]')).toContainText(covered.length === 0 ? 'Not covered in this course yet' : `${done} of ${covered.length} covered topics done`)
      await expect(card.locator('[data-section-mix]')).toHaveText(
        `${s.topics.filter((t) => t.coverage === 'full').length} full · ${s.topics.filter((t) => t.coverage === 'partial').length} partial · ${s.topics.filter((t) => t.coverage === 'none').length} not covered`,
      )
      await expect(card.locator('[data-section-lessons]')).toHaveText(`Lessons ${[...lessons.values()].filter((x) => x === 'completed').length}/${lessons.size}`)
      await expect(card.locator('[data-section-labs]')).toHaveText(`Labs ${[...labs.values()].filter((x) => x === 'completed').length}/${labs.size}`)
    }

    // Expanding a section shows its topics; "none" topics are shown, say so, and carry no progress numbers.
    for (const s of g.sections) await page.locator(`[data-guide-section="${s.id}"] h3 button`).click()
    const none = topicsOf(g).filter((t) => t.coverage === 'none')
    await expect(page.locator('[data-coverage="none"][data-guide-topic]')).toHaveCount(none.length)
    await expect(page.locator('[data-not-covered]')).toHaveCount(none.length)
    for (const t of none) await expect(page.locator(`[data-guide-topic="${t.id}"]`)).toContainText('Not covered in this course yet')
    expect(await studyText(page)).not.toMatch(noLockWords)

    // The completed lesson's topics show it, with the topic's quiz score and lab status.
    const done = topicsOf(g).find((t) => t.lessons.some((l) => l.id === target.id))!
    const row = page.locator(`[data-guide-topic="${done.id}"]`)
    await expect(row.locator(`[data-lesson-status="completed"]`).first()).toBeVisible()
    if (done.progress.best_section_score_pct != null) await expect(row.locator('[data-topic-quiz]')).toContainText(`${Math.round(done.progress.best_section_score_pct)}%`)
    for (const l of done.labs) await expect(row.locator('[data-topic-labs]')).toContainText(l.status === 'not_started' ? 'not started' : l.status.replace('_', ' '))

    // Continue where you left off: the last lesson touched.
    expect(activity.last_lesson).not.toBeNull()
    await expect(page.locator('[data-continue] a')).toHaveAttribute('href', `/course/${activity.last_lesson!.section_id}/${activity.last_lesson!.lesson_id}`)

    // Activity: the heatmap holds exactly the events of the API, on their local days.
    const perDay = new Map<string, number>()
    for (const e of activity.events) perDay.set(localDate(new Date(e.at)), (perDay.get(localDate(new Date(e.at))) ?? 0) + 1)
    const cells = await page.locator('[data-heatmap-day]').evaluateAll((els) => els.map((e) => [(e as HTMLElement).dataset.heatmapDay!, Number((e as HTMLElement).dataset.count)] as const))
    expect(cells.reduce((n, [, c]) => n + c, 0)).toBe(activity.events.length)
    for (const [day, n] of perDay) expect(cells.find(([d]) => d === day)?.[1], day).toBe(n)
    await expect(page.locator('[data-sessions]')).toContainText('session')
    for (const l of activity.limits) await expect(page.locator('[data-activity-limits]')).toContainText(l)

    // Quiz performance: best score per section from /api/progress; the sparkline appears on hover and on focus.
    for (const s of progress.sections) {
      const best = page.locator(`[data-quiz-section="${s.section_id}"] [data-quiz-best]`)
      if ((course.sections.find((c) => c.id === s.section_id)?.question_count ?? 0) === 0) continue
      await expect(best).toHaveText(s.best_score_pct == null ? '—' : `${Math.round(s.best_score_pct)}%`)
    }
    const attempts = activity.quiz.find((q) => q.section_id === quizSection.id)!.attempts.length
    expect(attempts).toBeGreaterThanOrEqual(2)
    const row2 = page.locator(`[data-quiz-section="${quizSection.id}"]`)
    await expect(row2.locator('[data-sparkline]')).toBeHidden()
    await row2.locator('[tabindex="0"]').focus()
    await expect(row2.locator('[data-sparkline]')).toBeVisible()
    await expect(row2.locator('[data-sparkline]')).toHaveAttribute('data-sparkline', String(Math.min(attempts, 10)))

    // Goal coverage (step 15) is still there, and the old sections live under Details.
    await expect(page.locator('[data-goal-coverage]')).toBeVisible()
    await page.locator('[data-progress-details] summary').click()
    await expect(page.getByTestId('overall-pct')).toBeVisible()
    await expect(page.getByTestId('labs-summary')).toContainText(`${progress.labs.completed} of`)
  })

  test('weak topics link to their study guide topic, which opens on the dashboard', async ({ page, request }) => {
    const g = await getGuide(request)
    // Two blank attempts make every topic of a section weak (a blank answer is wrong): at least 3 answers, accuracy under 70%.
    // The section is the first guide topic's, so its quiz tags are mapped to guide topics.
    const section = topicsOf(g).find((t) => t.lessons.length > 0)!.lessons[0]!.section_id
    for (let i = 0; i < 2; i++) expect((await request.post(`/api/sections/${section}/quiz/attempts`, { data: { answers: [] } })).ok()).toBeTruthy()
    const weak = (await getJSON<{ topics: { topic: string; weak: boolean }[] }>(request, '/api/progress/topics')).topics.filter((t) => t.weak)
    expect(weak.length).toBeGreaterThan(0)
    const mapped = new Set(topicsOf(g).flatMap((t) => t.quiz_topics.map((q) => q.topic)))
    await page.goto('/progress')
    for (const w of weak) {
      const item = page.locator(`[data-weak-topic="${w.topic}"]`)
      await expect(item).toBeVisible()
      // A tag with no guide topic (the routing section is beyond the study guide) simply has no guide link.
      await expect(item.locator('[data-weak-guide-topics]')).toHaveCount(mapped.has(w.topic) ? 1 : 0)
    }
    const withGuide = weak.find((w) => mapped.has(w.topic))!
    await page.locator(`[data-weak-topic="${withGuide.topic}"] [data-weak-guide-topics] a`).first().click()
    const id = new URL(page.url()).hash.replace('#topic-', '')
    expect(topicsOf(g).some((t) => t.id === id && t.quiz_topics.some((q) => q.topic === withGuide.topic))).toBeTruthy()
    await expect(page.locator(`[data-guide-topic="${id}"]`)).toBeVisible()
    await expect(page.locator(`[data-guide-topic="${id}"]`)).toBeInViewport()
  })
})

test.describe('sidebar order', () => {
  test('toggles between course order and study guide order, and remembers the choice', async ({ page, request }) => {
    const g = await getGuide(request)
    const course = await getJSON<{ sections: { id: string; title: string }[] }>(request, '/api/course')
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: 'Course' })
    await expect(nav.locator('[data-order-toggle="course"]')).toHaveAttribute('aria-pressed', 'true')
    await expect(nav.locator('[data-sidebar-order]')).toHaveCount(0)

    await nav.locator('[data-order-toggle="guide"]').click()
    await expect(nav.locator('[data-sidebar-order="guide"]')).toBeVisible()
    const keys = await nav.locator('[data-guide-nav]').evaluateAll((els) => els.map((e) => (e as HTMLElement).dataset.guideNav!))
    expect(keys).toEqual([...g.sections.map((s) => s.id), 'beyond', 'quizzes'])
    for (const s of g.sections) {
      const unique = new Set(s.topics.flatMap((t) => t.lessons.map((l) => l.id)))
      const hrefs = await nav.locator(`[data-guide-nav="${s.id}"] a`).evaluateAll((els) => els.map((e) => (e as HTMLAnchorElement).getAttribute('href')!))
      // A lesson listed under an earlier section is not repeated, so this is at most the section's unique lessons.
      expect(hrefs.length).toBeLessThanOrEqual(unique.size)
      for (const h of hrefs) expect(unique.has(h.split('/').pop()!), `${s.id}: ${h}`).toBeTruthy()
    }
    // Every lesson of the course appears exactly once in the guide order.
    const all = await nav.locator('[data-guide-nav] a').evaluateAll((els) => els.map((e) => (e as HTMLAnchorElement).getAttribute('href')!).filter((h) => h.split('/').length === 4 && !h.endsWith('/quiz')))
    expect(new Set(all).size).toBe(all.length)

    await page.reload()
    await expect(nav.locator('[data-order-toggle="guide"]')).toHaveAttribute('aria-pressed', 'true')
    await expect(nav.locator('[data-sidebar-order="guide"]')).toBeVisible()

    await nav.locator('[data-order-toggle="course"]').click()
    await expect(nav.locator('[data-sidebar-order]')).toHaveCount(0)
    const titles = await nav.locator('button[aria-expanded]').evaluateAll((els) => els.map((e) => (e.textContent ?? '').replace(/^▶/, '').trim()))
    for (const s of course.sections) expect(titles.some((t) => t.startsWith(s.title)), s.title).toBeTruthy()
    await page.reload()
    await expect(nav.locator('[data-order-toggle="course"]')).toHaveAttribute('aria-pressed', 'true')
    expect(await page.evaluate(() => localStorage.getItem('vca.sidebar-order'))).toBe('course')
  })
})

test.describe('dark mode at 390px', () => {
  test.use({ colorScheme: 'dark', viewport: { width: 390, height: 844 }, hasTouch: true })

  test('the dashboard has no horizontal overflow, even with everything open', async ({ page, request }) => {
    const g = await getGuide(request)
    await page.goto('/progress')
    await expect(page.locator('[data-ring]')).toBeVisible()
    expect(await overflow(page)).toBeLessThanOrEqual(1)
    for (const s of g.sections) await page.locator(`[data-guide-section="${s.id}"] h3 button`).click()
    await page.locator('[data-progress-details] summary').click()
    await page.waitForTimeout(400)
    expect(await overflow(page)).toBeLessThanOrEqual(1)
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
    const [r, gg, b] = (bg.match(/\d+/g) ?? ['255', '255', '255']).map(Number)
    expect((r! + gg! + b!) / 3).toBeLessThan(80)
  })

  test('the glossary has no horizontal overflow, and the section menu jumps to a section', async ({ page, request }) => {
    const g = await getGuide(request)
    await page.goto('/glossary')
    await expect(page.locator('[data-glossary-view="guide"]')).toBeVisible()
    expect(await overflow(page)).toBeLessThanOrEqual(1)
    await page.getByRole('searchbox').fill('a')
    await page.waitForTimeout(400)
    expect(await overflow(page)).toBeLessThanOrEqual(1)
    await page.getByRole('searchbox').fill('')
    const select = page.getByLabel('Jump to section')
    await expect(select).toBeVisible()
    await select.selectOption(g.sections[g.sections.length - 1]!.id)
    await expect(page.locator(`#guide-${g.sections[g.sections.length - 1]!.id}`)).toBeInViewport()
    expect(await overflow(page)).toBeLessThanOrEqual(1)
  })
})
