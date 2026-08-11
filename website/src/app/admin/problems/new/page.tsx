'use client';

import { useRouter } from 'next/navigation';
import { AdminPageHeader } from '@/components/admin/AdminStates';
import { AdminProblemForm } from '@/components/admin/AdminProblemForm';
import { useToast } from '@/components/ToastProvider';
import { adminProblemApi } from '@/lib/api';
import type { CreateAdminProblemRequest, UpdateAdminProblemRequest } from '@/lib/types';

export default function AdminNewProblemPage() {
  const router = useRouter();
  const { showToast } = useToast();

  const createProblem = async (body: CreateAdminProblemRequest | UpdateAdminProblemRequest) => {
    const created = await adminProblemApi.create(body as CreateAdminProblemRequest);
    showToast('Problem created', 'success');
    router.push(`/admin/problems/${created.id}`);
  };

  return (
    <>
      <AdminPageHeader title="New problem" description="Create a problem through the real problem-service admin API." />
      <AdminProblemForm mode="create" onSubmit={createProblem} />
    </>
  );
}
