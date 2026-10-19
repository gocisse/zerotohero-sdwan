import MultiChoice from './MultiChoice'
import Ordering from './Ordering'
import SingleChoice from './SingleChoice'
import TrueFalse from './TrueFalse'
import type { QuestionInputProps } from './shared'

/** Picks the input component for a question's type. */
export default function QuestionInput(props: QuestionInputProps) {
  switch (props.question.type) {
    case 'single_choice':
      return <SingleChoice {...props} />
    case 'true_false':
      return <TrueFalse {...props} />
    case 'multi_choice':
      return <MultiChoice {...props} />
    case 'ordering':
      return <Ordering {...props} />
  }
}
