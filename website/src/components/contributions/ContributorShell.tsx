'use client';

import { useEffect, type ReactNode } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { ErrorState, SkeletonBar } from '@/components/ui';
import { roleAtLeast } from '@/components/admin/roles';

export function ContributorShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && !user) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    }
  }, [loading, pathname, router, user]);

  if (loading || !user) {
    return (
      <AppShell maxWidth={1180}>
        <div aria-label="Loading contributions" style={{ display: 'grid', gap: 12 }}>
          <SkeletonBar width="32%" height={28} />
          <SkeletonBar width="100%" height={160} />
        </div>
      </AppShell>
    );
  }

  if (!roleAtLeast(user.role, 'contributor')) {
    return (
      <AppShell maxWidth={900}>
        <ErrorState
          title="Contributor access required"
          detail="Problem authoring is available to Contributor, Moderator, and Admin roles."
        />
      </AppShell>
    );
  }

  return <AppShell maxWidth={1180}>{children}</AppShell>;
}
