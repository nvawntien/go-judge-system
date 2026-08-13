'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';
import { Icon, Logo, Spinner, Wordmark } from '@/components/ui';
import { ApiError, NetworkError, authApi } from '@/lib/api';

function ResetPasswordView() {
  const searchParams = useSearchParams();
  const { resolved, toggle } = useTheme();
  const token = searchParams.get('token')?.trim() ?? '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [complete, setComplete] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy || !token) return;
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    if (password !== confirm) {
      setError('Passwords do not match.');
      return;
    }

    setBusy(true);
    setError('');
    try {
      await authApi.resetPassword({ token, new_password: password, confirm_password: confirm });
      setComplete(true);
      setPassword('');
      setConfirm('');
    } catch (cause) {
      if (cause instanceof NetworkError) setError('Cannot reach the API gateway. Try again.');
      else if (cause instanceof ApiError) setError(cause.message || 'This reset link is invalid or has expired.');
      else setError('Could not reset your password. Try again.');
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="ac-auth-page">
      <button
        type="button"
        onClick={toggle}
        aria-label={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
        className="ac-icon-button ac-auth-theme-toggle"
      >
        {resolved === 'dark' ? <Icon.Sun /> : <Icon.Moon />}
      </button>

      <div className="ac-auth-layout">
        <Link href="/" className="ac-auth-brand" aria-label="AstraCode home" style={{ color: 'var(--text)', textDecoration: 'none' }}>
          <Logo size={30} />
          <Wordmark fontSize={20} />
        </Link>

        <section className="ac-auth-card ac-reset-card">
          <div className="ac-auth-form">
            <div>
              <span className="ac-eyebrow">Account recovery</span>
              <h1 className="ac-reset-title">Choose a new password</h1>
              <p className="ac-reset-copy">A successful reset revokes existing sessions. Sign in again with your new password.</p>
            </div>

            {!token ? (
              <div role="alert" className="ac-inline-message ac-inline-message-error">
                This reset link is missing its token. Request a new link from the sign-in page.
              </div>
            ) : complete ? (
              <div role="status" className="ac-inline-message ac-inline-message-success">
                Password updated. All previous sessions have been revoked.
              </div>
            ) : (
              <form onSubmit={submit} className="ac-reset-form">
                <label className="ac-form-label">
                  New password
                  <span className="ac-password-field">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                      aria-invalid={Boolean(error)}
                      className="ac-field"
                    />
                    <button type="button" onClick={() => setShowPassword((value) => !value)} aria-label={showPassword ? 'Hide password' : 'Show password'}>
                      <Icon.Eye />
                    </button>
                  </span>
                </label>
                <label className="ac-form-label">
                  Confirm new password
                  <input
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="new-password"
                    value={confirm}
                    onChange={(event) => setConfirm(event.target.value)}
                    aria-invalid={Boolean(error)}
                    className="ac-field"
                  />
                </label>
                {error && <p role="alert" className="ac-form-error">{error}</p>}
                <button type="submit" disabled={busy} className="ac-button ac-button-primary ac-reset-submit">
                  {busy && <Spinner size={14} color="currentColor" />}
                  Update password
                </button>
              </form>
            )}

            <Link href="/login" className="ac-button ac-button-secondary">Back to sign in</Link>
          </div>
        </section>
      </div>
    </main>
  );
}

export default function ResetPasswordPage() {
  return <Suspense fallback={null}><ResetPasswordView /></Suspense>;
}
