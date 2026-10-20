import { useOutletContext } from 'react-router-dom'
import type { Conditions, TraceAssumptions, TopologyDetail } from '../api'

export interface TopologyCtx {
  detail: TopologyDetail
  topologyId: string
  /** What-if conditions for the overlay screens. In memory only: nothing is stored. */
  conditions: Conditions
  setConditions: (c: Conditions) => void
  /** Routing assumption switches the reader has changed (missing keys use the documented defaults). In memory only. */
  assumptions: Partial<TraceAssumptions>
  setAssumptions: (a: Partial<TraceAssumptions>) => void
  /** The query string that keeps a lab's faulted state on links to other simulator screens (empty outside a lab). */
  labSearch: string
}

export const useTopologyCtx = () => useOutletContext<TopologyCtx>()
