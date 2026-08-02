'use client';

import { AdminFeatureUnavailable, AdminPageHeader } from '@/components/admin/AdminStates';

export default function AdminUserDetailPage() {
  return (
    <>
      <AdminPageHeader title="User detail" description="This route is reserved for the admin user detail API." />
      <AdminFeatureUnavailable
        description="The auth service has no admin user detail endpoint yet, so this page does not call a user profile endpoint to bypass authorization."
        backHref="/admin/users"
        backLabel="Back to Users"
      />
    </>
  );
}
