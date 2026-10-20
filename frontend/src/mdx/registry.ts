import type { MDXComponents } from 'mdx/types'
import { Callout, Table, Td, Th } from './components'
import Diagram from './diagram/Diagram'
import MdxLink from './Link'
import Predict from './Predict'
import Term from './Term'

/** Everything lessons may use in MDX without importing. */
export const mdxComponents: MDXComponents = {
  Callout,
  Diagram,
  Predict,
  Term,
  a: MdxLink,
  table: Table,
  th: Th,
  td: Td,
}
