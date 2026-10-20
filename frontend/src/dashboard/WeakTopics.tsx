import { Link } from 'react-router-dom'
import type { StudyGuide, TopicMastery } from '../api'
import { pct } from '../lib/course'
import { topicsForQuizTag } from '../studyguide/derive'

/** Weak quiz topics (accuracy under 70% over at least 3 answers), weakest first, each with the guide topic and lessons to review. */
export default function WeakTopics({ topics, guide }: { topics: TopicMastery[] | null; guide: StudyGuide | null }) {
  const weak = (topics ?? []).filter((t) => t.weak)
  return (
    <section aria-labelledby="weak-h" data-weak-topics>
      <h2 id="weak-h" className="text-lg font-semibold">
        Topics worth revisiting
      </h2>
      {topics && weak.length === 0 && (
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400" data-weak-empty>
          {topics.length === 0
            ? 'Take a quiz and your topic strengths will appear here.'
            : 'No weak topics right now. Nice!'}
        </p>
      )}
      <ol className="mt-2 space-y-3">
        {weak.map((t) => {
          const inGuide = guide ? topicsForQuizTag(guide, t.topic) : []
          return (
            <li
              key={t.topic}
              data-weak-topic={t.topic}
              className="rounded-lg border border-rose-200 bg-rose-50 p-4 dark:border-rose-900 dark:bg-rose-950/30"
            >
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium">{t.topic}</span>
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {pct(t.accuracy_pct)} accuracy over {t.answers} answers
                </span>
              </div>
              {inGuide.length > 0 && (
                <p className="mt-1 text-sm" data-weak-guide-topics>
                  Study guide:{' '}
                  {inGuide.map(({ section, topic }, i) => (
                    <span key={topic.id}>
                      {i > 0 && ', '}
                      <Link
                        className="text-sky-700 underline dark:text-sky-300"
                        to={{ pathname: '/progress', hash: `#topic-${topic.id}` }}
                      >
                        {section.id} · {topic.title}
                      </Link>
                    </span>
                  ))}
                </p>
              )}
              {t.lessons.length > 0 && (
                <p className="mt-1 text-sm">
                  Review:{' '}
                  {t.lessons.map((l, i) => (
                    <span key={l.id}>
                      {i > 0 && ', '}
                      <Link className="text-sky-700 underline dark:text-sky-300" to={`/course/${l.section_id}/${l.id}`}>
                        {l.title}
                      </Link>
                    </span>
                  ))}
                </p>
              )}
            </li>
          )
        })}
      </ol>
      {topics && topics.length > weak.length && (
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer text-slate-600 dark:text-slate-400">All topics</summary>
          <ul className="mt-2 space-y-1">
            {topics.map((t) => (
              <li key={t.topic}>
                {t.topic}: {pct(t.accuracy_pct)} ({t.answers} answers)
              </li>
            ))}
          </ul>
        </details>
      )}
    </section>
  )
}
