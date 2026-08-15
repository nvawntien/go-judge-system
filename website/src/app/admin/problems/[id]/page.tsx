'use client';

import { useParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdminPageHeader, AdminApiError, AdminLoadingState, adminCard } from '@/components/admin/AdminStates';
import { DateText, BooleanPill } from '@/components/admin/AdminData';
import { AdminProblemForm } from '@/components/admin/AdminProblemForm';
import { TestcaseManager } from '@/components/admin/TestcaseManager';
import { useToast } from '@/components/ToastProvider';
import { ApiError, NetworkError, adminProblemApi } from '@/lib/api';
import type { AdminProblemDetail, CreateAdminProblemRequest, UpdateAdminProblemRequest } from '@/lib/types';

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

export default function AdminProblemDetailPage() {
  const params = useParams<{ id: string }>();
  const problemId = useMemo(() => Number(params.id), [params.id]);
  const { showToast } = useToast();
  const [problem, setProblem] = useState<AdminProblemDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(
    (signal?: AbortSignal) => {
      setLoading(true);
      setError('');
      adminProblemApi
        .get(problemId, signal)
        .then(setProblem)
        .catch((err) => {
          if (!signal?.aborted) setError(errorMessage(err));
        })
        .finally(() => {
          if (!signal?.aborted) setLoading(false);
        });
    },
    [problemId],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const updateProblem = async (body: CreateAdminProblemRequest | UpdateAdminProblemRequest) => {
    await adminProblemApi.update(problemId, body as UpdateAdminProblemRequest);
    showToast('Problem updated', 'success');
    const next = await adminProblemApi.get(problemId);
    setProblem(next);
  };

  if (loading) return <AdminLoadingState title="Loading problem detail" />;
  if (error) return <AdminApiError title="Could not load problem" error={error} onRetry={() => load()} />;
  if (!problem) return <AdminApiError title="Problem not found" error="The admin detail API did not return a problem." onRetry={() => load()} />;

  return (
    <>
      <AdminPageHeader
        title={`Problem #${problem.id}`}
        description={
          <>
            {problem.slug} · <DateText value={problem.updated_at || problem.created_at} />
          </>
        }
      />
      <section style={{ ...adminCard, padding: 16, marginBottom: 14, display: 'grid', gap: 10 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}>
          <BooleanPill value={!problem.is_hidden} trueLabel="Published" falseLabel="Hidden" />
          <span style={{ color: 'var(--text2)', fontSize: 12 }}>Author {problem.author_id || '-'}</span>
        </div>
      </section>
      <TestcaseManager
        problemId={problemId}
        testcase={problem.testcase}
        onUpload={async (file) => {
          await adminProblemApi.uploadTestCase(problemId, file);
          setProblem(await adminProblemApi.get(problemId));
        }}
        onDelete={async () => {
          await adminProblemApi.deleteTestCase(problemId);
          setProblem(await adminProblemApi.get(problemId));
        }}
      />
      <AdminProblemForm mode="edit" problem={problem} onSubmit={updateProblem} />
    </>
  );
}
