'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import { AppShell, PageHeading } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { SubmissionDetail } from '@/components/submission/SubmissionDetail';
import { EmptyState, ErrorState, Icon, SkeletonBar } from '@/components/ui';
import { ApiError, NetworkError, submissionApi } from '@/lib/api';
import {
  LANGUAGES,
  STATUS_FILTERS,
  formatDateTime,
  isPendingStatus,
  languageLabel,
  timeAgo,
  verdictMeta,
} from '@/lib/format';
import { useViewportWidth } from '@/lib/hooks';
import type {
  LanguageCode,
  SubmissionListItem,
  SubmissionStatus,
} from '@/lib/types';

const PAGE_SIZE = 20;
const GRID = 'minmax(200px,1fr) 190px 100px 110px 130px 24px';

export default function SubmissionsPage() {
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const width = useViewportWidth();
  const isMobile = width < 760;

  const [status, setStatus] = useState<SubmissionStatus | ''>('');
  const [language, setLanguage] = useState<LanguageCode | ''>('');
  const [page, setPage] = useState(1);

  const [items, setItems] = useState<SubmissionListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nonce, setNonce] = useState(0);
  const [expanded, setExpanded] = useState<number | null>(null);

  useEffect(() => {
    if (!authLoading && !user) router.replace('/login?next=/submissions');
  }, [authLoading, user, router]);

  useEffect(() => {
    if (!user) return;
    const controller = new AbortController();
    setLoading(true);
    setError(null);

    submissionApi
      .listMine({ page, limit: PAGE_SIZE, status, language }, controller.signal)
      .then((res) => {
        setItems(res.items ?? []);
        setTotal(res.pagination?.total ?? 0);
        setTotalPages(Math.max(1, res.pagination?.total_pages ?? 1));
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        setError(
          err instanceof NetworkError
            ? 'Cannot reach the API gateway'
            : err instanceof ApiError
              ? `GET /api/v1/me/submissions — ${err.httpStatus} ${err.message}`
              : 'Unexpected error',
        );
        setItems([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });

    return () => controller.abort();
  }, [user, page, status, language, nonce]);

  // Anything still queued/judging is polled until it settles.
  const hasPending = items.some((item) => isPendingStatus(item.status));
  useEffect(() => {
    if (!hasPending) return;
    const id = setTimeout(() => setNonce((n) => n + 1), 4000);
    return () => clearTimeout(id);
  }, [hasPending, items]);

  const toggle = useCallback((id: number) => {
    setExpanded((current) => (current === id ? null : id));
  }, []);

  if (!user) return null;

  return (
    <AppShell maxWidth={1100}>
      <PageHeading
        title="Submissions"
        subtitle="Your recent judge activity"
        actions={
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <select
              value={status}
              onChange={(event) => {
                setStatus(event.target.value as SubmissionStatus | '');
                setPage(1);
              }}
              aria-label="Filter by status"
              style={filterStyle}
            >
              {STATUS_FILTERS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <select
              value={language}
              onChange={(event) => {
                setLanguage(event.target.value as LanguageCode | '');
                setPage(1);
              }}
              aria-label="Filter by language"
              style={filterStyle}
            >
              <option value="">Language: All</option>
              {LANGUAGES.map((item) => (
                <option key={item.code} value={item.code}>
                  {item.label}
                </option>
              ))}
            </select>
          </div>
        }
      />

      <section
        aria-label="Submission history"
        className="ac-data-frame"
      >
        <div
          className="ac-toolbar"
          style={{ justifyContent: 'space-between' }}
        >
          <span style={{ fontSize: 12, color: 'var(--text3)', fontFamily: 'var(--font-mono)' }}>
            {loading ? 'loading…' : `${total} submission${total === 1 ? '' : 's'}`}
          </span>
          <span style={{ fontSize: 11.5, color: 'var(--text3)' }}>Click a row for details</span>
        </div>

        {loading && (
          <div style={{ padding: '4px 18px 16px', display: 'flex', flexDirection: 'column', gap: 10 }}>
            {Array.from({ length: 6 }, (_, i) => (
              <SkeletonBar key={i} height={34} radius={8} />
            ))}
          </div>
        )}

        {!loading && error && (
          <ErrorState
            title="Couldn't load submissions"
            detail={error}
            onRetry={() => setNonce((n) => n + 1)}
          />
        )}

        {!loading && !error && items.length === 0 && (
          <EmptyState
            title={status || language ? 'No submissions match these filters' : 'No submissions yet'}
            description={
              status || language
                ? 'Try a different status or language.'
                : 'Solve your first problem to see activity here.'
            }
            action={
              !status && !language ? (
                <Link
                  href="/problems"
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
                  Browse problems
                </Link>
              ) : undefined
            }
          />
        )}

        {!loading && !error && items.length > 0 && (
          <div style={{ overflowX: 'auto' }}>
            <div style={{ minWidth: isMobile ? undefined : 860 }}>
              {!isMobile && (
                <div
                  aria-hidden="true"
                  className="ac-table-head"
                  style={{
                    display: 'grid',
                    gridTemplateColumns: GRID,
                    gap: 12,
                    alignItems: 'center',
                    padding: '9px 18px',
                  }}
                >
                  <span>Problem</span>
                  <span>Result</span>
                  <span>Language</span>
                  <span style={{ textAlign: 'right' }}>
                    Submission
                  </span>
                  <span style={{ textAlign: 'right' }}>
                    Submitted
                  </span>
                  <span />
                </div>
              )}

              <div style={{ display: 'flex', flexDirection: 'column' }}>
                {items.map((item) => {
                  const verdict = verdictMeta(item.status);
                  const open = expanded === item.id;

                  return (
                    <div key={item.id}>
                      <div
                        className="ac-table-row ac-submission-list-row"
                        style={
                          isMobile
                            ? {
                                display: 'flex',
                                alignItems: 'center',
                                gap: 10,
                                width: '100%',
                                padding: '12px 16px',
                                background: open ? 'var(--surface2)' : 'transparent',
                                textAlign: 'left',
                                color: 'var(--text)',
                              }
                            : {
                                display: 'grid',
                                gridTemplateColumns: GRID,
                                gap: 12,
                                alignItems: 'center',
                                width: '100%',
                                padding: 'var(--rowpad) 18px',
                                background: open ? 'var(--surface2)' : 'transparent',
                                textAlign: 'left',
                                color: 'var(--text)',
                                transition: 'background .12s',
                              }
                        }
                      >
                        {isMobile ? (
                          <>
                            <span
                              aria-hidden="true"
                              style={{
                                width: 24,
                                height: 24,
                                borderRadius: 7,
                                background: verdict.bg,
                                color: verdict.color,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                fontSize: 12,
                                fontWeight: 700,
                                flexShrink: 0,
                              }}
                            >
                              {verdict.icon}
                            </span>
                            <Link href={`/submissions/${item.id}`} style={{ flex: 1, minWidth: 0, color: 'var(--text)', textDecoration: 'none' }}>
                              <span
                                style={{
                                  display: 'block',
                                  fontSize: 13,
                                  fontWeight: 550,
                                  whiteSpace: 'nowrap',
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                }}
                              >
                                {item.problem_title}
                              </span>
                              <span style={{ fontSize: 11, color: 'var(--text3)' }}>
                                {verdict.label} · {languageLabel(item.language)} ·{' '}
                                {timeAgo(item.created_at)}
                              </span>
                            </Link>
                            <button
                              type="button"
                              onClick={() => toggle(item.id)}
                              aria-expanded={open}
                              aria-label={`${open ? 'Hide' : 'Preview'} submission ${item.id}`}
                              className="ac-icon-button ac-submission-expand"
                            >
                              <span style={{ display: 'flex', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform .2s' }}>
                                <Icon.Chevron color="var(--text3)" />
                              </span>
                            </button>
                          </>
                        ) : (
                          <>
                            <span style={{ minWidth: 0 }}>
                              <Link
                                href={`/submissions/${item.id}`}
                                style={{
                                  display: 'block',
                                  fontSize: 13,
                                  fontWeight: 550,
                                  color: 'var(--text)',
                                  textDecoration: 'none',
                                  whiteSpace: 'nowrap',
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                }}
                              >
                                {item.problem_title}
                              </Link>
                              <span
                                style={{
                                  fontSize: 11,
                                  color: 'var(--text3)',
                                  fontFamily: 'var(--font-mono)',
                                }}
                              >
                                problem #{item.problem_id}
                              </span>
                            </span>
                            <span
                              style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}
                            >
                              <span
                                aria-hidden="true"
                                style={{
                                  width: 24,
                                  height: 24,
                                  borderRadius: 7,
                                  background: verdict.bg,
                                  color: verdict.color,
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  fontSize: 12,
                                  fontWeight: 700,
                                  flexShrink: 0,
                                }}
                              >
                                {verdict.icon}
                              </span>
                              <span
                                style={{
                                  fontSize: 12,
                                  fontWeight: 600,
                                  color: verdict.color,
                                  whiteSpace: 'nowrap',
                                }}
                              >
                                {verdict.label}
                              </span>
                            </span>
                            <span>
                              <span
                                style={{
                                  fontSize: 11.5,
                                  color: 'var(--text2)',
                                  background: 'var(--surface2)',
                                  border: '1px solid var(--border)',
                                  borderRadius: 5,
                                  padding: '1px 8px',
                                  fontFamily: 'var(--font-mono)',
                                  whiteSpace: 'nowrap',
                                }}
                              >
                                {languageLabel(item.language)}
                              </span>
                            </span>
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11.5,
                                color: 'var(--text2)',
                                textAlign: 'right',
                              }}
                            >
                              #{item.id}
                            </span>
                            <span
                              style={{
                                fontSize: 11.5,
                                color: 'var(--text3)',
                                textAlign: 'right',
                                whiteSpace: 'nowrap',
                              }}
                              title={formatDateTime(item.created_at)}
                            >
                              {timeAgo(item.created_at)}
                            </span>
                            <button
                              type="button"
                              onClick={() => toggle(item.id)}
                              aria-expanded={open}
                              aria-label={`${open ? 'Hide' : 'Preview'} submission ${item.id}`}
                              className="ac-submission-expand"
                              style={{
                                justifySelf: 'end',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                width: 28,
                                height: 28,
                                padding: 0,
                                border: 0,
                                borderRadius: 6,
                                background: 'transparent',
                                transform: open ? 'rotate(180deg)' : 'none',
                                transition: 'transform .2s',
                              }}
                            >
                              <Icon.Chevron color="var(--text3)" />
                            </button>
                          </>
                        )}
                      </div>

                      {open && <SubmissionDetail id={item.id} />}
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {!error && !loading && items.length > 0 && (
          <div
            className="ac-pagination"
          >
            <span className="ac-pagination-meta">
              Page {page} of {totalPages}
            </span>
            <div className="ac-pagination-actions">
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="ac-hover-surface2"
                style={{ ...pagerStyle, opacity: page <= 1 ? 0.45 : 1 }}
              >
                ← Prev
              </button>
              <button
                type="button"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="ac-hover-surface2"
                style={{ ...pagerStyle, opacity: page >= totalPages ? 0.45 : 1 }}
              >
                Next →
              </button>
            </div>
          </div>
        )}
      </section>
    </AppShell>
  );
}

const filterStyle: React.CSSProperties = {
  height: 34,
  borderRadius: 8,
  border: '1px solid var(--border)',
  background: 'var(--surface)',
  padding: '0 8px',
  fontSize: 12.5,
  color: 'var(--text)',
  cursor: 'pointer',
};

const pagerStyle: React.CSSProperties = {
  height: 32,
  padding: '0 12px',
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--surface)',
  color: 'var(--text2)',
  fontSize: 12.5,
  fontWeight: 600,
  cursor: 'pointer',
};
