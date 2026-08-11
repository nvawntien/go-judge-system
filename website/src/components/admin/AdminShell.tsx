'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useAuth } from '@/components/AuthProvider';
import { useTheme } from '@/components/ThemeProvider';
import { useToast } from '@/components/ToastProvider';
import { Icon, buttonStyles } from '@/components/ui';
import { useViewportWidth } from '@/lib/hooks';
import { AdminIcon } from './AdminIcons';
import { adminNavItemForPath, adminPageTitle, ADMIN_CONSOLE_MIN_ROLE, ADMIN_NAV_ITEMS } from './AdminNavigation';
import { AdminSidebar } from './AdminSidebar';
import { AdminForbiddenState, AdminShellLoading } from './AdminStates';
import { roleAtLeast, roleLabel } from './roles';

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

export function AdminShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const width = useViewportWidth();
  const isMobile = width < 760;
  const isTablet = width >= 760 && width < 1120;
  const { user, loading, logout } = useAuth();
  const { resolved, toggle } = useTheme();
  const { showToast } = useToast();
  const [collapsed, setCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const drawerRef = useRef<HTMLDivElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isTablet) setCollapsed(true);
  }, [isTablet]);

  useEffect(() => {
    if (!isMobile) setDrawerOpen(false);
  }, [isMobile]);

  useEffect(() => {
    if (!drawerOpen) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const firstLink = drawerRef.current?.querySelector<HTMLAnchorElement>('a[href]');
    firstLink?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setDrawerOpen(false);
        menuButtonRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [drawerOpen]);

  useEffect(() => {
    setDrawerOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!loading && !user) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    }
  }, [loading, pathname, router, user]);

  const visibleNav = useMemo(
    () => ADMIN_NAV_ITEMS.filter((item) => roleAtLeast(user?.role, item.minRole)),
    [user?.role],
  );

  const onSignOut = useCallback(async () => {
    await logout();
    showToast('Signed out', 'info');
    router.push('/login?next=/admin');
  }, [logout, router, showToast]);

  if (loading || (!user && typeof window !== 'undefined')) return <AdminShellLoading />;
  if (!user) return <AdminShellLoading />;
  if (!roleAtLeast(user.role, ADMIN_CONSOLE_MIN_ROLE)) return <AdminForbiddenState />;
  const currentNav = adminNavItemForPath(pathname);
  if (currentNav && !roleAtLeast(user.role, currentNav.minRole)) return <AdminForbiddenState />;

  const sidebarWidth = collapsed ? 72 : 256;
  const title = adminPageTitle(pathname);

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--bg)',
        color: 'var(--text)',
        display: 'flex',
      }}
    >
      {!isMobile && (
        <AdminSidebar
          items={visibleNav}
          pathname={pathname}
          collapsed={collapsed}
          width={sidebarWidth}
          user={user}
          onToggle={() => setCollapsed((current) => !current)}
          onSignOut={onSignOut}
        />
      )}

      {drawerOpen && isMobile && (
        <div
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              setDrawerOpen(false);
              menuButtonRef.current?.focus();
            }
          }}
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 80,
            background: 'rgba(17, 13, 35, .42)',
          }}
        >
          <div ref={drawerRef}>
            <AdminSidebar
              items={visibleNav}
              pathname={pathname}
              collapsed={false}
              width={276}
              user={user}
              mobile
              onToggle={() => {
                setDrawerOpen(false);
                menuButtonRef.current?.focus();
              }}
              onSignOut={onSignOut}
            />
          </div>
        </div>
      )}

      <div style={{ flex: 1, minWidth: 0, marginLeft: isMobile ? 0 : sidebarWidth }}>
        <header
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 30,
            height: 56,
            borderBottom: '1px solid var(--border)',
            background: 'var(--surface)',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '0 18px',
          }}
        >
          {isMobile && (
            <button
              ref={menuButtonRef}
              type="button"
              onClick={() => setDrawerOpen(true)}
              aria-label="Open admin navigation"
              className="ac-hover-surface2-text"
              style={buttonStyles.iconButton(40)}
            >
              <AdminIcon.Menu />
            </button>
          )}
          <div style={{ minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11.5, color: 'var(--text3)' }}>
              <Link href="/admin" style={{ color: 'var(--text3)' }}>
                Admin
              </Link>
              <span aria-hidden="true">/</span>
              <span style={{ color: 'var(--text2)' }}>{title}</span>
            </div>
            <div style={{ fontSize: 15, fontWeight: 660, lineHeight: 1.2 }}>{title}</div>
          </div>
          <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
            <button
              type="button"
              onClick={toggle}
              aria-label={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
              title={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
              className="ac-hover-surface2-text"
              style={buttonStyles.iconButton(38)}
            >
              {resolved === 'dark' ? <Icon.Sun /> : <Icon.Moon />}
            </button>
            <span
              title={`${user.full_name || user.username} · ${roleLabel(user.role)}`}
              style={{
                width: 32,
                height: 32,
                borderRadius: '50%',
                background: 'var(--accent-soft2)',
                color: 'var(--accent-fg)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 11,
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              {initials(user.full_name || user.username)}
            </span>
          </div>
        </header>
        <main
          style={{
            padding: isMobile ? '18px 14px 40px' : '24px 26px 56px',
            maxWidth: 1280,
            margin: '0 auto',
            animation: 'acFadeUp .22s ease',
          }}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
