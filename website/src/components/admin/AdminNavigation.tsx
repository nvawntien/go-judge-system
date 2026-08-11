'use client';

import type { ReactNode } from 'react';
import type { Role } from '@/lib/types';
import { AdminIcon } from './AdminIcons';

export type AdminNavItem = {
  label: string;
  href: string;
  icon: ReactNode;
  minRole: Role;
  readiness: 'available' | 'partial' | 'unavailable';
};

export const ADMIN_CONSOLE_MIN_ROLE: Role = 'moderator';

export const ADMIN_NAV_ITEMS: AdminNavItem[] = [
  { label: 'Overview', href: '/admin', icon: <AdminIcon.Dashboard />, minRole: 'moderator', readiness: 'partial' },
  { label: 'Problems', href: '/admin/problems', icon: <AdminIcon.Problems />, minRole: 'moderator', readiness: 'available' },
  { label: 'Tags', href: '/admin/tags', icon: <AdminIcon.Tags />, minRole: 'moderator', readiness: 'available' },
  {
    label: 'Submissions',
    href: '/admin/submissions',
    icon: <AdminIcon.Submissions />,
    minRole: 'moderator',
    readiness: 'partial',
  },
  { label: 'Users', href: '/admin/users', icon: <AdminIcon.Users />, minRole: 'admin', readiness: 'available' },
];

export function isAdminRouteActive(pathname: string, href: string) {
  if (href === '/admin') return pathname === '/admin';
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function adminNavItemForPath(pathname: string) {
  return ADMIN_NAV_ITEMS
    .filter((item) => isAdminRouteActive(pathname, item.href))
    .sort((left, right) => right.href.length - left.href.length)[0];
}

export function adminPageTitle(pathname: string) {
  if (pathname === '/admin') return 'Overview';
  if (pathname.startsWith('/admin/problems/new')) return 'New problem';
  if (pathname.startsWith('/admin/problems/')) return 'Problem detail';
  if (pathname.startsWith('/admin/problems')) return 'Problems';
  if (pathname.startsWith('/admin/tags')) return 'Tags';
  if (pathname.startsWith('/admin/submissions/')) return 'Submission detail';
  if (pathname.startsWith('/admin/submissions')) return 'Submissions';
  if (pathname.startsWith('/admin/users/')) return 'User detail';
  if (pathname.startsWith('/admin/users')) return 'Users';
  return 'Admin';
}
