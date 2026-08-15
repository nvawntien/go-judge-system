'use client';

import { useRouter } from 'next/navigation';
import { PageHeading } from '@/components/AppShell';
import { AdminProblemForm } from '@/components/admin/AdminProblemForm';
import { useToast } from '@/components/ToastProvider';
import { contributionProblemApi } from '@/lib/api';
import type { CreateAdminProblemRequest, UpdateAdminProblemRequest } from '@/lib/types';

export default function NewContributionPage() {
  const router = useRouter();
  const { showToast } = useToast();

  const createProblem = async (body: CreateAdminProblemRequest | UpdateAdminProblemRequest) => {
    const problem = await contributionProblemApi.create(body as CreateAdminProblemRequest);
    showToast('Problem draft created', 'success');
    router.push(`/contributions/${problem.id}`);
  };

  return (
    <>
      <PageHeading title="New problem draft" subtitle="Author the problem content now; a Moderator or Admin controls publication." />
      <AdminProblemForm mode="create" cancelHref="/contributions" onSubmit={createProblem} />
    </>
  );
}
