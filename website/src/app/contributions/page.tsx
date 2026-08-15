'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { PageHeading } from '@/components/AppShell';
import { ErrorState, SkeletonBar, buttonStyles } from '@/components/ui';
import { adminCard, adminField } from '@/components/admin/AdminStates';
import { ApiError, NetworkError, contributionProblemApi } from '@/lib/api';
import type { Difficulty, Problem } from '@/lib/types';

const PAGE_SIZE = 12;

function errorMessage(error: unknown) {
  if (error instanceof NetworkError) return 'The Problem Service could not be reached.';
  if (error instanceof ApiError) return error.message || `Request failed with ${error.httpStatus}.`;
  return 'Your contributions could not be loaded.';
}

export default function ContributionsPage() {
  const [items, setItems] = useState<Problem[]>([]);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [difficulty, setDifficulty] = useState<Difficulty | ''>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshKey, setRefreshKey] = useState(0);
  const totalPages = useMemo(() => (total > 0 ? Math.ceil(total / limit) : 0), [limit, total]);

  const load = useCallback((signal?: AbortSignal) => {
    setLoading(true);
    setError('');
    contributionProblemApi
      .listOwn({ page, limit: PAGE_SIZE, search: appliedSearch, difficulty }, signal)
      .then((response) => {
        setItems(response.items ?? []);
        setLimit(response.limit || PAGE_SIZE);
        setTotal(response.total ?? 0);
      })
      .catch((loadError) => {
        if (!signal?.aborted) setError(errorMessage(loadError));
      })
      .finally(() => {
        if (!signal?.aborted) setLoading(false);
      });
  }, [appliedSearch, difficulty, page]);

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load, refreshKey]);

  const applyFilters = (event: FormEvent) => {
    event.preventDefault();
    setPage(1);
    setAppliedSearch(search.trim());
  };

  return (
    <>
      <PageHeading
        title="My contributions"
        subtitle="Create and refine your own problem drafts. Publishing remains a moderation action."
        actions={
          <Link href="/contributions/new" className="ac-hover-accent" style={buttonStyles.primary(38)}>
            New problem
          </Link>
        }
      />

      <form className="ac-contribution-filters" onSubmit={applyFilters} style={adminCard}>
        <input
          type="search"
          aria-label="Search my contributions"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search title or slug"
          className="ac-input"
          style={adminField}
        />
        <select
          aria-label="Difficulty filter"
          value={difficulty}
          onChange={(event) => {
            setPage(1);
            setDifficulty(event.target.value as Difficulty | '');
          }}
          className="ac-field"
          style={adminField}
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
        <div aria-label="Loading contributed problems" className="ac-contribution-grid">
          {[0, 1, 2].map((item) => (
            <div key={item} style={{ ...adminCard, padding: 18, display: 'grid', gap: 10 }}>
              <SkeletonBar width="55%" height={18} />
              <SkeletonBar width="90%" />
              <SkeletonBar width="42%" />
            </div>
          ))}
        </div>
      ) : error ? (
        <ErrorState title="Could not load your contributions" detail={error} onRetry={() => setRefreshKey((key) => key + 1)} />
      ) : items.length === 0 ? (
        <section className="ac-state" style={adminCard}>
          <h2 className="ac-state-title">No contributions found</h2>
          <p className="ac-state-description">
            {appliedSearch || difficulty ? 'Adjust the filters to see other drafts.' : 'Create your first problem draft.'}
          </p>
          <Link href="/contributions/new" className="ac-button ac-button-primary ac-state-action">
            New problem
          </Link>
        </section>
      ) : (
        <>
          <ul className="ac-contribution-grid" aria-label="Contributed problems">
            {items.map((problem) => (
              <li key={problem.id} className="ac-contribution-card" style={adminCard}>
                <div className="ac-contribution-card-meta">
                  <span className={`ac-contribution-state ${problem.is_hidden ? 'is-draft' : 'is-published'}`}>
                    {problem.is_hidden ? 'Draft' : 'Published'}
                  </span>
                  <span>{problem.difficulty}</span>
                </div>
                <div>
                  <h2 className="ac-contribution-card-title">{problem.title}</h2>
                  <p className="ac-contribution-card-slug">{problem.slug}</p>
                </div>
                <div className="ac-contribution-card-actions">
                  <Link
                    href={`/contributions/${problem.id}`}
                    className="ac-hover-surface2"
                    style={buttonStyles.secondary(34)}
                  >
                    {problem.is_hidden ? 'Edit draft' : 'Open'}
                  </Link>
                  {!problem.is_hidden && (
                    <Link href={`/problems/${encodeURIComponent(problem.slug)}`} style={{ fontSize: 12.5, fontWeight: 650 }}>
                      View problem
                    </Link>
                  )}
                </div>
              </li>
            ))}
          </ul>
          {totalPages > 1 && (
            <nav aria-label="Contribution pages" className="ac-contribution-pagination">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
                className="ac-hover-surface2"
                style={buttonStyles.secondary(34)}
              >
                Previous
              </button>
              <span>Page {page} of {totalPages}</span>
              <button
                type="button"
                disabled={page >= totalPages}
                onClick={() => setPage((current) => current + 1)}
                className="ac-hover-surface2"
                style={buttonStyles.secondary(34)}
              >
                Next
              </button>
            </nav>
          )}
        </>
      )}
    </>
  );
}
