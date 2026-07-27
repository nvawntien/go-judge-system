'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { AppShell } from '@/components/AppShell';
import { Card, CodePath } from '@/components/ui';

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
      <h1 style={{ margin: 0, fontSize: 22, fontWeight: 650, letterSpacing: '-0.02em' }}>{title}</h1>
      <p style={{ margin: '4px 0 22px', color: 'var(--text2)', fontSize: 13.5 }}>{subtitle}</p>

      {children}

      <Card padding="52px 20px" style={{ textAlign: 'center' }}>
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 12, opacity: 0.75 }}>
          <CodePath total={4} done={1} width={72} height={36} dashed />
        </div>
        <p style={{ margin: '0 0 4px', fontSize: 14, fontWeight: 600 }}>{heading}</p>
        <p
          style={{
            margin: '0 auto 16px',
            fontSize: 12.5,
            color: 'var(--text3)',
            maxWidth: 460,
            lineHeight: 1.6,
          }}
        >
          {body}
        </p>
        {cta && (
          <Link
            href={cta.href}
            className="ac-hover-accent"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              height: 36,
              padding: '0 16px',
              borderRadius: 8,
              background: 'var(--accent)',
              color: 'var(--accent-ink)',
              fontSize: 13,
              fontWeight: 600,
              textDecoration: 'none',
            }}
          >
            {cta.label}
          </Link>
        )}
      </Card>
    </AppShell>
  );
}
