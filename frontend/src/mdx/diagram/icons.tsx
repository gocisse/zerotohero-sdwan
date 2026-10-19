import type { NodeKind } from '../../api'

/** Distinct SVG icons, drawn here (no external images). Centered on 0,0 within about ±24 units. */
const common = { fill: 'none', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round' } as const

const cloud = 'M-17 11 A9 9 0 0 1 -14 -6 A12 12 0 0 1 8 -11 A10 10 0 0 1 20 4 A6.5 6.5 0 0 1 16 11 Z'

export default function NodeIcon({ kind }: { kind: NodeKind }) {
  switch (kind) {
    case 'vce':
      return (
        <g {...common}>
          <rect x={-23} y={-17} width={46} height={34} rx={6} />
          <path d="M-12 -5 H12 M7 -10 L12 -5 L7 0 M12 7 H-12 M-7 2 L-12 7 L-7 12" />
        </g>
      )
    case 'vco':
      return (
        <g {...common}>
          <path d="M0 -24 L21 -12 L21 12 L0 24 L-21 12 L-21 -12 Z" />
          <circle cx={0} cy={0} r={6} />
          <path d="M0 -6 V-14 M5 3 L12 7 M-5 3 L-12 7" />
        </g>
      )
    case 'vcg':
      return (
        <g {...common}>
          <circle cx={0} cy={0} r={22} />
          <path d="M-10 14 V-1 A10 10 0 0 1 10 -1 V14 M-10 14 H10 M-16 0 H-10 M10 0 H16" />
        </g>
      )
    case 'router':
      return (
        <g {...common}>
          <circle cx={0} cy={0} r={22} />
          <path d="M-14 0 H14 M-14 0 l5 -5 M-14 0 l5 5 M14 0 l-5 -5 M14 0 l-5 5 M0 -14 V14 M0 -14 l-5 5 M0 -14 l5 5 M0 14 l-5 -5 M0 14 l5 -5" />
        </g>
      )
    case 'l3switch':
    case 'l2switch':
      return (
        <g {...common}>
          <rect x={-24} y={-14} width={48} height={28} rx={4} />
          <path d="M-16 6 H-12 M-6 6 H-2 M4 6 H8 M14 6 H18" strokeWidth={4} />
          <text x={0} y={-3} textAnchor="middle" fontSize={11} fontWeight={700} stroke="none" fill="currentColor">
            {kind === 'l3switch' ? 'L3' : 'L2'}
          </text>
        </g>
      )
    case 'firewall':
      return (
        <g {...common}>
          <rect x={-23} y={-18} width={46} height={36} rx={3} />
          <path d="M-23 -6 H23 M-23 6 H23 M-4 -18 V-6 M10 -6 V6 M-10 6 V18 M4 6 V18" />
        </g>
      )
    case 'host':
      return (
        <g {...common}>
          <rect x={-20} y={-17} width={40} height={25} rx={3} />
          <path d="M0 8 V16 M-11 16 H11" />
        </g>
      )
    case 'cloud':
      return (
        <g {...common}>
          <path d={cloud} />
        </g>
      )
    case 'internet':
      return (
        <g {...common}>
          <circle cx={0} cy={0} r={22} />
          <ellipse cx={0} cy={0} rx={9} ry={22} />
          <path d="M-22 0 H22 M-18 -11 H18 M-18 11 H18" />
        </g>
      )
    case 'mpls':
      return (
        <g {...common}>
          <path d={cloud} />
          <text x={1} y={3} textAnchor="middle" fontSize={9} fontWeight={700} stroke="none" fill="currentColor">
            MPLS
          </text>
        </g>
      )
    case 'datacenter':
      return (
        <g {...common}>
          <rect x={-18} y={-22} width={36} height={13} rx={2} />
          <rect x={-18} y={-7} width={36} height={13} rx={2} />
          <rect x={-18} y={8} width={36} height={13} rx={2} />
          <path d="M-12 -15.5 H-6 M-12 -0.5 H-6 M-12 14.5 H-6" />
          <circle cx={11} cy={-15.5} r={1.5} />
          <circle cx={11} cy={-0.5} r={1.5} />
          <circle cx={11} cy={14.5} r={1.5} />
        </g>
      )
    case 'saas':
      return (
        <g {...common}>
          <rect x={-23} y={-17} width={46} height={34} rx={5} />
          <path d="M-23 -8 H23" />
          <circle cx={-17} cy={-12.5} r={1.2} />
          <circle cx={-12} cy={-12.5} r={1.2} />
          <text x={0} y={9} textAnchor="middle" fontSize={11} fontWeight={700} stroke="none" fill="currentColor">
            SaaS
          </text>
        </g>
      )
    case 'generic':
      return (
        <g {...common}>
          <path d="M0 -22 L22 0 L0 22 L-22 0 Z" />
          <circle cx={0} cy={0} r={3} />
        </g>
      )
  }
}
