'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { AppShell, PageHeading } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { EmptyState, ErrorState, Icon } from '@/components/ui';
import { ApiError, NetworkError, problemApi } from '@/lib/api';
import {
  DIFFICULTIES,
  difficultyMeta,
  formatMemoryLimit,
  formatTimeLimit,
} from '@/lib/format';
import { useDebounced, useViewportWidth } from '@/lib/hooks';
import { useProgress } from '@/lib/progress';
import type { Difficulty, Problem, Tag } from '@/lib/types';

type StatusFilter = 'all' | 'solved' | 'attempted' | 'unsolved';
type SortOption = 'default' | 'title-asc' | 'difficulty-asc' | 'difficulty-desc' | 'newest';

const PAGE_SIZE = 20;
const DIFFICULTY_RANK: Record<string, number> = { easy: 0, medium: 1, hard: 2 };

const GRID = '44px 78px minmax(220px,1fr) 220px 92px 100px 90px';

function selectStyle(extra?: React.CSSProperties): React.CSSProperties {
  return {
    height: 34,
    borderRadius: 8,
    border: '1px solid var(--border)',
    background: 'var(--surface2)',
    padding: '0 8px',
    fontSize: 12.5,
    color: 'var(--text)',
    cursor: 'pointer',
    ...extra,
  };
}

/** Round status chip: ✓ solved, • attempted, empty ring otherwise. */
function StatusChip({ status }: { status: StatusFilter }) {
  const meta =
    status === 'solved'
      ? { color: 'var(--success)', icon: '✓', label: 'Solved', size: '11px' }
      : status === 'attempted'
        ? { color: 'var(--warn)', icon: '•', label: 'Attempted', size: '14px' }
        : { color: 'var(--border2)', icon: '', label: 'Not attempted', size: '11px' };

  return (
    <span
      role="img"
      aria-label={meta.label}
      title={meta.label}
      style={{
        width: 22,
        height: 22,
        borderRadius: '50%',
        border: `1.5px solid ${meta.color}`,
        color: meta.color,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: meta.size,
        fontWeight: 700,
        flexShrink: 0,
      }}
    >
      {meta.icon}
    </span>
  );
}

function ProblemsScreen() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user } = useAuth();
  const width = useViewportWidth();
  const isMobile = width < 760;

  const [search, setSearch] = useState(searchParams.get('search') ?? '');
  const [difficulty, setDifficulty] = useState<Difficulty | ''>(
    (searchParams.get('difficulty') as Difficulty) || '',
  );
  const [tagSlug, setTagSlug] = useState(searchParams.get('tag_slug') ?? '');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sort, setSort] = useState<SortOption>('default');
  const [page, setPage] = useState(Number(searchParams.get('page')) || 1);

  const [problems, setProblems] = useState<Problem[]>([]);
  const [total, setTotal] = useState(0);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nonce, setNonce] = useState(0);

  const debouncedSearch = useDebounced(search, 350);
  const { progress } = useProgress(Boolean(user));

  // Keep the URL shareable without re-triggering the fetch effect.
  useEffect(() => {
    const params = new URLSearchParams();
    if (debouncedSearch) params.set('search', debouncedSearch);
    if (difficulty) params.set('difficulty', difficulty);
    if (tagSlug) params.set('tag_slug', tagSlug);
    if (page > 1) params.set('page', String(page));
    const qs = params.toString();
    router.replace(qs ? `/problems?${qs}` : '/problems', { scroll: false });
  }, [debouncedSearch, difficulty, tagSlug, page, router]);

  useEffect(() => {
    const controller = new AbortController();
    problemApi
      .tags(controller.signal)
      .then((res) => setTags(res.items ?? []))
      .catch(() => setTags([]));
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError(null);

    problemApi
      .list(
        { page, limit: PAGE_SIZE, difficulty, search: debouncedSearch, tag_slug: tagSlug },
        controller.signal,
      )
      .then((res) => {
        setProblems(res.items ?? []);
        setTotal(res.total ?? 0);
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        if (err instanceof NetworkError) {
          setError('Cannot reach the API gateway');
        } else if (err instanceof ApiError) {
          setError(`GET /api/v1/problems — ${err.httpStatus} ${err.message}`);
        } else {
          setError('Unexpected error');
        }
        setProblems([]);
        setTotal(0);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });

    return () => controller.abort();
  }, [page, difficulty, debouncedSearch, tagSlug, nonce]);

  const statusOf = useCallback(
    (problem: Problem): StatusFilter => {
      if (progress.solvedIds.has(problem.id)) return 'solved';
      if (progress.attemptedIds.has(problem.id)) return 'attempted';
      return 'unsolved';
    },
    [progress],
  );

  // Status and sort are client-side: the list endpoint offers neither.
  const rows = useMemo(() => {
    let list = problems;
    if (statusFilter !== 'all') list = list.filter((p) => statusOf(p) === statusFilter);

    const sorted = [...list];
    switch (sort) {
      case 'title-asc':
        sorted.sort((a, b) => a.title.localeCompare(b.title));
        break;
      case 'difficulty-asc':
        sorted.sort((a, b) => DIFFICULTY_RANK[a.difficulty] - DIFFICULTY_RANK[b.difficulty]);
        break;
      case 'difficulty-desc':
        sorted.sort((a, b) => DIFFICULTY_RANK[b.difficulty] - DIFFICULTY_RANK[a.difficulty]);
        break;
      case 'newest':
        sorted.sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at));
        break;
      default:
        break;
    }
    return sorted;
  }, [problems, statusFilter, statusOf, sort]);

  const filtersActive = Boolean(search || difficulty || tagSlug || statusFilter !== 'all' || sort !== 'default');
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const solvedHere = problems.filter((p) => progress.solvedIds.has(p.id)).length;

  const resetFilters = () => {
    setSearch('');
    setDifficulty('');
    setTagSlug('');
    setStatusFilter('all');
    setSort('default');
    setPage(1);
  };

  return (
    <AppShell>
      <PageHeading
        title="Problems"
        subtitle={`Browse ${total} problem${total === 1 ? '' : 's'}${tags.length ? ` across ${tags.length} topics` : ''}`}
        actions={
          user ? (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 11,
                background: 'var(--surface)',
                border: '1px solid var(--border)',
                borderRadius: 12,
                padding: '8px 16px 8px 10px',
              }}
            >
              <svg width="30" height="30" viewBox="0 0 26 26" role="img" aria-label="Solved progress">
                <circle cx="13" cy="13" r="11" fill="none" stroke="var(--surface3)" strokeWidth="3" />
                <circle
                  cx="13"
                  cy="13"
                  r="11"
                  fill="none"
                  stroke="var(--accent)"
                  strokeWidth="3"
                  strokeLinecap="round"
                  strokeDasharray={69}
                  strokeDashoffset={
                    69 - (69 * (total ? progress.solvedIds.size / total : 0))
                  }
                  transform="rotate(-90 13 13)"
                />
              </svg>
              <span style={{ display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
                <span style={{ fontSize: 13, fontWeight: 650, whiteSpace: 'nowrap' }}>
                  <span style={{ fontFamily: 'var(--font-mono)' }}>{progress.solvedIds.size}</span> of{' '}
                  <span style={{ fontFamily: 'var(--font-mono)' }}>{total}</span> solved
                </span>
                <span style={{ fontSize: 11.5, color: 'var(--text3)', whiteSpace: 'nowrap' }}>
                  {total ? Math.round((progress.solvedIds.size / total) * 100) : 0}% complete
                </span>
              </span>
            </div>
          ) : null
        }
      />

      <section
        aria-label="Problem catalog"
        style={{
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: 14,
          boxShadow: 'var(--shadow)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 14,
            padding: '14px 16px',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <div
            style={{
              position: 'relative',
              display: 'flex',
              alignItems: 'center',
              flex: 1,
              minWidth: 180,
              maxWidth: 300,
            }}
          >
            <span style={{ position: 'absolute', left: 10, display: 'flex' }}>
              <Icon.Search size={14} color="var(--text3)" />
            </span>
            <input
              type="search"
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
                setPage(1);
              }}
              placeholder="Search by title or slug…"
              aria-label="Search problems by title or slug"
              className="ac-input"
              style={{
                height: 34,
                width: '100%',
                boxSizing: 'border-box',
                borderRadius: 8,
                border: '1px solid var(--border)',
                background: 'var(--surface2)',
                padding: '0 10px 0 31px',
                fontSize: 13,
                color: 'var(--text)',
              }}
            />
          </div>

          <div
            role="group"
            aria-label="Filters"
            style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 8 }}
          >
            <select
              value={difficulty}
              onChange={(event) => {
                setDifficulty(event.target.value as Difficulty | '');
                setPage(1);
              }}
              aria-label="Filter by difficulty"
              style={selectStyle()}
            >
              <option value="">Difficulty: All</option>
              {DIFFICULTIES.map((d) => (
                <option key={d} value={d}>
                  {difficultyMeta(d).label}
                </option>
              ))}
            </select>

            <select
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
              aria-label="Filter by status"
              disabled={!user}
              title={user ? undefined : 'Sign in to filter by your progress'}
              style={selectStyle({ opacity: user ? 1 : 0.55 })}
            >
              <option value="all">Status: All</option>
              <option value="solved">Solved</option>
              <option value="attempted">Attempted</option>
              <option value="unsolved">Unsolved</option>
            </select>

            <select
              value={tagSlug}
              onChange={(event) => {
                setTagSlug(event.target.value);
                setPage(1);
              }}
              aria-label="Filter by tag"
              style={selectStyle({ maxWidth: 170 })}
            >
              <option value="">Tag: All</option>
              {tags.map((tag) => (
                <option key={tag.id} value={tag.slug}>
                  {tag.name}
                </option>
              ))}
            </select>

            <select
              value={sort}
              onChange={(event) => setSort(event.target.value as SortOption)}
              aria-label="Sort problems"
              style={selectStyle()}
            >
              <option value="default">Sort: Default</option>
              <option value="title-asc">Title A–Z</option>
              <option value="difficulty-asc">Difficulty ↑</option>
              <option value="difficulty-desc">Difficulty ↓</option>
              <option value="newest">Newest</option>
            </select>

            {filtersActive && (
              <button
                type="button"
                onClick={resetFilters}
                className="ac-hover-accent-soft2"
                style={{
                  height: 34,
                  padding: '0 12px',
                  border: 'none',
                  borderRadius: 8,
                  background: 'var(--accent-soft)',
                  color: 'var(--accent-fg)',
                  fontSize: 12.5,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Clear filters
              </button>
            )}
          </div>

          <span
            aria-live="polite"
            style={{
              marginLeft: 'auto',
              fontSize: 11.5,
              color: 'var(--text3)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {loading ? 'loading…' : `${rows.length} shown · ${total} total`}
          </span>
        </div>

        {loading && (
          <div role="status" aria-label="Loading problems" style={{ padding: '4px 0' }}>
            {Array.from({ length: 8 }, (_, i) => (
              <div
                key={i}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 14,
                  padding: '13px 18px',
                  borderTop: '1px solid var(--border)',
                }}
              >
                <span
                  className="ac-skeleton"
                  style={{ width: 18, height: 18, borderRadius: '50%' }}
                />
                <span
                  className="ac-skeleton"
                  style={{ height: 12, borderRadius: 6, width: `${34 + ((i * 7) % 30)}%` }}
                />
                <span
                  className="ac-skeleton"
                  style={{ height: 12, borderRadius: 6, width: 70, marginLeft: 'auto' }}
                />
              </div>
            ))}
          </div>
        )}

        {!loading && error && (
          <ErrorState
            title="Couldn't load problems"
            detail={error}
            onRetry={() => setNonce((n) => n + 1)}
          />
        )}

        {!loading && !error && rows.length === 0 && (
          <EmptyState
            title="No problems match these filters"
            description="Try a broader search or clear the filters."
            action={
              <button
                type="button"
                onClick={resetFilters}
                className="ac-hover-accent"
                style={{
                  height: 36,
                  padding: '0 16px',
                  border: 'none',
                  borderRadius: 8,
                  background: 'var(--accent)',
                  color: 'var(--accent-ink)',
                  fontSize: 13,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                Reset filters
              </button>
            }
          />
        )}

        {!loading && !error && rows.length > 0 && !isMobile && (
          <div style={{ overflowX: 'auto' }}>
            <div role="table" aria-label="Problems" style={{ minWidth: 800, animation: 'acFadeUp .25s ease' }}>
              <div
                role="row"
                style={{
                  display: 'grid',
                  gridTemplateColumns: GRID,
                  gap: 10,
                  alignItems: 'center',
                  padding: '9px 18px',
                  fontSize: 11,
                  fontWeight: 650,
                  letterSpacing: '.06em',
                  textTransform: 'uppercase',
                  color: 'var(--text3)',
                }}
              >
                <span role="columnheader" aria-label="Status" />
                <span role="columnheader">ID</span>
                <span role="columnheader">Title</span>
                <span role="columnheader">Tags</span>
                <span role="columnheader">Difficulty</span>
                <span role="columnheader" style={{ textAlign: 'right' }}>
                  Time limit
                </span>
                <span role="columnheader" style={{ textAlign: 'right' }}>
                  Memory
                </span>
              </div>

              {rows.map((problem) => {
                const status = statusOf(problem);
                const diff = difficultyMeta(problem.difficulty);
                const shownTags = (problem.tags ?? []).slice(0, 2);
                const extra = (problem.tags?.length ?? 0) - shownTags.length;

                return (
                  <Link
                    key={problem.id}
                    role="row"
                    href={`/problems/${problem.slug}`}
                    data-prow="1"
                    className="ac-hover-accent-row"
                    style={{
                      display: 'grid',
                      gridTemplateColumns: GRID,
                      gap: 10,
                      alignItems: 'center',
                      width: '100%',
                      boxSizing: 'border-box',
                      padding: 'var(--rowpad) 18px',
                      borderTop: '1px solid var(--border)',
                      background: status === 'solved' ? 'var(--surface2)' : 'transparent',
                      textAlign: 'left',
                      color: 'var(--text)',
                      textDecoration: 'none',
                      transition: 'background .12s',
                    }}
                  >
                    <StatusChip status={status} />
                    <span
                      style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--text3)' }}
                    >
                      #{problem.id}
                    </span>
                    <span
                      data-ptitle="1"
                      style={{
                        fontSize: 13.5,
                        fontWeight: 550,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        transition: 'color .12s',
                      }}
                    >
                      {problem.title}
                    </span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                      {shownTags.map((tag) => (
                        <span
                          key={tag.id}
                          style={{
                            fontSize: 12,
                            fontWeight: 500,
                            color: 'var(--text)',
                            background: 'var(--surface2)',
                            border: '1px solid var(--border2)',
                            borderRadius: 5,
                            padding: '1px 8px',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {tag.name}
                        </span>
                      ))}
                      {extra > 0 && (
                        <span
                          title={problem.tags?.slice(2).map((t) => t.name).join(', ')}
                          style={{
                            fontSize: 12,
                            fontWeight: 600,
                            color: 'var(--text2)',
                            background: 'var(--surface3)',
                            border: '1px solid var(--border2)',
                            borderRadius: 5,
                            padding: '1px 7px',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          +{extra}
                        </span>
                      )}
                    </span>
                    <span
                      style={{
                        fontSize: 11.5,
                        fontWeight: 600,
                        color: diff.color,
                        background: diff.bg,
                        borderRadius: 6,
                        padding: '2px 8px',
                        justifySelf: 'start',
                      }}
                    >
                      {diff.label}
                    </span>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        color: 'var(--text2)',
                        textAlign: 'right',
                      }}
                    >
                      {formatTimeLimit(problem.time_limit)}
                    </span>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        color: 'var(--text3)',
                        textAlign: 'right',
                      }}
                    >
                      {formatMemoryLimit(problem.memory_limit)}
                    </span>
                  </Link>
                );
              })}
            </div>
          </div>
        )}

        {!loading && !error && rows.length > 0 && isMobile && (
          <div aria-label="Problems" style={{ display: 'flex', flexDirection: 'column' }}>
            {rows.map((problem) => {
              const diff = difficultyMeta(problem.difficulty);
              return (
                <Link
                  key={problem.id}
                  href={`/problems/${problem.slug}`}
                  data-prow="1"
                  className="ac-hover-accent-row"
                  style={{
                    display: 'flex',
                    gap: 12,
                    alignItems: 'flex-start',
                    minHeight: 44,
                    padding: '12px 16px',
                    borderTop: '1px solid var(--border)',
                    color: 'var(--text)',
                    textDecoration: 'none',
                  }}
                >
                  <span style={{ marginTop: 2 }}>
                    <StatusChip status={statusOf(problem)} />
                  </span>
                  <span style={{ flex: 1, minWidth: 0 }}>
                    <span
                      data-ptitle="1"
                      style={{ display: 'block', fontSize: 13.5, fontWeight: 550 }}
                    >
                      {problem.title}
                    </span>
                    <span
                      style={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        alignItems: 'center',
                        gap: 6,
                        marginTop: 5,
                      }}
                    >
                      <span
                        style={{
                          fontSize: 11,
                          fontWeight: 600,
                          color: diff.color,
                          background: diff.bg,
                          borderRadius: 5,
                          padding: '1px 7px',
                        }}
                      >
                        {diff.label}
                      </span>
                      <span
                        style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}
                      >
                        #{problem.id} · {formatTimeLimit(problem.time_limit)}
                      </span>
                    </span>
                  </span>
                </Link>
              );
            })}
          </div>
        )}

        {!error && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 10,
              padding: '12px 18px',
              borderTop: '1px solid var(--border)',
            }}
          >
            <span style={{ fontSize: 12, color: 'var(--text3)' }}>
              Page {page} of {totalPages}
              {user && problems.length > 0 ? ` · ${solvedHere} solved on this page` : ''}
            </span>
            <div style={{ display: 'flex', gap: 6 }}>
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                aria-label="Previous page"
                className="ac-hover-surface2"
                style={{
                  height: 32,
                  padding: '0 12px',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  background: 'var(--surface)',
                  color: 'var(--text2)',
                  fontSize: 12.5,
                  fontWeight: 600,
                  cursor: page <= 1 ? 'not-allowed' : 'pointer',
                  opacity: page <= 1 ? 0.45 : 1,
                }}
              >
                ← Prev
              </button>
              <button
                type="button"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                aria-label="Next page"
                className="ac-hover-surface2"
                style={{
                  height: 32,
                  padding: '0 12px',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  background: 'var(--surface)',
                  color: 'var(--text2)',
                  fontSize: 12.5,
                  fontWeight: 600,
                  cursor: page >= totalPages ? 'not-allowed' : 'pointer',
                  opacity: page >= totalPages ? 0.45 : 1,
                }}
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

export default function ProblemsPage() {
  return (
    <Suspense fallback={null}>
      <ProblemsScreen />
    </Suspense>
  );
}
