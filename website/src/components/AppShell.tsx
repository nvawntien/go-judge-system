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
    <div className="ac-app-shell">
      <Header />
      {flush ? (
        children
      ) : (
        <main className="ac-page-main" style={{ maxWidth, ...style }}>
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
    <div className="ac-page-header">
      <div className="ac-page-header-copy">
        <h1 className="ac-page-title">{title}</h1>
        {subtitle && <p className="ac-page-subtitle">{subtitle}</p>}
      </div>
      {actions}
    </div>
  );
}
