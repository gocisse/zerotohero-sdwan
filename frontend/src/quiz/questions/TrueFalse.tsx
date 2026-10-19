import SingleChoice from './SingleChoice'
import type { QuestionInputProps } from './shared'

/** True/false is a two-option single choice (option ids "true" and "false"). */
export default function TrueFalse(props: QuestionInputProps) {
  return <SingleChoice {...props} />
}
