'use client';

import { AdminFeatureUnavailable, AdminPageHeader } from '@/components/admin/AdminStates';

export default function AdminSubmissionDetailPage() {
  return (
    <>
      <AdminPageHeader title="Submission detail" description="This route is reserved for the admin submission detail API." />
      <AdminFeatureUnavailable
        description="The gateway exposes only the admin submission list today; service-side admin detail is not implemented."
        backHref="/admin/submissions"
        backLabel="Back to Submissions"
      />
    </>
  );
}
