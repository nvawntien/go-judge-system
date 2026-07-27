'use client';

import type { CSSProperties, ReactNode } from 'react';
import { Header } from './Header';

/**
 * Every screen except the auth page renders inside the sticky header shell.
 * `flush` is for the workspace, which manages its own full-height layout.
 */
export function AppShell({
  children,
  maxWidth = 1200,
  flush = false,
  style,
}: {
  children: ReactNode;
  maxWidth?: number;
  flush?: boolean;
  style?: CSSProperties;
}) {
  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--bg)',
        color: 'var(--text)',
        transition: 'background-color .25s ease',
      }}
    >
      <Header />
      {flush ? (
        children
      ) : (
        <main
          style={{
            maxWidth,
            margin: '0 auto',
            padding: '26px 20px 56px',
            animation: 'acFadeUp .3s ease',
            ...style,
          }}
        >
          {children}
        </main>
      )}
    </div>
  );
}

export function PageHeading({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'flex-end',
        justifyContent: 'space-between',
        gap: 12,
        marginBottom: 18,
      }}
    >
      <div>
        <h1 style={{ margin: 0, fontSize: 22, fontWeight: 650, letterSpacing: '-0.02em' }}>{title}</h1>
        {subtitle && (
          <p style={{ margin: '4px 0 0', color: 'var(--text2)', fontSize: 13.5 }}>{subtitle}</p>
        )}
      </div>
      {actions}
    </div>
  );
}
