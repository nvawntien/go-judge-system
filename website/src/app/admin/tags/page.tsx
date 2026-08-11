'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { AdminPageHeader, AdminApiError, AdminDialog, AdminLoadingState, adminCard, adminField } from '@/components/admin/AdminStates';
import { AdminTableShell, BooleanPill, DateText, adminTd, adminTh } from '@/components/admin/AdminData';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminTagApi } from '@/lib/api';
import type { AdminTag, CreateAdminTagRequest, UpdateAdminTagRequest } from '@/lib/types';

type TagForm = {
  name: string;
  slug: string;
  description: string;
  is_active: boolean;
};

const emptyTagForm: TagForm = { name: '', slug: '', description: '', is_active: true };

function slugPreview(name: string) {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

function toBody(form: TagForm): CreateAdminTagRequest {
  return {
    name: form.name.trim(),
    slug: form.slug.trim() || undefined,
    description: form.description.trim() || undefined,
    is_active: form.is_active,
  };
}

export default function AdminTagsPage() {
  const { showToast } = useToast();
  const [items, setItems] = useState<AdminTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [createForm, setCreateForm] = useState<TagForm>(emptyTagForm);
  const [editTarget, setEditTarget] = useState<AdminTag | null>(null);
  const [editForm, setEditForm] = useState<TagForm>(emptyTagForm);
  const [deleteTarget, setDeleteTarget] = useState<AdminTag | null>(null);
  const [busyId, setBusyId] = useState<number | 'create' | null>(null);
  const busyRef = useRef(false);

  const activeCount = useMemo(() => items.filter((tag) => tag.is_active).length, [items]);

  const load = useCallback((signal?: AbortSignal) => {
    setLoading(true);
    setError('');
    adminTagApi
      .list(signal)
      .then((res) => setItems(res.items ?? []))
      .catch((err) => {
        if (!signal?.aborted) setError(errorMessage(err));
      })
      .finally(() => {
        if (!signal?.aborted) setLoading(false);
      });
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const createTag = async (event: FormEvent) => {
    event.preventDefault();
    if (busyRef.current) return;
    if (!createForm.name.trim()) {
      showToast('Tag name is required', 'error');
      return;
    }
    busyRef.current = true;
    setBusyId('create');
    try {
      await adminTagApi.create(toBody(createForm));
      showToast('Tag created', 'success');
      setCreateForm(emptyTagForm);
      load();
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setBusyId(null);
      busyRef.current = false;
    }
  };

  const openEdit = (tag: AdminTag) => {
    setEditTarget(tag);
    setEditForm({
      name: tag.name,
      slug: tag.slug,
      description: tag.description ?? '',
      is_active: tag.is_active,
    });
  };

  const updateTag = async (event: FormEvent) => {
    event.preventDefault();
    if (!editTarget) return;
    if (busyRef.current) return;
    busyRef.current = true;
    setBusyId(editTarget.id);
    try {
      await adminTagApi.update(editTarget.id, toBody(editForm) as UpdateAdminTagRequest);
      showToast('Tag updated', 'success');
      setEditTarget(null);
      load();
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setBusyId(null);
      busyRef.current = false;
    }
  };

  const deleteTag = async () => {
    if (!deleteTarget) return;
    if (busyRef.current) return;
    busyRef.current = true;
    setBusyId(deleteTarget.id);
    try {
      await adminTagApi.delete(deleteTarget.id);
      showToast('Tag deactivated', 'success');
      setDeleteTarget(null);
      load();
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setBusyId(null);
      busyRef.current = false;
    }
  };

  return (
    <>
      <AdminPageHeader
        title="Tags"
        description={`Admin tag catalog from the problem-service API. ${activeCount} active tag${activeCount === 1 ? '' : 's'}.`}
      />

      <form
        onSubmit={createTag}
        style={{
          ...adminCard,
          padding: 14,
          marginBottom: 12,
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))',
          gap: 10,
          alignItems: 'end',
        }}
      >
        <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
          Name
          <input
            value={createForm.name}
            onChange={(event) => setCreateForm((current) => ({ ...current, name: event.target.value }))}
            className="ac-input"
            style={adminField}
            required
          />
        </label>
        <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
          Slug
          <input
            value={createForm.slug}
            onChange={(event) => setCreateForm((current) => ({ ...current, slug: event.target.value }))}
            placeholder={slugPreview(createForm.name) || 'auto-generated'}
            className="ac-input"
            style={adminField}
          />
        </label>
        <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
          Description
          <input
            value={createForm.description}
            onChange={(event) => setCreateForm((current) => ({ ...current, description: event.target.value }))}
            className="ac-input"
            style={adminField}
          />
        </label>
        <label style={{ minHeight: 38, display: 'inline-flex', alignItems: 'center', gap: 8, fontSize: 12.5, fontWeight: 650 }}>
          <input
            type="checkbox"
            checked={createForm.is_active}
            onChange={(event) => setCreateForm((current) => ({ ...current, is_active: event.target.checked }))}
          />
          Active
        </label>
        <button type="submit" disabled={busyId === 'create'} className="ac-hover-accent" style={buttonStyles.primary(38)}>
          {busyId === 'create' ? 'Creating...' : 'Create tag'}
        </button>
      </form>

      {loading ? (
        <AdminLoadingState title="Loading tags" />
      ) : error ? (
        <AdminApiError title="Could not load tags" error={error} onRetry={() => load()} />
      ) : items.length === 0 ? (
        <div style={{ ...adminCard, padding: 28, textAlign: 'center' }}>
          <p style={{ margin: '0 0 5px', fontSize: 14, fontWeight: 680 }}>No tags yet</p>
          <p style={{ margin: 0, color: 'var(--text2)', fontSize: 13 }}>Create the first admin tag above.</p>
        </div>
      ) : (
        <AdminTableShell>
          <table style={{ width: '100%', minWidth: 820, borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={adminTh}>ID</th>
                <th style={adminTh}>Name</th>
                <th style={adminTh}>Slug</th>
                <th style={adminTh}>State</th>
                <th style={adminTh}>Updated</th>
                <th style={{ ...adminTh, textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((tag) => (
                <tr key={tag.id}>
                  <td style={{ ...adminTd, fontFamily: 'var(--font-mono)', color: 'var(--text3)' }}>#{tag.id}</td>
                  <td style={adminTd}>
                    <strong>{tag.name}</strong>
                    {tag.description && <div style={{ color: 'var(--text3)', fontSize: 12 }}>{tag.description}</div>}
                  </td>
                  <td style={{ ...adminTd, fontFamily: 'var(--font-mono)', color: 'var(--text2)' }}>{tag.slug}</td>
                  <td style={adminTd}>
                    <BooleanPill value={tag.is_active} trueLabel="Active" falseLabel="Inactive" />
                  </td>
                  <td style={adminTd}>
                    <DateText value={tag.updated_at} />
                  </td>
                  <td style={{ ...adminTd, textAlign: 'right' }}>
                    <span style={{ display: 'inline-flex', gap: 6 }}>
                      <button type="button" onClick={() => openEdit(tag)} className="ac-hover-surface2" style={buttonStyles.secondary(32)}>
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(tag)}
                        disabled={busyId === tag.id}
                        className="ac-hover-surface2"
                        style={{ ...buttonStyles.secondary(32), color: 'var(--error)' }}
                      >
                        Deactivate
                      </button>
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </AdminTableShell>
      )}

      {editTarget && (
        <AdminDialog title={`Edit tag #${editTarget.id}`} onClose={() => setEditTarget(null)}>
          <form onSubmit={updateTag} style={{ display: 'grid', gap: 10 }}>
            <TagFields form={editForm} setForm={setEditForm} />
            <DialogActions
              cancel={() => setEditTarget(null)}
              busy={busyId === editTarget.id}
              label={busyId === editTarget.id ? 'Saving...' : 'Save tag'}
            />
          </form>
        </AdminDialog>
      )}

      {deleteTarget && (
        <AdminDialog title={`Deactivate tag #${deleteTarget.id}?`} onClose={() => setDeleteTarget(null)}>
          <p style={{ margin: '0 0 16px', color: 'var(--text2)', fontSize: 13.5 }}>
            This calls the real admin delete API. The current backend treats unavailable or protected deletes as API errors.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" onClick={() => setDeleteTarget(null)} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
              Cancel
            </button>
            <button
              type="button"
              onClick={deleteTag}
              disabled={busyId === deleteTarget.id}
              className="ac-hover-accent"
              style={{ ...buttonStyles.primary(38), background: 'var(--error)' }}
            >
              {busyId === deleteTarget.id ? 'Deactivating...' : 'Deactivate'}
            </button>
          </div>
        </AdminDialog>
      )}
    </>
  );
}

function TagFields({ form, setForm }: { form: TagForm; setForm: (form: TagForm) => void }) {
  return (
    <>
      <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
        Name
        <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} className="ac-input" style={adminField} required />
      </label>
      <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
        Slug
        <input value={form.slug} onChange={(event) => setForm({ ...form, slug: event.target.value })} className="ac-input" style={adminField} />
      </label>
      <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
        Description
        <textarea
          value={form.description}
          onChange={(event) => setForm({ ...form, description: event.target.value })}
          className="ac-input"
          rows={4}
          style={{ ...adminField, height: 'auto', padding: 10, resize: 'vertical' }}
        />
      </label>
      <label style={{ minHeight: 38, display: 'inline-flex', alignItems: 'center', gap: 8, fontSize: 12.5, fontWeight: 650 }}>
        <input type="checkbox" checked={form.is_active} onChange={(event) => setForm({ ...form, is_active: event.target.checked })} />
        Active
      </label>
    </>
  );
}

function DialogActions({
  cancel,
  busy,
  label,
}: {
  cancel: () => void;
  busy: boolean;
  label: string;
}) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
      <button type="button" onClick={cancel} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
        Cancel
      </button>
      <button type="submit" disabled={busy} className="ac-hover-accent" style={buttonStyles.primary(38)}>
        {label}
      </button>
    </div>
  );
}
