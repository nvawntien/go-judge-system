'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { AdminApiError, AdminLoadingState, AdminPageHeader, adminCard, adminField } from '@/components/admin/AdminStates';
import { AdminPagination, AdminTableShell, DateText, LanguageText, StatusPill, adminTd, adminTh } from '@/components/admin/AdminData';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminSubmissionApi } from '@/lib/api';
import type { LanguageCode, SubmissionStatus } from '@/lib/types';

const PAGE_SIZE = 20;
const TERMINAL_STATUSES: SubmissionStatus[] = [
  'PENDING',
  'JUDGING',
  'ACCEPTED',
  'WRONG_ANSWER',
  'TIME_LIMIT_EXCEEDED',
  'MEMORY_LIMIT_EXCEEDED',
  'RUNTIME_ERROR',
  'COMPILATION_ERROR',
  'SYSTEM_ERROR',
];
const LANGUAGES: LanguageCode[] = ['GO', 'CPP', 'PYTHON', 'JAVA'];

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

export default function AdminSubmissionsPage() {
  const [items, setItems] = useState<Awaited<ReturnType<typeof adminSubmissionApi.list>>['items']>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [status, setStatus] = useState<SubmissionStatus | ''>('');
  const [language, setLanguage] = useState<LanguageCode | ''>('');
  const [problemId, setProblemId] = useState('');
  const [userId, setUserId] = useState('');
  const [appliedProblemId, setAppliedProblemId] = useState('');
  const [appliedUserId, setAppliedUserId] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const parsedProblemId = useMemo(() => {
    const value = Number(appliedProblemId);
    return Number.isFinite(value) && value > 0 ? value : undefined;
  }, [appliedProblemId]);

  const load = useCallback(
    (signal?: AbortSignal) => {
      setLoading(true);
      setError('');
      adminSubmissionApi
        .list(
          {
            page,
            limit: PAGE_SIZE,
            status,
            language,
            problem_id: parsedProblemId,
            user_id: appliedUserId.trim() || undefined,
          },
          signal,
        )
        .then((res) => {
          setItems(res.items ?? []);
          setTotal(res.pagination?.total ?? 0);
          setTotalPages(res.pagination?.total_pages ?? 0);
        })
        .catch((err) => {
          if (!signal?.aborted) setError(errorMessage(err));
        })
        .finally(() => {
          if (!signal?.aborted) setLoading(false);
        });
    },
    [appliedUserId, language, page, parsedProblemId, status],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const applyFilters = (event: FormEvent) => {
    event.preventDefault();
    setPage(1);
    setAppliedProblemId(problemId.trim());
    setAppliedUserId(userId.trim());
  };

  return (
    <>
      <AdminPageHeader
        title="Submissions"
        description="System-wide submission list from the submission-service admin API. Rejudge is reserved for a later phase."
      />

      <form
        onSubmit={applyFilters}
        style={{
          ...adminCard,
          padding: 12,
          marginBottom: 12,
          display: 'flex',
          flexWrap: 'wrap',
          gap: 10,
          alignItems: 'center',
        }}
      >
        <select
          aria-label="Status filter"
          value={status}
          onChange={(event) => {
            setPage(1);
            setStatus(event.target.value as SubmissionStatus | '');
          }}
          className="ac-field"
          style={{ ...adminField, minWidth: 190 }}
        >
          <option value="">All statuses</option>
          {TERMINAL_STATUSES.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </select>
        <select
          aria-label="Language filter"
          value={language}
          onChange={(event) => {
            setPage(1);
            setLanguage(event.target.value as LanguageCode | '');
          }}
          className="ac-field"
          style={{ ...adminField, minWidth: 140 }}
        >
          <option value="">All languages</option>
          {LANGUAGES.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </select>
        <input
          type="number"
          min={1}
          aria-label="Problem ID filter"
          value={problemId}
          onChange={(event) => setProblemId(event.target.value)}
          placeholder="Problem ID"
          className="ac-input"
          style={{ ...adminField, width: 150 }}
        />
        <input
          aria-label="User ID filter"
          value={userId}
          onChange={(event) => setUserId(event.target.value)}
          placeholder="User ID"
          className="ac-input"
          style={{ ...adminField, minWidth: 210, flex: '1 1 220px' }}
        />
        <button type="submit" className="ac-hover-accent" style={buttonStyles.primary(38)}>
          Apply
        </button>
      </form>

      {loading ? (
        <AdminLoadingState title="Loading submissions" />
      ) : error ? (
        <AdminApiError title="Could not load submissions" error={error} onRetry={() => load()} />
      ) : items.length === 0 ? (
        <div style={{ ...adminCard, padding: 28, textAlign: 'center' }}>
          <p style={{ margin: '0 0 5px', fontSize: 14, fontWeight: 680 }}>No submissions found</p>
          <p style={{ margin: 0, color: 'var(--text2)', fontSize: 13 }}>Adjust filters and try again.</p>
        </div>
      ) : (
        <>
          <AdminTableShell>
            <table style={{ width: '100%', minWidth: 980, borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={adminTh}>ID</th>
                  <th style={adminTh}>Problem</th>
                  <th style={adminTh}>User</th>
                  <th style={adminTh}>Language</th>
                  <th style={adminTh}>Status</th>
                  <th style={adminTh}>Created</th>
                  <th style={{ ...adminTh, textAlign: 'right' }}>Detail</th>
                </tr>
              </thead>
              <tbody>
                {items.map((submission) => (
                  <tr key={submission.id}>
                    <td style={{ ...adminTd, fontFamily: 'var(--font-mono)', color: 'var(--text3)' }}>#{submission.id}</td>
                    <td style={adminTd}>
                      <strong>{submission.problem_title || `Problem #${submission.problem_id}`}</strong>
                      <div style={{ color: 'var(--text3)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>#{submission.problem_id}</div>
                    </td>
                    <td style={adminTd}>
                      <strong>{submission.username || '-'}</strong>
                      <div style={{ color: 'var(--text3)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>{submission.user_id}</div>
                    </td>
                    <td style={adminTd}>
                      <LanguageText code={submission.language} />
                    </td>
                    <td style={adminTd}>
                      <StatusPill status={submission.status} />
                    </td>
                    <td style={adminTd}>
                      <DateText value={submission.created_at} />
                    </td>
                    <td style={{ ...adminTd, textAlign: 'right' }}>
                      <Link href={`/admin/submissions/${submission.id}`} className="ac-hover-surface2" style={buttonStyles.secondary(32)}>
                        View
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </AdminTableShell>
          <AdminPagination page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}
    </>
  );
}
