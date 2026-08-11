'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { AdminPagination, AdminTableShell, AdminUserStatusPill, DateText, RolePill, adminTd, adminTh } from '@/components/admin/AdminData';
import { AdminApiError, AdminLoadingState, AdminPageHeader, adminCard, adminField } from '@/components/admin/AdminStates';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminUserApi } from '@/lib/api';
import { useViewportWidth } from '@/lib/hooks';
import type { AdminUser, AdminUserRole, AdminUserStatus } from '@/lib/types';

const PAGE_SIZE = 20;
const ROLES: AdminUserRole[] = ['user', 'contributor', 'moderator', 'admin'];
const STATUSES: AdminUserStatus[] = ['active', 'unverified', 'suspended'];

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

function parsePage(value: string | null) {
  const page = Number(value);
  return Number.isInteger(page) && page > 0 ? page : 1;
}

function parseRole(value: string | null): AdminUserRole | '' {
  return ROLES.includes(value as AdminUserRole) ? (value as AdminUserRole) : '';
}

function parseStatus(value: string | null): AdminUserStatus | '' {
  return STATUSES.includes(value as AdminUserStatus) ? (value as AdminUserStatus) : '';
}

function userInitials(user: AdminUser) {
  const source = user.full_name || user.username;
  return source
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

function UserAvatar({ user, size = 34 }: { user: AdminUser; size?: number }) {
  return (
    <span
      aria-hidden="true"
      style={{
        position: 'relative',
        width: size,
        height: size,
        borderRadius: '50%',
        background: 'var(--accent-soft2)',
        color: 'var(--accent-fg)',
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: Math.max(10, Math.round(size * 0.32)),
        fontWeight: 750,
        flexShrink: 0,
        overflow: 'hidden',
      }}
    >
      {userInitials(user)}
      {user.avatar_url && (
        <img
          src={user.avatar_url}
          alt=""
          onError={(event) => {
            event.currentTarget.style.display = 'none';
          }}
          style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', background: 'var(--surface2)' }}
        />
      )}
    </span>
  );
}

export default function AdminUsersPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const viewportWidth = useViewportWidth();
  const compact = viewportWidth < 860;
  const requestSeq = useRef(0);
  const [items, setItems] = useState<AdminUser[]>([]);
  const [pagination, setPagination] = useState({ page: 1, limit: PAGE_SIZE, total: 0, total_pages: 0 });
  const [searchDraft, setSearchDraft] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const filters = useMemo(
    () => ({
      page: parsePage(searchParams.get('page')),
      search: searchParams.get('search')?.trim() ?? '',
      role: parseRole(searchParams.get('role')),
      status: parseStatus(searchParams.get('status')),
    }),
    [searchParams],
  );

  useEffect(() => {
    setSearchDraft(filters.search);
  }, [filters.search]);

  const replaceFilters = useCallback(
    (nextValues: Partial<typeof filters>) => {
      const next = new URLSearchParams(searchParams.toString());
      next.delete('limit');
      const merged = { ...filters, ...nextValues };
      const write = (key: keyof typeof merged, value: string | number) => {
        if (value === '' || value === 1) next.delete(key);
        else next.set(key, String(value));
      };
      write('page', merged.page);
      write('search', merged.search);
      write('role', merged.role);
      write('status', merged.status);
      const query = next.toString();
      router.replace(query ? `/admin/users?${query}` : '/admin/users', { scroll: false });
    },
    [filters, router, searchParams],
  );

  const load = useCallback(
    (signal?: AbortSignal) => {
      const sequence = requestSeq.current + 1;
      requestSeq.current = sequence;
      setLoading(true);
      setError('');
      adminUserApi
        .list({ page: filters.page, limit: PAGE_SIZE, search: filters.search, role: filters.role, status: filters.status }, signal)
        .then((response) => {
          if (signal?.aborted || requestSeq.current !== sequence) return;
          setItems(response.items ?? []);
          setPagination(response.pagination);
        })
        .catch((err) => {
          if (!signal?.aborted && requestSeq.current === sequence) setError(errorMessage(err));
        })
        .finally(() => {
          if (!signal?.aborted && requestSeq.current === sequence) setLoading(false);
        });
    },
    [filters],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const applySearch = (event: FormEvent) => {
    event.preventDefault();
    replaceFilters({ page: 1, search: searchDraft.trim() });
  };

  const hasFilters = Boolean(filters.search || filters.role || filters.status);
  const totalPages = pagination.total_pages || (pagination.total > 0 ? Math.ceil(pagination.total / pagination.limit) : 0);
  const outOfRangePage = pagination.total > 0 && filters.page > totalPages;

  if (loading && items.length === 0) {
    return (
      <>
        <AdminPageHeader title="Users" description="Manage user accounts, roles, and account access." />
        <AdminLoadingState title="Loading users" />
      </>
    );
  }

  if (error && items.length === 0) {
    return (
      <>
        <AdminPageHeader title="Users" description="Manage user accounts, roles, and account access." />
        <AdminApiError title="Could not load users" error={error} onRetry={() => load()} />
      </>
    );
  }

  return (
    <>
      <AdminPageHeader title="Users" description="Manage user accounts, roles, and account access." />

      <form
        onSubmit={applySearch}
        style={{ ...adminCard, padding: 12, marginBottom: 12, display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}
      >
        <input
          type="search"
          aria-label="Search users"
          value={searchDraft}
          onChange={(event) => setSearchDraft(event.target.value)}
          placeholder="Search username, email, or name"
          className="ac-input"
          style={{ ...adminField, minWidth: 220, flex: '1 1 290px' }}
        />
        {searchDraft && (
          <button
            type="button"
            onClick={() => {
              setSearchDraft('');
              replaceFilters({ page: 1, search: '' });
            }}
            className="ac-hover-surface2"
            style={buttonStyles.secondary(38)}
          >
            Clear
          </button>
        )}
        <select
          aria-label="Role filter"
          value={filters.role}
          onChange={(event) => replaceFilters({ page: 1, role: event.target.value as AdminUserRole | '' })}
          className="ac-field"
          style={{ ...adminField, minWidth: 145 }}
        >
          <option value="">All roles</option>
          {ROLES.map((role) => (
            <option key={role} value={role}>{role[0].toUpperCase() + role.slice(1)}</option>
          ))}
        </select>
        <select
          aria-label="Account status filter"
          value={filters.status}
          onChange={(event) => replaceFilters({ page: 1, status: event.target.value as AdminUserStatus | '' })}
          className="ac-field"
          style={{ ...adminField, minWidth: 155 }}
        >
          <option value="">All statuses</option>
          {STATUSES.map((status) => (
            <option key={status} value={status}>{status[0].toUpperCase() + status.slice(1)}</option>
          ))}
        </select>
        <button type="submit" className="ac-hover-accent" style={buttonStyles.primary(38)}>
          Search
        </button>
      </form>

      {error && (
        <div role="alert" style={{ ...adminCard, borderColor: 'var(--error)', padding: 12, marginBottom: 12, display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}>
          <span style={{ color: 'var(--error)', fontWeight: 700 }}>Could not refresh users.</span>
          <span style={{ color: 'var(--text2)', fontSize: 13, flex: 1 }}>{error}</span>
          <button type="button" onClick={() => load()} className="ac-hover-surface2" style={buttonStyles.secondary(34)}>Retry</button>
        </div>
      )}

      {items.length === 0 ? (
        <section style={{ ...adminCard, padding: '42px 20px', textAlign: 'center' }}>
          <h2 style={{ margin: '0 0 6px', fontSize: 16, fontWeight: 700 }}>{outOfRangePage ? 'This page is out of range' : hasFilters ? 'No users match these filters' : 'No users found'}</h2>
          <p style={{ margin: '0 auto 18px', maxWidth: 440, color: 'var(--text2)', fontSize: 13.5 }}>
            {outOfRangePage ? `Page ${filters.page} does not contain users. Return to the first page or choose a previous page below.` : hasFilters ? 'Try a different search, role, or account status.' : 'The Auth service did not return any user accounts.'}
          </p>
          {outOfRangePage ? (
            <button type="button" onClick={() => replaceFilters({ page: 1 })} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
              Return to first page
            </button>
          ) : hasFilters && (
            <button type="button" onClick={() => replaceFilters({ page: 1, search: '', role: '', status: '' })} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
              Reset filters
            </button>
          )}
        </section>
      ) : compact ? (
        <div aria-busy={loading} style={{ display: 'grid', gap: 10 }}>
          {items.map((user) => (
            <article key={user.id} style={{ ...adminCard, padding: 14, display: 'grid', gap: 12 }}>
              <div style={{ display: 'flex', gap: 10, alignItems: 'center', minWidth: 0 }}>
                <UserAvatar user={user} />
                <div style={{ minWidth: 0, flex: 1 }}>
                  <Link href={`/admin/users/${user.id}`} style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 700 }}>
                    @{user.username}
                  </Link>
                  {user.full_name && <div style={{ color: 'var(--text2)', fontSize: 12.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{user.full_name}</div>}
                  <div title={user.email} style={{ color: 'var(--text3)', fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{user.email}</div>
                </div>
                <AdminUserStatusPill user={user} />
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
                <RolePill role={user.role} />
                <span style={{ color: 'var(--text2)', fontFamily: 'var(--font-mono)', fontSize: 12 }}>Rating {user.rating.toLocaleString()}</span>
                <span style={{ marginLeft: 'auto' }}><Link href={`/admin/users/${user.id}`} className="ac-hover-surface2" style={buttonStyles.secondary(32)}>View</Link></span>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <AdminTableShell>
          <table aria-busy={loading} style={{ width: '100%', minWidth: 900, borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={adminTh}>User</th>
                <th style={adminTh}>Email</th>
                <th style={adminTh}>Role</th>
                <th style={adminTh}>Status</th>
                <th style={adminTh}>Rating</th>
                <th style={adminTh}>Joined</th>
                <th style={{ ...adminTh, textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((user) => (
                <tr key={user.id}>
                  <td style={adminTd}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 190 }}>
                      <UserAvatar user={user} />
                      <div style={{ minWidth: 0 }}>
                        <Link href={`/admin/users/${user.id}`} style={{ display: 'block', fontWeight: 700 }}>@{user.username}</Link>
                        {user.full_name && <div style={{ color: 'var(--text3)', fontSize: 11.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 210 }}>{user.full_name}</div>}
                      </div>
                    </div>
                  </td>
                  <td style={{ ...adminTd, color: 'var(--text2)' }}>{user.email}</td>
                  <td style={adminTd}><RolePill role={user.role} /></td>
                  <td style={adminTd}><AdminUserStatusPill user={user} /></td>
                  <td style={{ ...adminTd, fontFamily: 'var(--font-mono)' }}>{user.rating.toLocaleString()}</td>
                  <td style={adminTd}><DateText value={user.created_at} /></td>
                  <td style={{ ...adminTd, textAlign: 'right' }}>
                    <Link href={`/admin/users/${user.id}`} className="ac-hover-surface2" style={buttonStyles.secondary(32)}>View</Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </AdminTableShell>
      )}

      {(items.length > 0 || pagination.total > 0) && <AdminPagination page={filters.page} totalPages={totalPages} total={pagination.total} onPage={(page) => replaceFilters({ page })} />}
    </>
  );
}
