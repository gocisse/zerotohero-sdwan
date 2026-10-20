import { expect, type APIRequestContext, type Page } from '@playwright/test'

// ---- API shapes (only the fields the tests use) ----
export interface LessonSummary { id: string; title: string }
export interface SectionSummary {
  id: string
  title: string
  placeholder: boolean
  question_count: number
  lessons: LessonSummary[]
}
export interface Course { title: string; sections: SectionSummary[] }
export interface QuizQuestion { id: string; type: 'single_choice' | 'multi_choice' | 'true_false' | 'ordering'; topic: string; options: { id: string; text: string }[] }
export interface QuestionResult {
  question_id: string
  topic: string
  correct: boolean
  correct_answer: string | string[]
  explanation: string
  option_explanations: Record<string, string>
  lesson_ref?: string
}
export interface AttemptResult { score: number; total: number; percentage: number; passed: boolean; results: QuestionResult[] }
export interface Diagram {
  id: string
  nodes: { id: string; label: string; kind: string }[]
  links: { id: string; style: string }[]
  steps: { title: string; packets: unknown[] }[]
}
export interface GlossaryTerm { id: string; term: string; aliases: string[]; short: string }

export const answerKey = (s: string | string[]) => (Array.isArray(s) ? s : [s])

// ---- data discovery ----
export async function getJSON<T>(request: APIRequestContext, path: string): Promise<T> {
  const res = await request.get(path)
  expect(res.ok(), `${path} -> ${res.status()}`).toBeTruthy()
  return (await res.json()) as T
}

export const getCourse = (r: APIRequestContext) => getJSON<Course>(r, '/api/course')

export interface LessonRef { sectionId: string; id: string; title: string; placeholder: boolean }
export const allLessons = (c: Course): LessonRef[] =>
  c.sections.flatMap((s) => s.lessons.map((l) => ({ sectionId: s.id, id: l.id, title: l.title, placeholder: s.placeholder })))

export async function lessonSource(r: APIRequestContext, id: string): Promise<string> {
  return (await getJSON<{ source: string }>(r, `/api/lessons/${id}`)).source
}

export interface PredictInfo { choices: number; answer: number }
export interface Used { diagrams: string[]; predicts: PredictInfo[]; terms: string[] }

/** Components a lesson uses, read from its MDX source. */
export function scanSource(src: string): Used {
  const diagrams = [...src.matchAll(/<Diagram\s+id="([^"]+)"/g)].map((m) => m[1]!)
  const terms = [...src.matchAll(/<Term\s+id="([^"]+)"/g)].map((m) => m[1]!)
  const predicts = [...src.matchAll(/<Predict\b([\s\S]*?)<\/Predict>/g)].map((m) => {
    const body = m[1]!
    const answer = Number(/answer=\{(\d+)\}/.exec(body)?.[1])
    const list = /choices=\{\[([\s\S]*?)\]\}/.exec(body)?.[1] ?? ''
    return { answer, choices: [...list.matchAll(/"(?:[^"\\]|\\.)*"/g)].length }
  })
  return { diagrams, predicts, terms }
}

/** Every lesson, with what it uses. */
export async function scanCourse(r: APIRequestContext) {
  const course = await getCourse(r)
  const lessons = allLessons(course)
  const used = await Promise.all(lessons.map(async (l) => ({ ...l, ...scanSource(await lessonSource(r, l.id)) })))
  return { course, lessons, used }
}

export const lessonUrl = (l: { sectionId: string; id: string }) => `/course/${l.sectionId}/${l.id}`

export async function openLesson(page: Page, l: LessonRef) {
  await page.goto(lessonUrl(l))
  await expect(page.getByRole('heading', { level: 1, name: l.title })).toBeVisible()
  await page.locator('[data-lesson-end]').waitFor()
}

// ---- quiz helpers ----
export async function gotoQuestion(page: Page, n: number) {
  await page.getByRole('button', { name: new RegExp(`^Question ${n},`) }).click()
}

/** Order the ordering question's items to match `desired` (option ids), using the up buttons. */
export async function arrange(page: Page, qid: string, desired: string[]) {
  const prefix = `ord-${qid}-`
  for (let i = 0; i < desired.length; i++) {
    for (;;) {
      const ids = await page.locator(`button[id^="${prefix}"][id$="-up"]`).evaluateAll((els, p) =>
        els.map((e) => e.id.slice((p as string).length, -'-up'.length)), prefix)
      const at = ids.indexOf(desired[i]!)
      if (at <= i) break
      await page.locator(`#${prefix}${desired[i]}-up`).click()
    }
  }
}

export async function answerUI(page: Page, q: QuizQuestion, answer: string | string[]) {
  switch (q.type) {
    case 'single_choice':
    case 'true_false':
      await page.locator(`input[name="q-${q.id}"][value="${answer as string}"]`).check()
      break
    case 'multi_choice':
      for (const id of answer as string[]) await page.locator(`input[type=checkbox][value="${id}"]`).check()
      break
    case 'ordering': {
      await arrange(page, q.id, answer as string[])
      // The server shuffles on every request, so the order can already be right and nothing needs moving. The page counts
      // an ordering answer only once something moved or "Keep this order" was pressed, so confirm it here.
      const keep = page.getByRole('button', { name: 'Keep this order' })
      if (await keep.isVisible()) await keep.click()
      await expect(page.getByRole('button', { name: /Order confirmed/ })).toBeVisible()
      break
    }
  }
}

export async function getQuiz(r: APIRequestContext, sectionId: string): Promise<QuizQuestion[]> {
  return (await getJSON<{ questions: QuizQuestion[] }>(r, `/api/sections/${sectionId}/quiz`)).questions
}

export function waitForAttempt(page: Page, timeout?: number) {
  return page.waitForResponse((res) => /\/quiz\/attempts$/.test(res.url()) && res.request().method() === 'POST', { timeout })
}
