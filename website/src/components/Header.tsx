'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { Fragment, useCallback, useEffect, useRef, useState } from 'react';
import { useAuth } from './AuthProvider';
import { useTheme } from './ThemeProvider';
import { useToast } from './ToastProvider';
import { API_BASE_URL, problemApi, userApi } from '@/lib/api';
import { useDebounced, useDismissable, useViewportWidth } from '@/lib/hooks';
import { avatarUrl, difficultyMeta, initials } from '@/lib/format';
import type { Me, Problem, PublicUserSearchItem } from '@/lib/types';
import { Icon, Logo, Wordmark, buttonStyles } from './ui';
import { ADMIN_CONSOLE_MIN_ROLE } from './admin/AdminNavigation';
import { isContributorWorkspaceUser, roleAtLeast } from './admin/roles';

const NAV_ITEMS = [
  { label: 'Problems', href: '/problems' },
  { label: 'Contests', href: '/contests' },
  { label: 'Leaderboard', href: '/leaderboard' },
  { label: 'Discussions', href: '/discuss' },
];

type GlobalSearchResult =
  | { key: string; kind: 'problem'; problem: Problem }
  | { key: string; kind: 'user'; user: PublicUserSearchItem };

export function Header() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const { resolved, toggle } = useTheme();
  const { showToast } = useToast();
  const width = useViewportWidth();
  const isContributorWorkspace = isContributorWorkspaceUser(user?.role);
  const canAccessAdminConsole = roleAtLeast(user?.role, ADMIN_CONSOLE_MIN_ROLE);
  const navItems = [
    ...NAV_ITEMS,
    ...(isContributorWorkspace ? [{ label: 'Contributions', href: '/contributions' }] : []),
    ...(canAccessAdminConsole ? [{ label: 'Admin Console', href: '/admin' }] : []),
  ];
  // The desktop navigation, search, and account controls no longer fit reliably
  // at tablet widths. Privileged navigation needs one extra item, so it moves
  // to the compact menu slightly earlier to prevent page-level overflow.
  const isMobile = width < (canAccessAdminConsole ? 1240 : isContributorWorkspace ? 1060 : 960);

  const [menu, setMenu] = useState<'notif' | 'user' | 'mobile' | null>(null);
  const [query, setQuery] = useState('');
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchLoading, setSearchLoading] = useState(false);
  const [searchError, setSearchError] = useState(false);
  const [searchResults, setSearchResults] = useState<GlobalSearchResult[]>([]);
  const [activeSearchIndex, setActiveSearchIndex] = useState(-1);
  const debouncedQuery = useDebounced(query, 250);
  const latestSearchQuery = useRef('');

  const close = useCallback(() => setMenu(null), []);
  const closeSearch = useCallback(() => {
    setSearchOpen(false);
    setActiveSearchIndex(-1);
  }, []);
  const notifRef = useDismissable<HTMLDivElement>(menu === 'notif', close);
  const userRef = useDismissable<HTMLDivElement>(menu === 'user', close);
  const searchRef = useDismissable<HTMLDivElement>(searchOpen, closeSearch);

  // ⌘K / Ctrl+K focuses search, as advertised by the kbd hint.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        document.getElementById('ac-global-search')?.focus();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  useEffect(() => {
    setMenu(null);
    closeSearch();
  }, [pathname, closeSearch]);

  useEffect(() => {
    if (menu !== 'user') return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMenu(null);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [menu]);

  useEffect(() => {
    const trimmed = debouncedQuery.trim();
    if (trimmed.length < 2) {
      setSearchLoading(false);
      setSearchError(false);
      setSearchResults([]);
      return;
    }

    const controller = new AbortController();
    setSearchOpen(true);
    setSearchLoading(true);
    setSearchError(false);
    setSearchResults([]);
    setActiveSearchIndex(-1);

    void Promise.allSettled([
      problemApi.list({ search: trimmed, page: 1, limit: 5 }, controller.signal),
      userApi.searchUsers({ q: trimmed, page: 1, limit: 5 }, controller.signal),
    ])
      .then(([problems, users]) => {
        if (controller.signal.aborted || latestSearchQuery.current !== trimmed) return;
        if (problems.status === 'rejected' && users.status === 'rejected') {
          setSearchError(true);
          return;
        }
        const next: GlobalSearchResult[] = [];
        if (problems.status === 'fulfilled') {
          next.push(...problems.value.items.map((problem) => ({
            key: `problem-${problem.id}`,
            kind: 'problem' as const,
            problem,
          })));
        }
        if (users.status === 'fulfilled') {
          next.push(...users.value.items.map((user) => ({
            key: `user-${user.username}`,
            kind: 'user' as const,
            user,
          })));
        }
        setSearchResults(next);
      })
      .finally(() => {
        if (!controller.signal.aborted && latestSearchQuery.current === trimmed) setSearchLoading(false);
      });

    return () => controller.abort();
  }, [debouncedQuery]);

  const onSearchChange = (value: string) => {
    const trimmed = value.trim();
    latestSearchQuery.current = trimmed;
    setQuery(value);
    setActiveSearchIndex(-1);
    setSearchResults([]);
    setSearchError(false);
    if (trimmed.length < 2) {
      setSearchLoading(false);
      setSearchOpen(false);
    } else {
      setSearchLoading(true);
      setSearchOpen(true);
    }
  };

  const navigateToSearchResult = (result: GlobalSearchResult) => {
    closeSearch();
    latestSearchQuery.current = '';
    setQuery('');
    setSearchResults([]);
    router.push(
      result.kind === 'problem'
        ? `/problems/${encodeURIComponent(result.problem.slug)}`
        : `/u/${encodeURIComponent(result.user.username)}`,
    );
  };

  const submitSearch = (event: React.FormEvent) => {
    event.preventDefault();
    if (activeSearchIndex >= 0 && activeSearchIndex < searchResults.length) {
      navigateToSearchResult(searchResults[activeSearchIndex]);
    }
  };

  const onSearchKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      closeSearch();
      return;
    }
    if (searchResults.length === 0) return;
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setSearchOpen(true);
      setActiveSearchIndex((index) => (index + 1) % searchResults.length);
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setSearchOpen(true);
      setActiveSearchIndex((index) => (index <= 0 ? searchResults.length - 1 : index - 1));
    }
  };

  const onSignOut = async () => {
    await logout();
    showToast('Signed out', 'info');
    router.push('/login');
  };

  const isCurrent = (href: string) => pathname === href || pathname.startsWith(`${href}/`);

  return (
    <header className="ac-site-header">
      <div
        className="ac-site-header-inner"
        style={{
          gap: 8,
        }}
      >
        <Link
          href="/"
          aria-label="AstraCode home"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 9,
            color: 'var(--text)',
            textDecoration: 'none',
            marginRight: 20,
            flexShrink: 0,
          }}
        >
          <Logo />
          <Wordmark />
        </Link>

        {!isMobile && (
          <nav aria-label="Primary" style={{ display: 'flex', alignItems: 'center', gap: 4, height: '100%' }}>
            {navItems.map((item) => {
              const current = isCurrent(item.href);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  aria-current={current ? 'page' : undefined}
                  className="ac-hover-text"
                  style={{
                    position: 'relative',
                    height: 56,
                    padding: '0 13px',
                    textDecoration: 'none',
                    fontSize: 13.5,
                    fontWeight: current ? 600 : 500,
                    color: current ? 'var(--text)' : 'var(--text2)',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 3,
                  }}
                >
                  <span>{item.label}</span>
                  <span
                    style={{
                      width: 20,
                      height: 2,
                      borderRadius: 2,
                      background: current ? 'var(--accent)' : 'transparent',
                      transition: 'background .2s',
                    }}
                  />
                </Link>
              );
            })}
          </nav>
        )}

        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 10 }}>
          {!isMobile && (
            <div ref={searchRef}>
              <UserSearch
                query={query}
                open={searchOpen && query.trim().length >= 2}
                loading={searchLoading}
                failed={searchError}
                results={searchResults}
                activeIndex={activeSearchIndex}
                onChange={onSearchChange}
                onFocus={() => query.trim().length >= 2 && setSearchOpen(true)}
                onKeyDown={onSearchKeyDown}
                onSubmit={submitSearch}
                onActivate={setActiveSearchIndex}
                onSelect={navigateToSearchResult}
              />
            </div>
          )}

          <button
            type="button"
            onClick={toggle}
            aria-label={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            title={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            className="ac-hover-surface2-text"
            style={buttonStyles.iconButton(isMobile ? 44 : 36)}
          >
            {resolved === 'dark' ? <Icon.Sun /> : <Icon.Moon />}
          </button>

          <div style={{ position: 'relative' }} ref={notifRef}>
            <button
              type="button"
              onClick={() => setMenu(menu === 'notif' ? null : 'notif')}
              aria-label="Notifications"
              aria-expanded={menu === 'notif'}
              className="ac-hover-surface2-text"
              style={buttonStyles.iconButton(isMobile ? 44 : 36)}
            >
              <Icon.Bell />
            </button>
            {menu === 'notif' && (
              <div
                role="menu"
                aria-label="Notifications"
                style={{
                  position: 'absolute',
                  right: 0,
                  top: 44,
                  width: 320,
                  background: 'var(--surface)',
                  border: '1px solid var(--border)',
                  borderRadius: 12,
                  boxShadow: 'var(--shadow-lg)',
                  padding: 6,
                  animation: 'acPop .15s ease',
                }}
              >
                <div style={{ padding: '8px 10px 6px' }}>
                  <span style={{ fontSize: 13, fontWeight: 600 }}>Notifications</span>
                </div>
                <p style={{ margin: 0, padding: '4px 10px 12px', fontSize: 12.5, color: 'var(--text3)' }}>
                  No notifications yet.
                </p>
              </div>
            )}
          </div>

          {user ? (
            <div style={{ position: 'relative' }} ref={userRef}>
              <button
                type="button"
                onClick={() => setMenu(menu === 'user' ? null : 'user')}
                aria-label={`Account menu for ${user.username}`}
                aria-expanded={menu === 'user'}
                aria-haspopup="menu"
                className="ac-hover-surface2"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  height: isMobile ? 44 : 36,
                  padding: '0 8px 0 4px',
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: 'var(--surface)',
                  cursor: 'pointer',
                  color: 'var(--text)',
                }}
              >
                <AccountAvatar user={user} size={28} />
                <Icon.Chevron color="var(--text3)" />
              </button>

              {menu === 'user' && (
                <div
                  role="menu"
                  aria-label="Account"
                  style={{
                    position: 'absolute',
                    right: 0,
                    top: 44,
                    width: 224,
                    background: 'var(--surface)',
                    border: '1px solid var(--border)',
                    borderRadius: 12,
                    boxShadow: 'var(--shadow-lg)',
                    padding: 6,
                    animation: 'acPop .15s ease',
                  }}
                >
                  <Link
                    role="menuitem"
                    href="/profile"
                    onClick={() => setMenu(null)}
                    className="ac-hover-surface2"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 10,
                      padding: '9px 10px',
                      borderRadius: 8,
                      borderBottom: '1px solid var(--border)',
                      marginBottom: 4,
                      color: 'var(--text)',
                      textDecoration: 'none',
                    }}
                  >
                    <AccountAvatar user={user} size={34} />
                    <span style={{ minWidth: 0 }}>
                      <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 13, fontWeight: 600 }}>
                        {user.full_name || user.username}
                      </span>
                      <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}>
                        @{user.username}
                      </span>
                    </span>
                  </Link>
                  <div aria-hidden="true" style={{ height: 1, margin: '4px 0', background: 'var(--border)' }} />
                  <button
                    type="button"
                    role="menuitem"
                    onClick={onSignOut}
                    className="ac-hover-surface2"
                    style={{
                      display: 'flex',
                      width: '100%',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 10px',
                      border: 'none',
                      background: 'none',
                      borderRadius: 8,
                      cursor: 'pointer',
                      fontSize: 13,
                      color: 'var(--error)',
                      textAlign: 'left',
                    }}
                  >
                    Sign out
                  </button>
                </div>
              )}
            </div>
          ) : (
            <Link
              href="/login"
              className="ac-hover-accent"
              style={{
                ...buttonStyles.primary(isMobile ? 44 : 36),
                display: 'inline-flex',
                alignItems: 'center',
                textDecoration: 'none',
                fontSize: 12.5,
              }}
            >
              Sign in
            </Link>
          )}

          {isMobile && (
            <button
              type="button"
              onClick={() => setMenu(menu === 'mobile' ? null : 'mobile')}
              aria-label="Menu"
              aria-expanded={menu === 'mobile'}
              style={{ ...buttonStyles.iconButton(44) }}
            >
              <Icon.Menu />
            </button>
          )}
        </div>
      </div>

      {menu === 'mobile' && (
        <nav
          aria-label="Mobile"
          style={{
            borderTop: '1px solid var(--border)',
            background: 'var(--surface)',
            padding: '8px 12px 12px',
            display: 'flex',
            flexDirection: 'column',
            gap: 2,
            animation: 'acFadeUp .18s ease',
          }}
        >
          {navItems.map((item) => {
            const current = isCurrent(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  minHeight: 44,
                  padding: '0 12px',
                  background: current ? 'var(--accent-soft)' : 'transparent',
                  borderRadius: 8,
                  fontSize: 14,
                  fontWeight: current ? 600 : 500,
                  color: current ? 'var(--text)' : 'var(--text2)',
                  textDecoration: 'none',
                }}
              >
                <span
                  style={{
                    width: 5,
                    height: 5,
                    borderRadius: '50%',
                    background: current ? 'var(--accent)' : 'transparent',
                  }}
                />
                {item.label}
              </Link>
            );
          })}
          <div ref={searchRef}>
            <UserSearch
              mobile
              query={query}
              open={searchOpen && query.trim().length >= 2}
              loading={searchLoading}
              failed={searchError}
              results={searchResults}
              activeIndex={activeSearchIndex}
              onChange={onSearchChange}
              onFocus={() => query.trim().length >= 2 && setSearchOpen(true)}
              onKeyDown={onSearchKeyDown}
              onSubmit={submitSearch}
              onActivate={setActiveSearchIndex}
              onSelect={navigateToSearchResult}
            />
          </div>
        </nav>
      )}
    </header>
  );
}

function AccountAvatar({ user, size }: { user: Me; size: number }) {
  const source = avatarUrl(user.avatar_url, API_BASE_URL);
  const displayName = user.full_name || user.username;

  return (
    <span
      aria-hidden="true"
      style={{
        position: 'relative',
        width: size,
        height: size,
        flex: `0 0 ${size}px`,
        overflow: 'hidden',
        borderRadius: '50%',
        background: 'var(--accent-soft2)',
        color: 'var(--accent-fg)',
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: Math.max(11, Math.round(size * 0.4)),
        fontWeight: 650,
      }}
    >
      {initials(displayName)}
      {source && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={source}
          alt=""
          onError={(event) => { event.currentTarget.style.display = 'none'; }}
          style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', background: 'var(--surface)' }}
        />
      )}
    </span>
  );
}

function UserSearch({
  mobile = false,
  query,
  open,
  loading,
  failed,
  results,
  activeIndex,
  onChange,
  onFocus,
  onKeyDown,
  onSubmit,
  onActivate,
  onSelect,
}: {
  mobile?: boolean;
  query: string;
  open: boolean;
  loading: boolean;
  failed: boolean;
  results: GlobalSearchResult[];
  activeIndex: number;
  onChange: (value: string) => void;
  onFocus: () => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  onSubmit: (event: React.FormEvent) => void;
  onActivate: (index: number) => void;
  onSelect: (result: GlobalSearchResult) => void;
}) {
  const listboxID = mobile ? 'ac-global-search-results-mobile' : 'ac-global-search-results';
  const activeID = activeIndex >= 0 ? `${listboxID}-${activeIndex}` : undefined;

  return (
    <form onSubmit={onSubmit} style={{ position: 'relative', display: mobile ? 'block' : 'flex', alignItems: 'center' }}>
      {!mobile && (
        <span aria-hidden="true" style={{ position: 'absolute', left: 11, display: 'flex' }}>
          <Icon.Search color="var(--text3)" />
        </span>
      )}
      <input
        id="ac-global-search"
        type="search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={open}
        aria-controls={open ? listboxID : undefined}
        aria-activedescendant={open ? activeID : undefined}
        value={query}
        onChange={(event) => onChange(event.target.value)}
        onFocus={onFocus}
        onKeyDown={onKeyDown}
        placeholder="Search AstraCode…"
        aria-label="Search AstraCode problems and users"
        className={!mobile ? 'ac-input' : undefined}
        style={
          mobile
            ? {
                marginTop: 8,
                width: '100%',
                boxSizing: 'border-box',
                height: 44,
                borderRadius: 8,
                border: '1px solid var(--border)',
                background: 'var(--surface2)',
                padding: '0 14px',
                fontSize: 14,
                color: 'var(--text)',
              }
            : {
                height: 34,
                width: 210,
                borderRadius: 8,
                border: '1px solid var(--border)',
                background: 'var(--surface2)',
                padding: '0 44px 0 33px',
                fontSize: 13,
                color: 'var(--text)',
                transition: 'border-color .15s, background-color .25s',
              }
        }
      />
      {!mobile && (
        <span
          aria-hidden="true"
          style={{
            position: 'absolute',
            right: 9,
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text3)',
            border: '1px solid var(--border)',
            borderRadius: 4,
            padding: '1px 5px',
            background: 'var(--surface)',
          }}
        >
          ⌘K
        </span>
      )}

      {open && (
        <div
          id={listboxID}
          role="listbox"
          aria-label="Problem and user search results"
          style={
            mobile
              ? {
                  marginTop: 6,
                  width: '100%',
                  boxSizing: 'border-box',
                  background: 'var(--surface)',
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  boxShadow: 'var(--shadow)',
                  padding: 4,
                }
              : {
                  position: 'absolute',
                  zIndex: 50,
                  top: 40,
                  left: 0,
                  width: 320,
                  background: 'var(--surface)',
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  boxShadow: 'var(--shadow-lg)',
                  padding: 4,
                  animation: 'acPop .15s ease',
                }
          }
        >
          {loading ? (
            <p aria-live="polite" style={{ margin: 0, padding: '10px 12px', fontSize: 12.5, color: 'var(--text3)' }}>
              Searching AstraCode…
            </p>
          ) : failed ? (
            <p role="alert" style={{ margin: 0, padding: '10px 12px', fontSize: 12.5, color: 'var(--error)' }}>
              Search is unavailable. Try again.
            </p>
          ) : results.length === 0 ? (
            <p style={{ margin: 0, padding: '10px 12px', fontSize: 12.5, color: 'var(--text3)' }}>
              No problems or users found.
            </p>
          ) : (
            results.map((result, index) => {
              const user = result.kind === 'user' ? result.user : null;
              const avatar = user ? avatarUrl(user.avatar_url, API_BASE_URL) : null;
              const selected = index === activeIndex;
              const startsGroup = index === 0 || results[index - 1].kind !== result.kind;
              return (
                <Fragment key={result.key}>
                  {startsGroup && (
                    <div className="ac-search-group-label" role="presentation">
                      {result.kind === 'problem' ? 'Problems' : 'Users'}
                    </div>
                  )}
                  <button
                    id={`${listboxID}-${index}`}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onMouseEnter={() => onActivate(index)}
                    onClick={() => onSelect(result)}
                    className="ac-search-option"
                    style={{ background: selected ? 'var(--accent-soft)' : 'transparent' }}
                  >
                    {result.kind === 'problem' ? (
                      <>
                        <span className="ac-search-problem-icon" aria-hidden="true">{'</>'}</span>
                        <span style={{ minWidth: 0, flex: 1 }}>
                          <span className="ac-search-result-title">{result.problem.title}</span>
                          <span className="ac-search-result-meta">
                            #{result.problem.id} · {difficultyMeta(result.problem.difficulty).label}
                          </span>
                        </span>
                      </>
                    ) : (
                      <>
                        {avatar ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={avatar} alt="" width={30} height={30} className="ac-search-avatar" />
                        ) : (
                          <span aria-hidden="true" className="ac-search-avatar ac-search-avatar-fallback">
                            {initials(user?.full_name || user?.username)}
                          </span>
                        )}
                        <span style={{ minWidth: 0 }}>
                          <span className="ac-search-result-title ac-search-result-handle">@{user?.username}</span>
                          {user?.full_name && <span className="ac-search-result-meta">{user.full_name}</span>}
                        </span>
                      </>
                    )}
                  </button>
                </Fragment>
              );
            })
          )}
        </div>
      )}
    </form>
  );
}
