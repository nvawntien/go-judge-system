'use client';

import type { CSSProperties, ReactNode } from 'react';
import { buttonStyles } from '@/components/ui';
import { difficultyMeta, formatDateTime, languageLabel, verdictMeta } from '@/lib/format';
import type { Difficulty, SubmissionStatus } from '@/lib/types';

export function AdminTableShell({ children }: { children: ReactNode }) {
  return (
    <div
      style={{
        border: '1px solid var(--border)',
        borderRadius: 8,
        background: 'var(--surface)',
        overflowX: 'auto',
        boxShadow: 'var(--shadow)',
      }}
    >
      {children}
    </div>
  );
}

export function DifficultyPill({ value }: { value: Difficulty | string }) {
  const meta = difficultyMeta(value);
  return (
    <span
      style={{
        color: meta.color,
        background: meta.bg,
        borderRadius: 6,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 700,
        whiteSpace: 'nowrap',
      }}
    >
      {meta.label}
    </span>
  );
}

export function StatusPill({ status }: { status: SubmissionStatus | string }) {
  const meta = verdictMeta(status);
  return (
    <span
      style={{
        color: meta.color,
        background: meta.bg,
        borderRadius: 6,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 700,
        whiteSpace: 'nowrap',
      }}
    >
      {meta.label}
    </span>
  );
}

export function BooleanPill({ value, trueLabel, falseLabel }: { value: boolean; trueLabel: string; falseLabel: string }) {
  return (
    <span
      style={{
        color: value ? 'var(--success)' : 'var(--text3)',
        background: value ? 'var(--success-bg)' : 'var(--surface2)',
        borderRadius: 6,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 700,
        whiteSpace: 'nowrap',
      }}
    >
      {value ? trueLabel : falseLabel}
    </span>
  );
}

export function LanguageText({ code }: { code: string }) {
  return <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{languageLabel(code)}</span>;
}

export function DateText({ value }: { value?: string | Date | null }) {
  if (!value) return <span style={{ color: 'var(--text3)' }}>-</span>;
  return <span style={{ color: 'var(--text2)', fontSize: 12 }}>{formatDateTime(value)}</span>;
}

export function AdminPagination({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number;
  totalPages: number;
  total?: number;
  onPage: (page: number) => void;
}) {
  const canPrev = page > 1;
  const canNext = totalPages > 0 && page < totalPages;
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: 10, marginTop: 12 }}>
      <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text3)', fontSize: 11.5 }}>
        Page {page}{totalPages > 0 ? ` / ${totalPages}` : ''}{typeof total === 'number' ? ` · ${total} total` : ''}
      </span>
      <span style={{ display: 'flex', gap: 8 }}>
        <button
          type="button"
          disabled={!canPrev}
          onClick={() => onPage(Math.max(1, page - 1))}
          className="ac-hover-surface2"
          style={{ ...buttonStyles.secondary(36), opacity: canPrev ? 1 : 0.5 }}
        >
          Previous
        </button>
        <button
          type="button"
          disabled={!canNext}
          onClick={() => onPage(page + 1)}
          className="ac-hover-surface2"
          style={{ ...buttonStyles.secondary(36), opacity: canNext ? 1 : 0.5 }}
        >
          Next
        </button>
      </span>
    </div>
  );
}

export const adminTh: CSSProperties = {
  padding: '10px 12px',
  color: 'var(--text3)',
  fontSize: 11,
  fontWeight: 750,
  textAlign: 'left',
  textTransform: 'uppercase',
  letterSpacing: 0.3,
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap',
};

export const adminTd: CSSProperties = {
  padding: '11px 12px',
  borderBottom: '1px solid var(--border)',
  verticalAlign: 'middle',
  fontSize: 13,
};
