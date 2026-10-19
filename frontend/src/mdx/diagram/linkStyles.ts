import type { LinkStyle, NodeKind } from '../../api'

/**
 * Link looks, defined ONCE and used for drawing, packets and the legend.
 * Colors come from the Okabe-Ito colorblind-safe palette; every style also differs in dash pattern and/or
 * weight, so no style relies on color alone. Classes are complete strings so Tailwind can see them.
 */
export interface LinkLook {
  label: string
  stroke: string
  fill: string
  width: number
  dash?: string
  round?: boolean
}

export const LINK_STYLES: Record<LinkStyle, LinkLook> = {
  'underlay-internet': {
    label: 'Internet underlay',
    stroke: 'stroke-[#0072B2] dark:stroke-[#56B4E9]',
    fill: 'fill-[#0072B2] dark:fill-[#56B4E9]',
    width: 3.5,
  },
  'underlay-mpls': {
    label: 'MPLS underlay',
    stroke: 'stroke-[#D55E00] dark:stroke-[#FF9A55]',
    fill: 'fill-[#D55E00] dark:fill-[#FF9A55]',
    width: 3.5,
    dash: '12 6',
  },
  'underlay-lte': {
    label: 'LTE underlay',
    stroke: 'stroke-[#CC79A7] dark:stroke-[#E8A8CB]',
    fill: 'fill-[#CC79A7] dark:fill-[#E8A8CB]',
    width: 4.5,
    dash: '0.1 8',
    round: true,
  },
  'overlay-tunnel': {
    label: 'Overlay tunnel',
    stroke: 'stroke-[#009E73] dark:stroke-[#34D3A5]',
    fill: 'fill-[#009E73] dark:fill-[#34D3A5]',
    width: 3,
    dash: '16 4 3 4',
  },
  ipsec: {
    label: 'IPsec',
    stroke: 'stroke-[#7B4FB5] dark:stroke-[#B79BE6]',
    fill: 'fill-[#7B4FB5] dark:fill-[#B79BE6]',
    width: 6,
    dash: '5 3',
  },
  control: {
    label: 'Control',
    stroke: 'stroke-[#6B7280] dark:stroke-[#9CA3AF]',
    fill: 'fill-[#6B7280] dark:fill-[#9CA3AF]',
    width: 1.6,
    dash: '5 4',
  },
  lan: {
    label: 'LAN',
    stroke: 'stroke-[#334155] dark:stroke-[#CBD5E1]',
    fill: 'fill-[#334155] dark:fill-[#CBD5E1]',
    width: 2.4,
  },
}

export const KIND_NAMES: Record<NodeKind, string> = {
  vce: 'VCE',
  vco: 'VCO',
  vcg: 'VCG',
  router: 'Router',
  l3switch: 'L3 switch',
  l2switch: 'L2 switch',
  firewall: 'Firewall',
  host: 'Host',
  cloud: 'Cloud',
  internet: 'Internet',
  mpls: 'MPLS',
  datacenter: 'Data center',
  saas: 'SaaS',
  generic: 'Generic',
}
