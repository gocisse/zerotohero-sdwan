import { useOutletContext } from 'react-router-dom'
import type { Conditions, RoutingAssumptions, TopologyDetail } from '../api'

export interface TopologyCtx {
  detail: TopologyDetail
  topologyId: string
  /** What-if conditions for the overlay screens. In memory only: nothing is stored. */
  conditions: Conditions
  setConditions: (c: Conditions) => void
  /** Routing assumption switches the reader has changed (missing keys use the documented defaults). In memory only. */
  assumptions: Partial<RoutingAssumptions>
  setAssumptions: (a: Partial<RoutingAssumptions>) => void
}

export const useTopologyCtx = () => useOutletContext<TopologyCtx>()
