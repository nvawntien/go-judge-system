'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { AdminUserStatusPill, DateText, RolePill } from '@/components/admin/AdminData';
import { AdminApiError, AdminDialog, AdminForbiddenState, AdminLoadingState, AdminPageHeader, adminCard, adminField } from '@/components/admin/AdminStates';
import { useAuth } from '@/components/AuthProvider';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminUserApi } from '@/lib/api';
import type { AdminUser, AdminUserRole } from '@/lib/types';

const ROLES: AdminUserRole[] = ['user', 'contributor', 'moderator', 'admin'];

type LoadError = 'forbidden' | 'not-found' | 'generic';

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return { kind: 'generic' as const, message: 'Cannot reach the API gateway.' };
  if (err instanceof ApiError) {
    if (err.httpStatus === 403) return { kind: 'forbidden' as const, message: err.message || 'Forbidden.' };
    if (err.httpStatus === 404) return { kind: 'not-found' as const, message: err.message || 'User not found.' };
    return { kind: 'generic' as const, message: err.message || `Request failed with ${err.httpStatus}.` };
  }
  return { kind: 'generic' as const, message: 'Request failed.' };
}

function initials(user: AdminUser) {
  return (user.full_name || user.username)
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

function Avatar({ user }: { user: AdminUser }) {
  return (
    <span
      aria-hidden="true"
      style={{ position: 'relative', width: 58, height: 58, borderRadius: '50%', overflow: 'hidden', background: 'var(--accent-soft2)', color: 'var(--accent-fg)', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: 17, fontWeight: 750, flexShrink: 0 }}
    >
      {initials(user)}
      {user.avatar_url && <img src={user.avatar_url} alt="" onError={(event) => { event.currentTarget.style.display = 'none'; }} style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', background: 'var(--surface2)' }} />}
    </span>
  );
}

function DetailSection({ title, children, description }: { title: string; children: ReactNode; description?: string }) {
  return (
    <section style={{ ...adminCard, padding: 16 }}>
      <h2 style={{ margin: 0, fontSize: 15, fontWeight: 720 }}>{title}</h2>
      {description && <p style={{ margin: '4px 0 14px', color: 'var(--text2)', fontSize: 13 }}>{description}</p>}
      <div style={{ marginTop: description ? 0 : 14 }}>{children}</div>
    </section>
  );
}

function DetailField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div style={{ minWidth: 0 }}>
      <div style={{ color: 'var(--text3)', fontSize: 11, fontWeight: 750, textTransform: 'uppercase', letterSpacing: 0 }}>{label}</div>
      <div style={{ marginTop: 4, minHeight: 20, color: 'var(--text)', fontSize: 13.5, overflowWrap: 'anywhere' }}>{children}</div>
    </div>
  );
}

function safeUrl(value: string | null | undefined) {
  if (!value) return null;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' || url.protocol === 'http:' ? url.toString() : null;
  } catch {
    return null;
  }
}

export default function AdminUserDetailPage() {
  const params = useParams<{ id: string }>();
  const { user: currentAdmin } = useAuth();
  const { showToast } = useToast();
  const requestSeq = useRef(0);
  const mutationRef = useRef(false);
  const suspensionTriggerRef = useRef<HTMLButtonElement>(null);
  const [user, setUser] = useState<AdminUser | null>(null);
  const [roleDraft, setRoleDraft] = useState<AdminUserRole>('user');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ kind: LoadError; message: string } | null>(null);
  const [mutation, setMutation] = useState<'role' | 'suspension' | null>(null);
  const [suspensionDialogOpen, setSuspensionDialogOpen] = useState(false);

  const userID = useMemo(() => params.id?.trim() ?? '', [params.id]);
  const load = useCallback(
    (signal?: AbortSignal) => {
      if (!userID) {
        setLoading(false);
        setError({ kind: 'not-found', message: 'User ID is missing.' });
        return;
      }
      const sequence = requestSeq.current + 1;
      requestSeq.current = sequence;
      setLoading(true);
      setError(null);
      adminUserApi
        .get(userID, signal)
        .then((response) => {
          if (signal?.aborted || requestSeq.current !== sequence) return;
          setUser(response);
          setRoleDraft(response.role);
        })
        .catch((err) => {
          if (!signal?.aborted && requestSeq.current === sequence) setError(errorMessage(err));
        })
        .finally(() => {
          if (!signal?.aborted && requestSeq.current === sequence) setLoading(false);
        });
    },
    [userID],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const closeSuspensionDialog = useCallback(() => {
    setSuspensionDialogOpen(false);
    requestAnimationFrame(() => suspensionTriggerRef.current?.focus());
  }, []);

  const saveRole = async () => {
    if (!user || roleDraft === user.role || mutationRef.current) return;
    mutationRef.current = true;
    setMutation('role');
    try {
      await adminUserApi.assignRole(user.id, { role: roleDraft });
      setUser((current) => (current ? { ...current, role: roleDraft } : current));
      showToast('User role updated', 'success');
    } catch (err) {
      showToast(errorMessage(err).message, 'error');
    } finally {
      mutationRef.current = false;
      setMutation(null);
    }
  };

  const setSuspension = async () => {
    if (!user || mutationRef.current) return;
    mutationRef.current = true;
    setMutation('suspension');
    try {
      const next = await adminUserApi.setSuspension(user.id, !user.is_suspended);
      setUser(next);
      closeSuspensionDialog();
      showToast(next.is_suspended ? 'User suspended and sessions invalidated' : 'User unsuspended. A new login is required.', 'success');
    } catch (err) {
      showToast(errorMessage(err).message, 'error');
    } finally {
      mutationRef.current = false;
      setMutation(null);
    }
  };

  if (loading) {
    return (
      <>
        <AdminPageHeader title="User detail" description="Loading account and profile information." />
        <AdminLoadingState title="Loading user detail" />
      </>
    );
  }
  if (error?.kind === 'forbidden') return <AdminForbiddenState />;
  if (error?.kind === 'not-found') {
    return (
      <>
        <AdminPageHeader title="User detail" description="The requested user account could not be loaded." />
        <section role="alert" style={{ ...adminCard, padding: '42px 20px', textAlign: 'center' }}>
          <h2 style={{ margin: '0 0 6px', fontSize: 17, fontWeight: 720 }}>User not found</h2>
          <p style={{ margin: '0 auto 18px', maxWidth: 420, color: 'var(--text2)', fontSize: 13.5 }}>{error.message}</p>
          <Link href="/admin/users" className="ac-hover-surface2" style={buttonStyles.secondary(38)}>Back to Users</Link>
        </section>
      </>
    );
  }
  if (error) {
    return (
      <>
        <AdminPageHeader title="User detail" description="The requested user account could not be loaded." />
        <AdminApiError title="Could not load user" error={error.message} onRetry={() => load()} />
      </>
    );
  }
  if (!user) return null;

  const profileFacts = [
    ['Country', user.country],
    ['School', user.school],
    ['Company', user.company],
  ].filter((entry): entry is [string, string] => Boolean(entry[1]));
  const links = [
    ['GitHub', safeUrl(user.github_url)],
    ['Website', safeUrl(user.website_url)],
    ['LinkedIn', safeUrl(user.linkedin_url)],
  ].filter((entry): entry is [string, string] => Boolean(entry[1]));
  const suspending = !user.is_suspended;
  const mutationBusy = mutation !== null;
  const isSelf = currentAdmin?.id === user.id;

  return (
    <>
      <AdminPageHeader
        title="User detail"
        description="Inspect account state and make deliberate administrative changes."
        actions={<Link href="/admin/users" className="ac-hover-surface2" style={buttonStyles.secondary(38)}>Back to Users</Link>}
      />

      <section style={{ ...adminCard, padding: 18, marginBottom: 14, display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 14 }}>
        <Avatar user={user} />
        <div style={{ minWidth: 200, flex: 1 }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
            <h2 style={{ margin: 0, fontSize: 21, lineHeight: 1.2, fontWeight: 730 }}>@{user.username}</h2>
            <AdminUserStatusPill user={user} />
          </div>
          {user.full_name && <div style={{ marginTop: 3, color: 'var(--text2)', fontSize: 14 }}>{user.full_name}</div>}
          <div style={{ marginTop: 4, color: 'var(--text2)', fontSize: 13 }}>{user.email}</div>
        </div>
        <RolePill role={user.role} />
      </section>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(285px, 1fr))', gap: 14 }}>
        <DetailSection title="Profile">
          {user.bio && <p style={{ margin: '0 0 14px', color: 'var(--text2)', fontSize: 13.5, lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>{user.bio}</p>}
          {profileFacts.length > 0 && (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: 14 }}>
              {profileFacts.map(([label, value]) => <DetailField key={label} label={label}>{value}</DetailField>)}
            </div>
          )}
          {links.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, marginTop: user.bio || profileFacts.length > 0 ? 16 : 0 }}>
              {links.map(([label, href]) => <a key={label} href={href} target="_blank" rel="noreferrer" className="ac-hover-underline" style={{ fontSize: 13, fontWeight: 650 }}>{label}</a>)}
            </div>
          )}
          {!user.bio && profileFacts.length === 0 && links.length === 0 && <p style={{ margin: 0, color: 'var(--text2)', fontSize: 13.5 }}>No public profile information is available.</p>}
        </DetailSection>

        <DetailSection title="Account">
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: 14 }}>
            <DetailField label="Account status"><AdminUserStatusPill user={user} /></DetailField>
            <DetailField label="Email verification">{user.is_active ? 'Verified' : 'Not verified'}</DetailField>
            <DetailField label="Rating">{user.rating.toLocaleString()}</DetailField>
            <DetailField label="Joined"><DateText value={user.created_at} /></DetailField>
            <DetailField label="Updated"><DateText value={user.updated_at} /></DetailField>
          </div>
          <div style={{ marginTop: 16 }}><DetailField label="User ID"><span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{user.id}</span></DetailField></div>
        </DetailSection>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(285px, 1fr))', gap: 14, marginTop: 14 }}>
        <DetailSection title="Role" description="Role changes apply to the account after you save them.">
          <label style={{ display: 'grid', gap: 6, maxWidth: 260 }}>
            <span style={{ color: 'var(--text2)', fontSize: 12, fontWeight: 650 }}>Assigned role</span>
            <select value={roleDraft} onChange={(event) => setRoleDraft(event.target.value as AdminUserRole)} disabled={mutationBusy} className="ac-field" style={adminField}>
              {ROLES.map((role) => <option key={role} value={role}>{role[0].toUpperCase() + role.slice(1)}</option>)}
            </select>
          </label>
          <div style={{ marginTop: 12 }}>
            <button type="button" onClick={saveRole} disabled={mutationBusy || roleDraft === user.role} className="ac-hover-accent" style={{ ...buttonStyles.primary(38), opacity: roleDraft === user.role ? 0.55 : 1 }}>
              {mutation === 'role' ? 'Saving role...' : 'Save role'}
            </button>
          </div>
        </DetailSection>

        <DetailSection title="Moderation" description={user.is_suspended ? 'The account cannot log in or refresh existing sessions.' : 'Suspension invalidates existing sessions and prevents future login or refresh.'}>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
            <AdminUserStatusPill user={user} />
            {isSelf && <span style={{ color: 'var(--warn)', fontSize: 12.5, fontWeight: 650 }}>This is your account.</span>}
          </div>
          <div style={{ marginTop: 14 }}>
            <button
              ref={suspensionTriggerRef}
              type="button"
              disabled={mutationBusy}
              onClick={() => setSuspensionDialogOpen(true)}
              className={suspending ? 'ac-hover-accent' : 'ac-hover-surface2'}
              style={suspending ? { ...buttonStyles.primary(38), background: 'var(--error)' } : buttonStyles.secondary(38)}
            >
              {suspending ? 'Suspend user' : 'Unsuspend user'}
            </button>
          </div>
        </DetailSection>
      </div>

      {suspensionDialogOpen && (
        <AdminDialog title={`${suspending ? 'Suspend' : 'Unsuspend'} @${user.username}?`} onClose={() => { if (!mutationBusy) closeSuspensionDialog(); }}>
          <p style={{ margin: '0 0 18px', color: 'var(--text2)', fontSize: 13.5, lineHeight: 1.55 }}>
            {suspending
              ? 'This immediately invalidates existing sessions. The user cannot log in or refresh a token until the account is unsuspended.'
              : 'This restores account access, but does not restore old sessions. The user must authenticate again.'}
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" onClick={closeSuspensionDialog} disabled={mutationBusy} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>Cancel</button>
            <button type="button" onClick={setSuspension} disabled={mutationBusy} className={suspending ? 'ac-hover-accent' : 'ac-hover-accent'} style={suspending ? { ...buttonStyles.primary(38), background: 'var(--error)' } : buttonStyles.primary(38)}>
              {mutation === 'suspension' ? (suspending ? 'Suspending...' : 'Unsuspending...') : (suspending ? 'Suspend user' : 'Unsuspend user')}
            </button>
          </div>
        </AdminDialog>
      )}
    </>
  );
}
