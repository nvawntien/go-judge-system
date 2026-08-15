'use client';

import { useId, useRef, useState } from 'react';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError } from '@/lib/api';
import type { AdminTestCaseMetadata } from '@/lib/types';
import { AdminDialog, adminCard } from './AdminStates';

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Testcase request failed.';
}

export function TestcaseManager({
  problemId,
  testcase,
  onUpload,
  onDelete,
}: {
  problemId: number;
  testcase: AdminTestCaseMetadata;
  onUpload: (file: File) => Promise<void>;
  onDelete: () => Promise<void>;
}) {
  const { showToast } = useToast();
  const inputId = useId();
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [error, setError] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);

  const runMutation = async (action: () => Promise<void>, successMessage: string) => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError('');
    try {
      await action();
      showToast(successMessage, 'success');
    } catch (err) {
      const message = errorMessage(err);
      setError(message);
      showToast(message, 'error');
    } finally {
      setBusy(false);
      busyRef.current = false;
    }
  };

  const upload = async () => {
    if (!file) return;
    await runMutation(async () => {
      await onUpload(file);
      setFile(null);
    }, testcase.has_testcase ? 'Testcase replaced' : 'Testcase uploaded');
  };

  const remove = async () => {
    await runMutation(async () => {
      await onDelete();
      setConfirmDelete(false);
    }, 'Testcase deleted');
  };

  const versionLabel = testcase.version ? `Version ${testcase.version}` : 'Dataset uploaded';
  const datasetLabel = testcase.has_testcase
    ? `${testcase.test_count ?? 0} test${testcase.test_count === 1 ? '' : 's'} · ${versionLabel}`
    : 'No testcase package uploaded yet.';

  return (
    <section aria-labelledby="testcase-manager-heading" style={{ ...adminCard, padding: 16, display: 'grid', gap: 12 }}>
      <div style={{ display: 'grid', gap: 3 }}>
        <h2 id="testcase-manager-heading" style={{ margin: 0, fontSize: 15 }}>Testcases</h2>
        <p style={{ margin: 0, color: 'var(--text2)', fontSize: 12.5, lineHeight: 1.5 }}>{datasetLabel}</p>
        <p style={{ margin: 0, color: 'var(--text3)', fontSize: 12, lineHeight: 1.5 }}>
          Upload a ZIP containing consecutive <code>n.in</code> and <code>n.out</code> pairs. A new archive replaces the current dataset.
        </p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', gap: 8, alignItems: 'end' }}>
        <label htmlFor={inputId} style={{ display: 'grid', gap: 6, minWidth: 0, fontSize: 12, fontWeight: 650 }}>
          Testcase ZIP
          <input
            id={inputId}
            type="file"
            accept=".zip,application/zip"
            disabled={busy}
            onChange={(event) => {
              setFile(event.target.files?.[0] ?? null);
              setError('');
            }}
            className="ac-input"
            style={{ ...buttonStyles.secondary(36), width: '100%', maxWidth: '100%', padding: '6px 8px', fontWeight: 400 }}
          />
        </label>
        <button
          type="button"
          disabled={busy || !file}
          aria-busy={busy}
          onClick={() => void upload()}
          className="ac-hover-accent"
          style={buttonStyles.primary(36)}
        >
          {busy ? 'Uploading...' : testcase.has_testcase ? 'Replace ZIP' : 'Upload ZIP'}
        </button>
      </div>

      {file && <p style={{ margin: '-4px 0 0', color: 'var(--text3)', fontSize: 12, overflowWrap: 'anywhere' }}>Selected: {file.name}</p>}
      {error && <p role="alert" style={{ margin: 0, color: 'var(--error)', fontSize: 12.5 }}>{error}</p>}

      {testcase.has_testcase && (
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <button
            type="button"
            disabled={busy}
            onClick={() => setConfirmDelete(true)}
            className="ac-hover-surface2"
            style={{ ...buttonStyles.secondary(36), color: 'var(--error)' }}
          >
            Delete testcase
          </button>
        </div>
      )}

      {confirmDelete && (
        <AdminDialog title={`Delete testcase for problem #${problemId}?`} onClose={() => !busy && setConfirmDelete(false)}>
          <p style={{ margin: '0 0 16px', color: 'var(--text2)', fontSize: 13.5, lineHeight: 1.55 }}>
            This permanently removes the stored testcase package and its metadata. Testcase contents are not shown here.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', flexWrap: 'wrap', gap: 8 }}>
            <button type="button" onClick={() => setConfirmDelete(false)} disabled={busy} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void remove()}
              disabled={busy}
              aria-busy={busy}
              className="ac-hover-accent"
              style={{ ...buttonStyles.primary(38), background: 'var(--error)' }}
            >
              {busy ? 'Deleting...' : 'Delete testcase'}
            </button>
          </div>
        </AdminDialog>
      )}
    </section>
  );
}
