'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useRef, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { useAuth } from '@/components/AuthProvider';
import { useTheme, type ThemePreference } from '@/components/ThemeProvider';
import { useToast } from '@/components/ToastProvider';
import { Card, Spinner } from '@/components/ui';
import { API_BASE_URL, ApiError, NetworkError, authApi, userApi } from '@/lib/api';
import { avatarUrl, initials } from '@/lib/format';
import type { UpdateProfileRequest } from '@/lib/types';

interface Fields {
  full_name: string;
  country: string;
  bio: string;
  school: string;
  company: string;
  github_url: string;
  website_url: string;
  linkedin_url: string;
}

const EMPTY: Fields = {
  full_name: '',
  country: '',
  bio: '',
  school: '',
  company: '',
  github_url: '',
  website_url: '',
  linkedin_url: '',
};

export default function SettingsPage() {
  const router = useRouter();
  const { user, loading: authLoading, setUser } = useAuth();
  const { preference, setPreference } = useTheme();
  const { showToast } = useToast();
  const fileRef = useRef<HTMLInputElement>(null);

  const [fields, setFields] = useState<Fields>(EMPTY);
  const [saved, setSaved] = useState<Fields>(EMPTY);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);

  const [pwOpen, setPwOpen] = useState(false);
  const [pwFields, setPwFields] = useState({ current: '', next: '', confirm: '' });
  const [pwSaving, setPwSaving] = useState(false);

  useEffect(() => {
    if (!authLoading && !user) router.replace('/login?next=/settings');
  }, [authLoading, user, router]);

  useEffect(() => {
    if (!user) return;
    const next: Fields = {
      full_name: user.full_name ?? '',
      country: user.country ?? '',
      bio: user.bio ?? '',
      school: user.school ?? '',
      company: user.company ?? '',
      github_url: user.github_url ?? '',
      website_url: user.website_url ?? '',
      linkedin_url: user.linkedin_url ?? '',
    };
    setFields(next);
    setSaved(next);
  }, [user]);

  const dirty = useMemo(
    () => (Object.keys(fields) as (keyof Fields)[]).some((key) => fields[key] !== saved[key]),
    [fields, saved],
  );

  const set = (key: keyof Fields) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setFields((current) => ({ ...current, [key]: event.target.value }));

  const onSave = async () => {
    if (saving || !dirty) return;
    setSaving(true);

    // Only send what changed; empty strings clear the field server-side.
    const patch: UpdateProfileRequest = {};
    (Object.keys(fields) as (keyof Fields)[]).forEach((key) => {
      if (fields[key] !== saved[key]) patch[key] = fields[key];
    });

    try {
      const updated = await userApi.updateProfile(patch);
      setUser(updated);
      setSaved(fields);
      showToast('Profile saved', 'success');
    } catch (err) {
      if (err instanceof NetworkError) showToast('Cannot reach the API gateway', 'error');
      else if (err instanceof ApiError) showToast(err.message || 'Save failed', 'error');
      else showToast('Save failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  const onAvatar = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;

    setUploading(true);
    try {
      await userApi.uploadAvatar(file);
      const me = await userApi.me();
      setUser(me);
      showToast('Avatar updated', 'success');
    } catch (err) {
      if (err instanceof ApiError) showToast(err.message || 'Upload failed', 'error');
      else showToast('Upload failed', 'error');
    } finally {
      setUploading(false);
    }
  };

  const onChangePassword = async (event: React.FormEvent) => {
    event.preventDefault();
    if (pwSaving) return;
    if (pwFields.next !== pwFields.confirm) {
      showToast('New passwords do not match', 'error');
      return;
    }
    if (pwFields.next.length < 8) {
      showToast('New password must be at least 8 characters', 'error');
      return;
    }

    setPwSaving(true);
    try {
      await authApi.changePassword({
        current_password: pwFields.current,
        new_password: pwFields.next,
        confirm_password: pwFields.confirm,
      });
      showToast('Password changed', 'success');
      setPwFields({ current: '', next: '', confirm: '' });
      setPwOpen(false);
    } catch (err) {
      if (err instanceof ApiError) showToast(err.message || 'Could not change password', 'error');
      else showToast('Could not change password', 'error');
    } finally {
      setPwSaving(false);
    }
  };

  if (!user) return null;
  const avatar = avatarUrl(user.avatar_url, API_BASE_URL);

  return (
    <AppShell maxWidth={760}>
      <h1 style={{ margin: 0, fontSize: 22, fontWeight: 650, letterSpacing: '-0.02em' }}>
        Profile settings
      </h1>
      <p style={{ margin: '4px 0 20px', color: 'var(--text2)', fontSize: 13.5 }}>
        This information appears on your public profile.
      </p>

      {dirty && (
        <div
          role="status"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            background: 'var(--warn-bg)',
            border: '1px solid var(--warn)',
            borderRadius: 10,
            padding: '10px 14px',
            marginBottom: 16,
            animation: 'acFadeUp .2s ease',
          }}
        >
          <span style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--warn)' }} />
          <span style={{ fontSize: 12.5, fontWeight: 550, color: 'var(--text)' }}>
            You have unsaved changes
          </span>
        </div>
      )}

      <Card padding={22} style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          {avatar ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={avatar}
              alt=""
              width={64}
              height={64}
              style={{
                width: 64,
                height: 64,
                borderRadius: '50%',
                objectFit: 'cover',
                border: '2px solid var(--accent-soft2)',
              }}
            />
          ) : (
            <span
              aria-hidden="true"
              style={{
                width: 64,
                height: 64,
                borderRadius: '50%',
                background: 'var(--accent-soft)',
                border: '2px solid var(--accent-soft2)',
                color: 'var(--accent-fg)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 20,
                fontWeight: 650,
              }}
            >
              {initials(user.full_name || user.username)}
            </span>
          )}
          <div>
            <input
              ref={fileRef}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onChange={onAvatar}
              style={{ display: 'none' }}
            />
            <button
              type="button"
              onClick={() => fileRef.current?.click()}
              disabled={uploading}
              className="ac-hover-surface2"
              style={{
                height: 34,
                padding: '0 14px',
                border: '1px solid var(--border2)',
                borderRadius: 8,
                background: 'var(--surface)',
                color: 'var(--text)',
                fontSize: 12.5,
                fontWeight: 600,
                cursor: uploading ? 'progress' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              {uploading && <Spinner size={12} />}
              Upload new avatar
            </button>
            <p style={{ margin: '6px 0 0', fontSize: 11.5, color: 'var(--text3)' }}>
              PNG, JPG or WebP, at least 240×240px.
            </p>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(240px,1fr))', gap: 14 }}>
          <Field label="Full name" value={fields.full_name} onChange={set('full_name')} />
          <Field label="Country" value={fields.country} onChange={set('country')} />
        </div>

        <label style={labelStyle}>
          Biography
          <textarea
            value={fields.bio}
            onChange={set('bio')}
            rows={3}
            className="ac-field"
            style={{
              resize: 'vertical',
              borderRadius: 8,
              border: '1px solid var(--border2)',
              background: 'var(--surface)',
              padding: '10px 12px',
              fontSize: 13,
              color: 'var(--text)',
              lineHeight: 1.5,
              fontFamily: 'inherit',
            }}
          />
        </label>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(240px,1fr))', gap: 14 }}>
          <Field label="School" value={fields.school} onChange={set('school')} />
          <Field
            label="Company"
            value={fields.company}
            onChange={set('company')}
            placeholder="Where you work"
          />
          <Field label="GitHub URL" value={fields.github_url} onChange={set('github_url')} mono />
          <Field
            label="Website"
            value={fields.website_url}
            onChange={set('website_url')}
            placeholder="https://"
            mono
          />
          <Field
            label="LinkedIn URL"
            value={fields.linkedin_url}
            onChange={set('linkedin_url')}
            placeholder="https://linkedin.com/in/…"
            mono
          />
        </div>

        <div>
          <span style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--text2)', marginBottom: 6 }}>
            Theme preference
          </span>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {(
              [
                { value: 'light', label: 'Light' },
                { value: 'dark', label: 'Dark' },
                { value: 'system', label: 'Follow system' },
              ] as { value: ThemePreference; label: string }[]
            ).map((option) => {
              const active = preference === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setPreference(option.value)}
                  className="ac-hover-surface2"
                  style={{
                    height: 36,
                    padding: '0 16px',
                    border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
                    borderRadius: 8,
                    background: active ? 'var(--accent-soft)' : 'var(--surface)',
                    color: active ? 'var(--accent-fg)' : 'var(--text2)',
                    fontSize: 12.5,
                    fontWeight: 600,
                    cursor: 'pointer',
                  }}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 8,
            paddingTop: 14,
            borderTop: '1px solid var(--border)',
          }}
        >
          <button
            type="button"
            onClick={() => setFields(saved)}
            disabled={!dirty}
            className="ac-hover-surface2"
            style={{
              height: 38,
              padding: '0 16px',
              border: '1px solid var(--border)',
              borderRadius: 8,
              background: 'var(--surface)',
              color: 'var(--text2)',
              fontSize: 13,
              fontWeight: 600,
              cursor: dirty ? 'pointer' : 'not-allowed',
              opacity: dirty ? 1 : 0.5,
            }}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onSave}
            disabled={!dirty || saving}
            className="ac-hover-accent"
            style={{
              height: 38,
              padding: '0 18px',
              border: 'none',
              borderRadius: 8,
              background: 'var(--accent)',
              color: 'var(--accent-ink)',
              fontSize: 13,
              fontWeight: 600,
              cursor: dirty && !saving ? 'pointer' : 'not-allowed',
              opacity: dirty && !saving ? 1 : 0.6,
              display: 'flex',
              alignItems: 'center',
              gap: 8,
            }}
          >
            {saving && <Spinner size={13} color="currentColor" />}
            Save changes
          </button>
        </div>
      </Card>

      <Card padding={22} style={{ marginTop: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 220 }}>
            <h2 style={{ margin: '0 0 3px', fontSize: 14, fontWeight: 650 }}>Password</h2>
            <p style={{ margin: 0, fontSize: 12.5, color: 'var(--text3)' }}>
              Signed in as <span style={{ fontFamily: 'var(--font-mono)' }}>{user.email}</span>
            </p>
          </div>
          <button
            type="button"
            onClick={() => setPwOpen(!pwOpen)}
            aria-expanded={pwOpen}
            className="ac-hover-surface2"
            style={{
              height: 34,
              padding: '0 14px',
              border: '1px solid var(--border2)',
              borderRadius: 8,
              background: 'var(--surface)',
              color: 'var(--text)',
              fontSize: 12.5,
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            {pwOpen ? 'Cancel' : 'Change password'}
          </button>
        </div>

        {pwOpen && (
          <form
            onSubmit={onChangePassword}
            style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 12 }}
          >
            <label style={labelStyle}>
              Current password
              <input
                type="password"
                autoComplete="current-password"
                value={pwFields.current}
                onChange={(event) => setPwFields({ ...pwFields, current: event.target.value })}
                className="ac-field"
                style={inputStyle()}
              />
            </label>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(220px,1fr))', gap: 12 }}>
              <label style={labelStyle}>
                New password
                <input
                  type="password"
                  autoComplete="new-password"
                  value={pwFields.next}
                  onChange={(event) => setPwFields({ ...pwFields, next: event.target.value })}
                  className="ac-field"
                  style={inputStyle()}
                />
              </label>
              <label style={labelStyle}>
                Confirm new password
                <input
                  type="password"
                  autoComplete="new-password"
                  value={pwFields.confirm}
                  onChange={(event) => setPwFields({ ...pwFields, confirm: event.target.value })}
                  className="ac-field"
                  style={inputStyle()}
                />
              </label>
            </div>
            <button
              type="submit"
              disabled={pwSaving}
              className="ac-hover-accent"
              style={{
                alignSelf: 'flex-start',
                height: 36,
                padding: '0 16px',
                border: 'none',
                borderRadius: 8,
                background: 'var(--accent)',
                color: 'var(--accent-ink)',
                fontSize: 12.5,
                fontWeight: 600,
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              {pwSaving && <Spinner size={12} color="currentColor" />}
              Update password
            </button>
          </form>
        )}
      </Card>
    </AppShell>
  );
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  mono = false,
}: {
  label: string;
  value: string;
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  mono?: boolean;
}) {
  return (
    <label style={labelStyle}>
      {label}
      <input
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        className="ac-field"
        style={inputStyle(mono)}
      />
    </label>
  );
}

const labelStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 5,
  fontSize: 12,
  fontWeight: 600,
  color: 'var(--text2)',
};

function inputStyle(mono = false): React.CSSProperties {
  return {
    height: 38,
    borderRadius: 8,
    border: '1px solid var(--border2)',
    background: 'var(--surface)',
    padding: '0 12px',
    fontSize: mono ? 12 : 13,
    color: 'var(--text)',
    fontFamily: mono ? 'var(--font-mono)' : 'inherit',
  };
}
