'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import { useAuth } from './AuthProvider';
import { useTheme } from './ThemeProvider';
import { useToast } from './ToastProvider';
import { useDismissable, useViewportWidth } from '@/lib/hooks';
import { initials, ratingTier } from '@/lib/format';
import { Icon, Logo, Wordmark, buttonStyles } from './ui';

const NAV_ITEMS = [
  { label: 'Problems', href: '/problems' },
  { label: 'Contests', href: '/contests' },
  { label: 'Leaderboard', href: '/leaderboard' },
  { label: 'Discussions', href: '/discuss' },
];

export function Header() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const { resolved, toggle } = useTheme();
  const { showToast } = useToast();
  const width = useViewportWidth();
  const isMobile = width < 760;

  const [menu, setMenu] = useState<'notif' | 'user' | 'mobile' | null>(null);
  const [query, setQuery] = useState('');

  const close = useCallback(() => setMenu(null), []);
  const notifRef = useDismissable<HTMLDivElement>(menu === 'notif', close);
  const userRef = useDismissable<HTMLDivElement>(menu === 'user', close);

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
  }, [pathname]);

  const submitSearch = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = query.trim();
    router.push(trimmed ? `/problems?search=${encodeURIComponent(trimmed)}` : '/problems');
  };

  const onSignOut = async () => {
    await logout();
    showToast('Signed out', 'info');
    router.push('/login');
  };

  const isCurrent = (href: string) => pathname === href || pathname.startsWith(`${href}/`);

  return (
    <header
      style={{
        position: 'sticky',
        top: 0,
        zIndex: 40,
        background: 'var(--surface)',
        borderBottom: '1px solid var(--border)',
        transition: 'background-color .25s ease, border-color .25s ease',
      }}
    >
      <div
        style={{
          maxWidth: 1440,
          margin: '0 auto',
          padding: '0 20px',
          height: 56,
          display: 'flex',
          alignItems: 'center',
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
            {NAV_ITEMS.map((item) => {
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
            <form
              onSubmit={submitSearch}
              style={{ position: 'relative', display: 'flex', alignItems: 'center' }}
            >
              <span style={{ position: 'absolute', left: 11, display: 'flex' }}>
                <Icon.Search color="var(--text3)" />
              </span>
              <input
                id="ac-global-search"
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search AstraCode…"
                aria-label="Search AstraCode"
                className="ac-input"
                style={{
                  height: 34,
                  width: 210,
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: 'var(--surface2)',
                  padding: '0 44px 0 33px',
                  fontSize: 13,
                  color: 'var(--text)',
                  transition: 'border-color .15s, background-color .25s',
                }}
              />
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
            </form>
          )}

          <button
            type="button"
            onClick={toggle}
            aria-label={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            title={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            className="ac-hover-surface2-text"
            style={buttonStyles.iconButton()}
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
              style={buttonStyles.iconButton()}
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
                  No notifications yet — the backend does not expose a notification feed.
                </p>
              </div>
            )}
          </div>

          {user ? (
            <div style={{ position: 'relative' }} ref={userRef}>
              <button
                type="button"
                onClick={() => setMenu(menu === 'user' ? null : 'user')}
                aria-label="Account menu"
                aria-expanded={menu === 'user'}
                className="ac-hover-surface2"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  height: 36,
                  padding: '0 8px 0 4px',
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: 'var(--surface)',
                  cursor: 'pointer',
                  color: 'var(--text)',
                }}
              >
                <span
                  aria-hidden="true"
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: '50%',
                    background: 'var(--accent-soft2)',
                    color: 'var(--accent-fg)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 11.5,
                    fontWeight: 650,
                  }}
                >
                  {initials(user.full_name || user.username)}
                </span>
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
                    width: 208,
                    background: 'var(--surface)',
                    border: '1px solid var(--border)',
                    borderRadius: 12,
                    boxShadow: 'var(--shadow-lg)',
                    padding: 6,
                    animation: 'acPop .15s ease',
                  }}
                >
                  <div
                    style={{
                      padding: '8px 10px',
                      borderBottom: '1px solid var(--border)',
                      marginBottom: 4,
                    }}
                  >
                    <span style={{ display: 'block', fontSize: 13, fontWeight: 600 }}>
                      {user.full_name || user.username}
                    </span>
                    <span
                      style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text3)' }}
                    >
                      @{user.username} · {user.rating}
                    </span>
                  </div>
                  {[
                    { label: 'Public profile', href: '/profile' },
                    { label: 'Profile settings', href: '/settings' },
                    { label: 'Design system notes', href: '/design-system' },
                  ].map((item) => (
                    <Link
                      key={item.href}
                      role="menuitem"
                      href={item.href}
                      className="ac-hover-surface2"
                      style={{
                        display: 'flex',
                        width: '100%',
                        alignItems: 'center',
                        gap: 8,
                        padding: '8px 10px',
                        borderRadius: 8,
                        fontSize: 13,
                        color: 'var(--text)',
                        textDecoration: 'none',
                        boxSizing: 'border-box',
                      }}
                    >
                      {item.label}
                    </Link>
                  ))}
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
                ...buttonStyles.primary(36),
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
          {NAV_ITEMS.map((item) => {
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
          <form onSubmit={submitSearch}>
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search AstraCode…"
              aria-label="Search AstraCode"
              style={{
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
              }}
            />
          </form>
        </nav>
      )}
    </header>
  );
}

/** Small badge shown next to a username. */
export function TierBadge({ rating }: { rating: number }) {
  return (
    <span
      style={{
        fontSize: 11,
        fontWeight: 650,
        color: 'var(--accent-fg)',
        background: 'var(--accent-soft)',
        borderRadius: 6,
        padding: '2px 9px',
        whiteSpace: 'nowrap',
      }}
    >
      {ratingTier(rating)}
    </span>
  );
}
