'use client';

import Link from 'next/link';
import { useAuth } from '@/components/AuthProvider';
import { AdminIcon } from '@/components/admin/AdminIcons';
import { AdminPageHeader, adminCard } from '@/components/admin/AdminStates';
import { buttonStyles } from '@/components/ui';
import type { Role } from '@/lib/types';
import { roleAtLeast } from '@/components/admin/roles';

const READINESS = [
  {
    title: 'Problems',
    href: '/admin/problems',
    status: 'Available',
    detail: 'List, detail, create, update, publish/hidden state, delete, and testcase metadata use real problem-service admin APIs.',
    icon: <AdminIcon.Problems />,
    minRole: 'moderator' as Role,
  },
  {
    title: 'Tags',
    href: '/admin/tags',
    status: 'Available',
    detail: 'Admin tag listing and mutations use real problem-service admin tag APIs.',
    icon: <AdminIcon.Tags />,
    minRole: 'moderator' as Role,
  },
  {
    title: 'Submissions',
    href: '/admin/submissions',
    status: 'Available',
    detail: 'Admin list, detail, attempt provenance, testcase result metadata, and rejudge use real submission-service admin APIs.',
    icon: <AdminIcon.Submissions />,
    minRole: 'moderator' as Role,
  },
  {
    title: 'Users',
    href: '/admin/users',
    status: 'Available',
    detail: 'Search user accounts, inspect profile and account state, manage roles, and suspend or restore account access through Auth APIs.',
    icon: <AdminIcon.Users />,
    minRole: 'admin' as Role,
  },
];

export default function AdminOverviewPage() {
  const { user } = useAuth();
  const visibleReadiness = READINESS.filter((item) => roleAtLeast(user?.role, item.minRole));

  return (
    <>
      <AdminPageHeader
        title="Admin Console"
        description="Operational workspace for real backend-backed management flows. Sections without an API contract are marked unavailable."
      />

      <section
        style={{
          ...adminCard,
          padding: 18,
          marginBottom: 16,
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 14,
        }}
      >
        <span
          style={{
            width: 42,
            height: 42,
            borderRadius: 8,
            background: 'var(--accent-soft)',
            color: 'var(--accent-fg)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <AdminIcon.Dashboard size={20} />
        </span>
        <div style={{ flex: 1, minWidth: 220 }}>
          <h2 style={{ margin: 0, fontSize: 15, fontWeight: 700 }}>API readiness first</h2>
          <p style={{ margin: '3px 0 0', color: 'var(--text2)', fontSize: 13.5 }}>
            This overview intentionally avoids fake totals and charts until dashboard aggregate APIs exist.
          </p>
        </div>
        <Link href="/admin/problems" className="ac-hover-accent" style={buttonStyles.primary(38)}>
          Manage problems
        </Link>
      </section>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 12 }}>
        {visibleReadiness.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="ac-hover-card"
            style={{
              ...adminCard,
              padding: 16,
              color: 'var(--text)',
              textDecoration: 'none',
              minHeight: 132,
              display: 'flex',
              flexDirection: 'column',
              gap: 10,
            }}
          >
            <span style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ color: 'var(--accent-fg)', display: 'flex' }}>{item.icon}</span>
              <strong style={{ fontSize: 14 }}>{item.title}</strong>
              <span
                style={{
                  marginLeft: 'auto',
                  borderRadius: 6,
                  padding: '2px 7px',
                  fontSize: 10.5,
                  fontWeight: 750,
                  color:
                    item.status === 'Available'
                      ? 'var(--success)'
                      : item.status === 'Partial'
                        ? 'var(--warn)'
                        : 'var(--text3)',
                  background:
                    item.status === 'Available'
                      ? 'var(--success-bg)'
                      : item.status === 'Partial'
                        ? 'var(--warn-bg)'
                        : 'var(--surface2)',
                }}
              >
                {item.status}
              </span>
            </span>
            <span style={{ fontSize: 12.8, color: 'var(--text2)', lineHeight: 1.55 }}>{item.detail}</span>
          </Link>
        ))}
      </div>
    </>
  );
}
