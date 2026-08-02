'use client';

import { useParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AdminPageHeader, AdminApiError, AdminDialog, AdminLoadingState, adminCard } from '@/components/admin/AdminStates';
import { DateText, BooleanPill } from '@/components/admin/AdminData';
import { AdminProblemForm } from '@/components/admin/AdminProblemForm';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
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
  const [testcaseBusy, setTestcaseBusy] = useState(false);
  const testcaseBusyRef = useRef(false);
  const [confirmDeleteTestCase, setConfirmDeleteTestCase] = useState(false);

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

  const uploadTestCase = async (file: File) => {
    if (testcaseBusyRef.current) return;
    testcaseBusyRef.current = true;
    setTestcaseBusy(true);
    try {
      await adminProblemApi.uploadTestCase(problemId, file);
      showToast('Testcase uploaded', 'success');
      const next = await adminProblemApi.get(problemId);
      setProblem(next);
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setTestcaseBusy(false);
      testcaseBusyRef.current = false;
    }
  };

  const deleteTestCase = async () => {
    if (testcaseBusyRef.current) return;
    testcaseBusyRef.current = true;
    setTestcaseBusy(true);
    try {
      await adminProblemApi.deleteTestCase(problemId);
      showToast('Testcase deleted', 'success');
      setConfirmDeleteTestCase(false);
      const next = await adminProblemApi.get(problemId);
      setProblem(next);
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setTestcaseBusy(false);
      testcaseBusyRef.current = false;
    }
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
          <span style={{ color: 'var(--text2)', fontSize: 12 }}>
            Testcase {problem.testcase?.has_testcase ? `${problem.testcase.test_count ?? '-'} tests` : 'not uploaded'}
          </span>
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          <label className="ac-hover-surface2" style={{ ...buttonStyles.secondary(36), cursor: testcaseBusy ? 'not-allowed' : 'pointer' }}>
            {testcaseBusy ? 'Uploading...' : 'Upload testcase zip'}
            <input
              type="file"
              accept=".zip,application/zip"
              disabled={testcaseBusy}
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) void uploadTestCase(file);
                event.currentTarget.value = '';
              }}
              style={{ position: 'absolute', opacity: 0, pointerEvents: 'none' }}
            />
          </label>
          <button
            type="button"
            disabled={testcaseBusy || !problem.testcase?.has_testcase}
            onClick={() => setConfirmDeleteTestCase(true)}
            className="ac-hover-surface2"
            style={{ ...buttonStyles.secondary(36), color: 'var(--error)' }}
          >
            Delete testcase
          </button>
        </div>
      </section>
      <AdminProblemForm mode="edit" problem={problem} onSubmit={updateProblem} />

      {confirmDeleteTestCase && (
        <AdminDialog title={`Delete testcase for problem #${problem.id}?`} onClose={() => setConfirmDeleteTestCase(false)}>
          <p style={{ margin: '0 0 16px', color: 'var(--text2)', fontSize: 13.5 }}>
            This removes testcase metadata and stored testcase archive through the real admin testcase API. Hidden testcase content is not displayed here.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button
              type="button"
              onClick={() => setConfirmDeleteTestCase(false)}
              className="ac-hover-surface2"
              style={buttonStyles.secondary(38)}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={deleteTestCase}
              disabled={testcaseBusy}
              className="ac-hover-accent"
              style={{ ...buttonStyles.primary(38), background: 'var(--error)' }}
            >
              {testcaseBusy ? 'Deleting...' : 'Delete testcase'}
            </button>
          </div>
        </AdminDialog>
      )}
    </>
  );
}
