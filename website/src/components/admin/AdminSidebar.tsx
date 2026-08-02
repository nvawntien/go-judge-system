'use client';

import Link from 'next/link';
import { Logo, Wordmark, buttonStyles } from '@/components/ui';
import type { Me } from '@/lib/types';
import { AdminIcon } from './AdminIcons';
import type { AdminNavItem } from './AdminNavigation';
import { isAdminRouteActive } from './AdminNavigation';
import { roleAtLeast, roleLabel } from './roles';

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

export function AdminSidebar({
  items,
  pathname,
  collapsed,
  width,
  user,
  mobile = false,
  onToggle,
  onSignOut,
}: {
  items: AdminNavItem[];
  pathname: string;
  collapsed: boolean;
  width: number;
  user: Me;
  mobile?: boolean;
  onToggle: () => void;
  onSignOut: () => void;
}) {
  return (
    <aside
      aria-label="Admin navigation"
      style={{
        position: 'fixed',
        inset: '0 auto 0 0',
        zIndex: mobile ? 90 : 50,
        width,
        height: '100vh',
        boxSizing: 'border-box',
        background: 'var(--surface)',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        transition: 'width .18s ease',
      }}
    >
      <div
        style={{
          height: 58,
          display: 'flex',
          alignItems: 'center',
          gap: 9,
          padding: collapsed ? '0 14px' : '0 18px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <Link
          href="/admin"
          aria-label="AstraCode admin overview"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 9,
            color: 'var(--text)',
            textDecoration: 'none',
            minWidth: 0,
          }}
        >
          <Logo />
          {!collapsed && <Wordmark />}
        </Link>
        <button
          type="button"
          onClick={onToggle}
          aria-label={mobile ? 'Close admin navigation' : collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          title={mobile ? 'Close' : collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="ac-hover-surface2-text"
          style={{ ...buttonStyles.iconButton(34), marginLeft: 'auto' }}
        >
          {mobile ? <AdminIcon.X /> : collapsed ? <AdminIcon.PanelLeft /> : <AdminIcon.ChevronLeft />}
        </button>
      </div>

      <nav style={{ flex: 1, overflowY: 'auto', padding: '14px 10px' }}>
        {!collapsed && (
          <div
            style={{
              padding: '0 9px 8px',
              fontSize: 10.5,
              letterSpacing: 0.4,
              textTransform: 'uppercase',
              color: 'var(--text3)',
              fontWeight: 700,
            }}
          >
            Workspace
          </div>
        )}
        {items.map((item) => {
          const active = isAdminRouteActive(pathname, item.href);
          const locked = !roleAtLeast(user.role, item.minRole);
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? 'page' : undefined}
              title={collapsed ? `${item.label}${locked ? ` · requires ${roleLabel(item.minRole)}` : ''}` : undefined}
              className="ac-hover-surface2-text"
              style={{
                minHeight: 44,
                display: 'flex',
                alignItems: 'center',
                justifyContent: collapsed ? 'center' : 'flex-start',
                gap: 10,
                padding: collapsed ? '0' : '0 10px',
                borderRadius: 8,
                marginBottom: 4,
                color: active ? 'var(--accent-fg)' : locked ? 'var(--text3)' : 'var(--text2)',
                background: active ? 'var(--accent-soft)' : 'transparent',
                border: active ? '1px solid var(--accent-soft2)' : '1px solid transparent',
                boxSizing: 'border-box',
                textDecoration: 'none',
                fontSize: 13,
                fontWeight: active ? 650 : 560,
              }}
            >
              <span style={{ display: 'flex', flexShrink: 0 }}>{item.icon}</span>
              {!collapsed && (
                <>
                  <span style={{ flex: 1 }}>{item.label}</span>
                  {item.readiness !== 'available' && (
                    <span
                      style={{
                        width: 7,
                        height: 7,
                        borderRadius: '50%',
                        background: item.readiness === 'partial' ? 'var(--warn)' : 'var(--border2)',
                      }}
                      aria-label={item.readiness === 'partial' ? 'Partially available' : 'Unavailable'}
                      title={item.readiness === 'partial' ? 'Partially available' : 'Unavailable'}
                    />
                  )}
                </>
              )}
            </Link>
          );
        })}
      </nav>

      <div style={{ padding: 10, borderTop: '1px solid var(--border)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 9,
            padding: collapsed ? '8px 0' : '8px',
            justifyContent: collapsed ? 'center' : 'flex-start',
          }}
        >
          <span
            aria-hidden="true"
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
          {!collapsed && (
            <span style={{ minWidth: 0, flex: 1 }}>
              <span style={{ display: 'block', fontSize: 12.5, fontWeight: 650, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {user.full_name || user.username}
              </span>
              <span style={{ display: 'block', fontSize: 11, color: 'var(--text3)' }}>{roleLabel(user.role)}</span>
            </span>
          )}
        </div>
        {!collapsed && (
          <button
            type="button"
            onClick={onSignOut}
            className="ac-hover-surface2"
            style={{
              width: '100%',
              minHeight: 40,
              border: 'none',
              borderRadius: 8,
              background: 'transparent',
              color: 'var(--error)',
              cursor: 'pointer',
              fontSize: 13,
              fontWeight: 600,
              textAlign: 'left',
              padding: '0 10px',
            }}
          >
            Sign out
          </button>
        )}
      </div>
    </aside>
  );
}
