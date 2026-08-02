'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { AdminIcon } from '@/components/admin/AdminIcons';
import {
  AdminApiError,
  AdminDialog,
  AdminLoadingState,
  AdminPageHeader,
  adminCard,
  adminField,
} from '@/components/admin/AdminStates';
import {
  AdminPagination,
  AdminTableShell,
  BooleanPill,
  DateText,
  DifficultyPill,
  adminTd,
  adminTh,
} from '@/components/admin/AdminData';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminProblemApi } from '@/lib/api';
import type { Difficulty, Problem } from '@/lib/types';

const PAGE_SIZE = 12;

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

export default function AdminProblemsPage() {
  const { showToast } = useToast();
  const [items, setItems] = useState<Problem[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [search, setSearch] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [difficulty, setDifficulty] = useState<Difficulty | ''>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshKey, setRefreshKey] = useState(0);
  const [busyId, setBusyId] = useState<number | null>(null);
  const busyRef = useRef(false);
  const [deleteTarget, setDeleteTarget] = useState<Problem | null>(null);

  const totalPages = useMemo(() => (total > 0 ? Math.ceil(total / limit) : 0), [limit, total]);

  const load = useCallback(
    (signal?: AbortSignal) => {
      setLoading(true);
      setError('');
      adminProblemApi
        .list({ page, limit: PAGE_SIZE, difficulty, search: appliedSearch }, signal)
        .then((res) => {
          setItems(res.items ?? []);
          setTotal(res.total ?? 0);
          setLimit(res.limit || PAGE_SIZE);
        })
        .catch((err) => {
          if (signal?.aborted) return;
          setError(errorMessage(err));
        })
        .finally(() => {
          if (!signal?.aborted) setLoading(false);
        });
    },
    [appliedSearch, difficulty, page],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load, refreshKey]);

  const refresh = () => setRefreshKey((current) => current + 1);

  const applyFilters = (event: FormEvent) => {
    event.preventDefault();
    setPage(1);
    setAppliedSearch(search.trim());
  };

  const updateVisibility = async (problem: Problem, next: 'publish' | 'hide') => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusyId(problem.id);
    try {
      if (next === 'publish') await adminProblemApi.publish(problem.id);
      else await adminProblemApi.setHidden(problem.id);
      showToast(next === 'publish' ? 'Problem published' : 'Problem hidden', 'success');
      refresh();
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setBusyId(null);
      busyRef.current = false;
    }
  };

  const deleteProblem = async () => {
    if (!deleteTarget) return;
    if (busyRef.current) return;
    busyRef.current = true;
    setBusyId(deleteTarget.id);
    try {
      await adminProblemApi.delete(deleteTarget.id);
      showToast('Problem deleted', 'success');
      setDeleteTarget(null);
      refresh();
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setBusyId(null);
      busyRef.current = false;
    }
  };

  return (
    <>
      <AdminPageHeader
        title="Problems"
        description="Manage problem catalog records using the problem-service admin API."
        actions={
          <Link href="/admin/problems/new" className="ac-hover-accent" style={{ ...buttonStyles.primary(38), gap: 8 }}>
            <AdminIcon.Plus /> New problem
          </Link>
        }
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
        <input
          type="search"
          aria-label="Search problems"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search title or slug"
          className="ac-input"
          style={{ ...adminField, minWidth: 220, flex: '1 1 260px' }}
        />
        <select
          aria-label="Difficulty filter"
          value={difficulty}
          onChange={(event) => {
            setPage(1);
            setDifficulty(event.target.value as Difficulty | '');
          }}
          className="ac-field"
          style={{ ...adminField, minWidth: 150 }}
        >
          <option value="">All difficulties</option>
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
        <button type="submit" className="ac-hover-accent" style={buttonStyles.primary(38)}>
          Search
        </button>
      </form>

      {loading ? (
        <AdminLoadingState title="Loading problems" />
      ) : error ? (
        <AdminApiError title="Could not load problems" error={error} onRetry={refresh} />
      ) : items.length === 0 ? (
        <div style={{ ...adminCard, padding: 28, textAlign: 'center' }}>
          <p style={{ margin: '0 0 5px', fontSize: 14, fontWeight: 680 }}>No problems found</p>
          <p style={{ margin: '0 0 16px', color: 'var(--text2)', fontSize: 13 }}>
            Adjust filters or create a new problem.
          </p>
          <Link href="/admin/problems/new" className="ac-hover-accent" style={buttonStyles.primary(38)}>
            New problem
          </Link>
        </div>
      ) : (
        <>
          <AdminTableShell>
            <table style={{ width: '100%', minWidth: 940, borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={adminTh}>ID</th>
                  <th style={adminTh}>Title</th>
                  <th style={adminTh}>Difficulty</th>
                  <th style={adminTh}>State</th>
                  <th style={adminTh}>Author</th>
                  <th style={adminTh}>Created</th>
                  <th style={{ ...adminTh, textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((problem) => (
                  <tr key={problem.id}>
                    <td style={{ ...adminTd, fontFamily: 'var(--font-mono)', color: 'var(--text3)' }}>#{problem.id}</td>
                    <td style={adminTd}>
                      <Link href={`/admin/problems/${problem.id}`} style={{ fontWeight: 680 }}>
                        {problem.title}
                      </Link>
                      <div style={{ color: 'var(--text3)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                        {problem.slug}
                      </div>
                    </td>
                    <td style={adminTd}>
                      <DifficultyPill value={problem.difficulty} />
                    </td>
                    <td style={adminTd}>
                      <BooleanPill value={!problem.is_hidden} trueLabel="Published" falseLabel="Hidden" />
                    </td>
                    <td style={{ ...adminTd, fontFamily: 'var(--font-mono)', color: 'var(--text2)' }}>
                      {problem.author_id || '-'}
                    </td>
                    <td style={adminTd}>
                      <DateText value={problem.created_at} />
                    </td>
                    <td style={{ ...adminTd, textAlign: 'right' }}>
                      <span style={{ display: 'inline-flex', flexWrap: 'wrap', justifyContent: 'flex-end', gap: 6 }}>
                        <Link href={`/admin/problems/${problem.id}`} className="ac-hover-surface2" style={buttonStyles.secondary(32)}>
                          Open
                        </Link>
                        {problem.is_hidden ? (
                          <button
                            type="button"
                            disabled={busyId === problem.id}
                            onClick={() => updateVisibility(problem, 'publish')}
                            className="ac-hover-surface2"
                            style={buttonStyles.secondary(32)}
                          >
                            Publish
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={busyId === problem.id}
                            onClick={() => updateVisibility(problem, 'hide')}
                            className="ac-hover-surface2"
                            style={buttonStyles.secondary(32)}
                          >
                            Hide
                          </button>
                        )}
                        <button
                          type="button"
                          disabled={busyId === problem.id}
                          onClick={() => setDeleteTarget(problem)}
                          className="ac-hover-surface2"
                          style={{ ...buttonStyles.secondary(32), color: 'var(--error)' }}
                        >
                          Delete
                        </button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </AdminTableShell>
          <AdminPagination page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}

      {deleteTarget && (
        <AdminDialog title={`Delete problem #${deleteTarget.id}?`} onClose={() => setDeleteTarget(null)}>
            <p style={{ margin: '0 0 18px', color: 'var(--text2)', fontSize: 13.5 }}>
              This calls the real admin delete API. The action should only be used when the backend contract allows
              removing this problem.
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
              <button type="button" onClick={() => setDeleteTarget(null)} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
                Cancel
              </button>
              <button
                type="button"
                onClick={deleteProblem}
                disabled={busyId === deleteTarget.id}
                className="ac-hover-accent"
                style={{ ...buttonStyles.primary(38), background: 'var(--error)' }}
              >
                {busyId === deleteTarget.id ? 'Deleting...' : 'Delete'}
              </button>
            </div>
        </AdminDialog>
      )}
    </>
  );
}
