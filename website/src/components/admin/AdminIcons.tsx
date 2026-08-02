const strokeProps = {
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
};

export type AdminIconProps = { size?: number };

export const AdminIcon = {
  Dashboard: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <rect x="3" y="3" width="7" height="8" rx="1.5" />
      <rect x="14" y="3" width="7" height="5" rx="1.5" />
      <rect x="14" y="12" width="7" height="9" rx="1.5" />
      <rect x="3" y="15" width="7" height="6" rx="1.5" />
    </svg>
  ),
  Problems: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M8 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2h-2" />
      <path d="M9 3h6v4H9zM9 13l-2 2 2 2M15 13l2 2-2 2" />
    </svg>
  ),
  Tags: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M20.5 13.5 13.6 20.4a2 2 0 0 1-2.8 0L3 12.6V4h8.6l8.9 8.9a2 2 0 0 1 0 2.8Z" />
      <circle cx="7.5" cy="7.5" r="1.2" />
    </svg>
  ),
  Submissions: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M9 11 12 14 22 4" />
      <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
    </svg>
  ),
  Users: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  ),
  Menu: ({ size = 18 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M4 7h16M4 12h16M4 17h16" />
    </svg>
  ),
  PanelLeft: ({ size = 17 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M9 4v16" />
    </svg>
  ),
  X: ({ size = 18 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M18 6 6 18M6 6l12 12" />
    </svg>
  ),
  ChevronLeft: ({ size = 16 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="m15 18-6-6 6-6" />
    </svg>
  ),
  Plus: ({ size = 16 }: AdminIconProps) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M12 5v14M5 12h14" />
    </svg>
  ),
};
