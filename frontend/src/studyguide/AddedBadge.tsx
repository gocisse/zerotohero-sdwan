/** Marks a study-guide section the course added beyond the learner's original outline (G6). */
export default function AddedBadge() {
  return (
    <span
      data-added-badge
      className="rounded-full border border-violet-400 bg-violet-50 px-2 py-0.5 text-xs font-medium whitespace-nowrap text-violet-900 dark:border-violet-500 dark:bg-violet-950 dark:text-violet-200"
      title="This section is not in the original study guide outline; the course added it."
    >
      Added by the course
    </span>
  )
}
