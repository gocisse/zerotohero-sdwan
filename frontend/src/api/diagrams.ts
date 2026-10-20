import { getJSON } from './client'

export type NodeKind =
  | 'vce'
  | 'vco'
  | 'vcg'
  | 'router'
  | 'l3switch'
  | 'l2switch'
  | 'firewall'
  | 'host'
  | 'cloud'
  | 'internet'
  | 'mpls'
  | 'datacenter'
  | 'saas'
  | 'generic'

export type LinkStyle =
  'underlay-internet' | 'underlay-mpls' | 'underlay-lte' | 'overlay-tunnel' | 'ipsec' | 'control' | 'lan'

export interface DiagramNode {
  id: string
  label: string
  kind: NodeKind
  x: number
  y: number
  sublabel?: string
  group?: string
}

export interface DiagramGroup {
  id: string
  label: string
  nodes: string[]
  box?: { x: number; y: number; width: number; height: number }
}

export interface DiagramLink {
  id: string
  from: string
  to: string
  style: LinkStyle
  label?: string
  curve?: number
}

export interface DiagramPacket {
  path: string[]
  label?: string
  color_style?: LinkStyle
  delay_ms: number
  /** Resolved by the backend: the node/link sequence the path travels. */
  route?: { nodes: string[]; links: string[] }
}

export interface DiagramStep {
  title: string
  narration: string
  highlight: string[]
  show: string[]
  hide: string[]
  packets: DiagramPacket[]
  callouts: { target: string; text: string }[]
}

export interface Diagram {
  id: string
  title: string
  alt: string
  width: number
  height: number
  nodes: DiagramNode[]
  groups: DiagramGroup[]
  links: DiagramLink[]
  steps: DiagramStep[]
}

export interface GlossaryTerm {
  id: string
  term: string
  aliases: string[]
  short: string
  long?: string
  related: string[]
  lesson_ref?: string
  /** Study-guide topic ids, primary first. */
  guide_topics: string[]
  beyond_guide: boolean
}

export const fetchDiagram = (id: string) => getJSON<Diagram>(`/api/diagrams/${encodeURIComponent(id)}`)
export const fetchGlossary = () => getJSON<{ terms: GlossaryTerm[] }>('/api/glossary')
