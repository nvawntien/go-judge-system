'use client';

import type { CSSProperties, ReactNode } from 'react';

/* ------------------------------------------------------------------ logo */

export function Logo({ size = 26 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 26 26" fill="none" aria-hidden="true">
      <path
        d="M4 20 L10 6 L16 16 L22 4"
        stroke="var(--accent)"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      <circle cx="4" cy="20" r="2.4" fill="var(--accent)" />
      <circle cx="10" cy="6" r="2.4" fill="var(--accent)" />
      <circle cx="16" cy="16" r="2.4" fill="var(--accent)" />
      <circle cx="22" cy="4" r="2.4" fill="var(--accent-fg)" opacity="0.55" />
    </svg>
  );
}

export function Wordmark({ fontSize = 17 }: { fontSize?: number }) {
  return (
    <span style={{ fontSize, fontWeight: 650, letterSpacing: '-0.02em' }}>
      Astra
      <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, color: 'var(--accent-fg)' }}>
        Code
      </span>
    </span>
  );
}

/* ---------------------------------------------------------- code-path svg */

/**
 * AstraCode's signature motif: thin line, small nodes, read left to right like
 * execution through test cases. Filled nodes are progress, hollow ones ahead.
 */
export function CodePath({
  total = 5,
  done = 3,
  width = 72,
  height = 36,
  dashed = false,
}: {
  total?: number;
  done?: number;
  width?: number;
  height?: number;
  dashed?: boolean;
}) {
  const nodes = Array.from({ length: total }, (_, i) => i);
  const step = total > 1 ? (width - 12) / (total - 1) : 0;
  const y = height / 2;

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <path
        d={`M6 ${y} H${width - 6}`}
        stroke={dashed ? 'var(--border2)' : 'var(--accent-soft2)'}
        strokeWidth="1.6"
        fill="none"
        strokeDasharray={dashed ? '3 4' : undefined}
      />
      {nodes.map((i) => (
        <circle
          key={i}
          cx={6 + i * step}
          cy={y}
          r="3.5"
          fill={i < done ? 'var(--accent)' : 'var(--surface3)'}
          stroke={i < done ? 'none' : 'var(--border2)'}
        />
      ))}
    </svg>
  );
}

/* ------------------------------------------------------------- skeletons */

export function SkeletonBar({
  width = '100%',
  height = 12,
  radius = 6,
  style,
}: {
  width?: number | string;
  height?: number;
  radius?: number;
  style?: CSSProperties;
}) {
  return (
    <span
      className="ac-skeleton"
      style={{ display: 'block', width, height, borderRadius: radius, ...style }}
    />
  );
}

/* ------------------------------------------------------------ empty/error */

export function EmptyState({
  title,
  description,
  action,
  nodes = 4,
  done = 0,
}: {
  title: string;
  description?: ReactNode;
  action?: ReactNode;
  nodes?: number;
  done?: number;
}) {
  return (
    <div style={{ padding: '48px 20px', textAlign: 'center' }}>
      <div style={{ marginBottom: 12, opacity: 0.7, display: 'flex', justifyContent: 'center' }}>
        <CodePath total={nodes} done={done} width={72} height={36} dashed />
      </div>
      <p style={{ margin: '0 0 4px', fontSize: 14, fontWeight: 600 }}>{title}</p>
      {description && (
        <p style={{ margin: '0 0 16px', fontSize: 12.5, color: 'var(--text3)' }}>{description}</p>
      )}
      {action}
    </div>
  );
}

export function ErrorState({
  title,
  detail,
  onRetry,
}: {
  title: string;
  detail?: string;
  onRetry?: () => void;
}) {
  return (
    <div role="alert" style={{ padding: '48px 20px', textAlign: 'center', animation: 'acFadeUp .25s ease' }}>
      <span
        aria-hidden="true"
        style={{
          display: 'inline-flex',
          width: 36,
          height: 36,
          borderRadius: '50%',
          background: 'var(--error-bg)',
          color: 'var(--error)',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 16,
          fontWeight: 700,
          marginBottom: 12,
        }}
      >
        !
      </span>
      <p style={{ margin: '0 0 4px', fontSize: 14, fontWeight: 600 }}>{title}</p>
      {detail && (
        <p
          style={{
            margin: '0 0 16px',
            fontSize: 12.5,
            color: 'var(--text3)',
            fontFamily: 'var(--font-mono)',
          }}
        >
          {detail}
        </p>
      )}
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="ac-hover-accent"
          style={{
            height: 36,
            padding: '0 16px',
            border: 'none',
            borderRadius: 8,
            background: 'var(--accent)',
            color: 'var(--accent-ink)',
            fontSize: 13,
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          Retry
        </button>
      )}
    </div>
  );
}

/* -------------------------------------------------------------- fragments */

export function Card({
  children,
  label,
  padding = 20,
  style,
}: {
  children: ReactNode;
  label?: string;
  padding?: number | string;
  style?: CSSProperties;
}) {
  return (
    <section
      aria-label={label}
      style={{
        background: 'var(--surface)',
        border: '1px solid var(--border)',
        borderRadius: 14,
        padding,
        boxShadow: 'var(--shadow)',
        ...style,
      }}
    >
      {children}
    </section>
  );
}

export function DifficultyBadge({
  label,
  color,
  bg,
  small = false,
}: {
  label: string;
  color: string;
  bg: string;
  small?: boolean;
}) {
  return (
    <span
      style={{
        fontSize: small ? 11 : 11.5,
        fontWeight: 600,
        color,
        background: bg,
        borderRadius: 6,
        padding: small ? '1px 7px' : '2px 8px',
        whiteSpace: 'nowrap',
      }}
    >
      {label}
    </span>
  );
}

export function Spinner({ size = 14, color = 'var(--accent)' }: { size?: number; color?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="2.4"
      strokeLinecap="round"
      aria-hidden="true"
      style={{ animation: 'acSpin .9s linear infinite' }}
    >
      <path d="M21 12a9 9 0 1 1-6.2-8.56" />
    </svg>
  );
}

/** Primary / secondary / ghost button styles as plain objects. */
export const buttonStyles = {
  primary: (height = 38): CSSProperties => ({
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    height,
    boxSizing: 'border-box',
    padding: '0 16px',
    border: 'none',
    borderRadius: 8,
    background: 'var(--accent)',
    color: 'var(--accent-ink)',
    fontSize: 13,
    fontWeight: 600,
    lineHeight: 1,
    textAlign: 'center',
    textDecoration: 'none',
    whiteSpace: 'nowrap',
    cursor: 'pointer',
    transition: 'background .15s',
  }),
  secondary: (height = 38): CSSProperties => ({
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    height,
    boxSizing: 'border-box',
    padding: '0 16px',
    border: '1px solid var(--border2)',
    borderRadius: 8,
    background: 'var(--surface)',
    color: 'var(--text)',
    fontSize: 13,
    fontWeight: 600,
    lineHeight: 1,
    textAlign: 'center',
    textDecoration: 'none',
    whiteSpace: 'nowrap',
    cursor: 'pointer',
  }),
  ghost: (height = 38): CSSProperties => ({
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    height,
    boxSizing: 'border-box',
    padding: '0 16px',
    border: 'none',
    borderRadius: 8,
    background: 'none',
    color: 'var(--accent-fg)',
    fontSize: 13,
    fontWeight: 600,
    lineHeight: 1,
    textAlign: 'center',
    textDecoration: 'none',
    whiteSpace: 'nowrap',
    cursor: 'pointer',
  }),
  iconButton: (size = 36): CSSProperties => ({
    width: size,
    height: size,
    borderRadius: 8,
    border: '1px solid var(--border)',
    background: 'var(--surface)',
    color: 'var(--text2)',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  }),
};

/* ------------------------------------------------------------------ icons */

const strokeProps = {
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
};

export const Icon = {
  Search: ({ size = 15, color = 'currentColor' }: { size?: number; color?: string }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps} stroke={color}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.4-3.4" />
    </svg>
  ),
  Sun: ({ size = 16 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  ),
  Moon: ({ size = 16 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
    </svg>
  ),
  Bell: ({ size = 16 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
      <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
    </svg>
  ),
  Chevron: ({ size = 12, color = 'currentColor' }: { size?: number; color?: string }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps} stroke={color} strokeWidth={2.4}>
      <path d="m6 9 6 6 6-6" />
    </svg>
  ),
  Menu: ({ size = 18 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M4 7h16M4 12h16M4 17h16" />
    </svg>
  ),
  Link: ({ size = 13 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    </svg>
  ),
  Bookmark: ({ size = 13, filled = false }: { size?: number; filled?: boolean }) => (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={filled ? 'currentColor' : 'none'}
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
    </svg>
  ),
  Dots: ({ size = 13 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="5" cy="12" r="1.8" />
      <circle cx="12" cy="12" r="1.8" />
      <circle cx="19" cy="12" r="1.8" />
    </svg>
  ),
  Gear: ({ size = 14 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M12.2 2h-.4a2 2 0 0 0-2 2v.2a2 2 0 0 1-1 1.7l-.4.3a2 2 0 0 1-2 0l-.2-.1a2 2 0 0 0-2.7.7l-.2.4a2 2 0 0 0 .7 2.7l.2.1a2 2 0 0 1 1 1.8v.6a2 2 0 0 1-1 1.8l-.2.1a2 2 0 0 0-.7 2.7l.2.4a2 2 0 0 0 2.7.7l.2-.1a2 2 0 0 1 2 0l.4.3a2 2 0 0 1 1 1.7v.2a2 2 0 0 0 2 2h.4a2 2 0 0 0 2-2v-.2a2 2 0 0 1 1-1.7l.4-.3a2 2 0 0 1 2 0l.2.1a2 2 0 0 0 2.7-.7l.2-.4a2 2 0 0 0-.7-2.7l-.2-.1a2 2 0 0 1-1-1.8v-.6a2 2 0 0 1 1-1.8l.2-.1a2 2 0 0 0 .7-2.7l-.2-.4a2 2 0 0 0-2.7-.7l-.2.1a2 2 0 0 1-2 0l-.4-.3a2 2 0 0 1-1-1.7V4a2 2 0 0 0-2-2Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  ),
  Reset: ({ size = 14 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
      <path d="M3 3v5h5" />
    </svg>
  ),
  Expand: ({ size = 14 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M8 3H5a2 2 0 0 0-2 2v3M21 8V5a2 2 0 0 0-2-2h-3M3 16v3a2 2 0 0 0 2 2h3M16 21h3a2 2 0 0 0 2-2v-3" />
    </svg>
  ),
  Collapse: ({ size = 14 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M8 3v3a2 2 0 0 1-2 2H3M21 8h-3a2 2 0 0 1-2-2V3M3 16h3a2 2 0 0 1 2 2v3M16 21v-3a2 2 0 0 1 2-2h3" />
    </svg>
  ),
  Play: ({ size = 11 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M8 5v14l11-7z" />
    </svg>
  ),
  Eye: ({ size = 15 }: { size?: number }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" {...strokeProps}>
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  ),
};
