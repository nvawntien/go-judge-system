'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { AppShell, PageHeading } from '@/components/AppShell';
import { CodePath } from '@/components/ui';

/**
 * Screens the design specifies but the backend has no endpoints for. Rather
 * than invent data, they keep the layout and say exactly what is missing.
 */
export function ComingSoon({
  title,
  subtitle,
  heading,
  body,
  cta = { label: 'Browse problems', href: '/problems' },
  children,
}: {
  title: string;
  subtitle: string;
  heading: string;
  body: ReactNode;
  cta?: { label: string; href: string } | null;
  children?: ReactNode;
}) {
  return (
    <AppShell maxWidth={900}>
      <PageHeading title={title} subtitle={subtitle} />

      {children}

      <section className="ac-panel ac-state ac-coming-soon">
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 12, opacity: 0.75 }}>
          <CodePath total={4} done={1} width={72} height={36} dashed />
        </div>
        <p className="ac-state-title">{heading}</p>
        <p className="ac-state-description">{body}</p>
        {cta && (
          <Link
            href={cta.href}
            className="ac-button ac-button-primary ac-state-action"
          >
            {cta.label}
          </Link>
        )}
      </section>
    </AppShell>
  );
}
