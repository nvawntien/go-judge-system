'use client';

import { useEffect, type ReactNode } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { ErrorState, SkeletonBar } from '@/components/ui';
import { ADMIN_CONSOLE_MIN_ROLE } from '@/components/admin/AdminNavigation';
import { isContributorWorkspaceUser, roleAtLeast } from '@/components/admin/roles';

export function ContributorShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
      return;
    }
    if (!isContributorWorkspaceUser(user.role) && roleAtLeast(user.role, ADMIN_CONSOLE_MIN_ROLE)) {
      router.replace('/admin/problems');
    }
  }, [loading, pathname, router, user]);

  const redirectingToAdmin = Boolean(
    user && !isContributorWorkspaceUser(user.role) && roleAtLeast(user.role, ADMIN_CONSOLE_MIN_ROLE),
  );

  if (loading || !user || redirectingToAdmin) {
    return (
      <AppShell maxWidth={1180}>
        <div
          aria-live="polite"
          aria-label={redirectingToAdmin ? 'Opening Admin Console' : 'Loading contributions'}
          style={{ display: 'grid', gap: 12 }}
        >
          <SkeletonBar width="32%" height={28} />
          <SkeletonBar width="100%" height={160} />
        </div>
      </AppShell>
    );
  }

  if (!isContributorWorkspaceUser(user.role)) {
    return (
      <AppShell maxWidth={900}>
        <ErrorState
          title="Contributor access required"
          detail="Problem authoring is available in the Contributor Workspace."
        />
      </AppShell>
    );
  }

  return <AppShell maxWidth={1180}>{children}</AppShell>;
}
