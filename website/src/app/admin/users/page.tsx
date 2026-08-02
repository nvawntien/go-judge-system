'use client';

import { AdminFeatureUnavailable, AdminPageHeader } from '@/components/admin/AdminStates';

export default function AdminUsersPage() {
  return (
    <>
      <AdminPageHeader title="Users" description="User administration requires an admin user list API before this page can load real data." />
      <AdminFeatureUnavailable description="The auth service currently exposes role assignment by user ID, but no admin user list or detail endpoint." />
    </>
  );
}
