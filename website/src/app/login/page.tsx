'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useAuth } from '@/components/AuthProvider';
import { useTheme } from '@/components/ThemeProvider';
import { useToast } from '@/components/ToastProvider';
import { ApiError, NetworkError, authApi } from '@/lib/api';
import { Icon, Logo, Spinner, Wordmark } from '@/components/ui';

type Mode = 'login' | 'register' | 'forgot';

const COPY: Record<Mode, { title: string; sub: string; cta: string }> = {
  login: {
    title: 'Welcome back',
    sub: 'Sign in to run code, submit solutions, and keep your streak alive.',
    cta: 'Sign in',
  },
  register: {
    title: 'Create your account',
    sub: 'A verification email is sent to confirm your address.',
    cta: 'Create account',
  },
  forgot: {
    title: 'Reset your password',
    sub: 'We email a reset link if an account matches this address.',
    cta: 'Send reset link',
  },
};

function labelStyle(): React.CSSProperties {
  return {
    display: 'flex',
    flexDirection: 'column',
    gap: 5,
    fontSize: 12,
    fontWeight: 600,
    color: 'var(--text2)',
  };
}

function fieldStyle(invalid: boolean): React.CSSProperties {
  return {
    height: 40,
    width: '100%',
    boxSizing: 'border-box',
    borderRadius: 8,
    border: `1px solid ${invalid ? 'var(--error)' : 'var(--border2)'}`,
    background: 'var(--surface)',
    padding: '0 12px',
    fontSize: 13.5,
    color: 'var(--text)',
  };
}

function AuthCard() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user, loading: authLoading, login, register } = useAuth();
  const { resolved, toggle } = useTheme();
  const { showToast } = useToast();

  const nextPath = searchParams.get('next') || '/';
  const [mode, setMode] = useState<Mode>(
    searchParams.get('mode') === 'register' ? 'register' : 'login',
  );
  const [identifier, setIdentifier] = useState('');
  const [fullName, setFullName] = useState('');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPw, setShowPw] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  // Already signed in — nothing to do on this screen.
  useEffect(() => {
    if (!authLoading && user) router.replace(nextPath);
  }, [authLoading, user, router, nextPath]);

  const copy = COPY[mode];

  const validate = (): boolean => {
    const next: Record<string, string> = {};
    if (mode === 'login') {
      if (!identifier.trim()) next.identifier = 'Enter your email or username';
      if (!password) next.password = 'Enter your password';
    } else if (mode === 'register') {
      if (!fullName.trim()) next.fullName = 'Enter your full name';
      if (username.trim().length < 3) next.username = 'Username must be at least 3 characters';
      if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) next.email = 'Enter a valid email address';
      if (password.length < 8) next.password = 'Password must be at least 8 characters';
    } else if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) {
      next.email = 'Enter a valid email address';
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy || !validate()) return;

    setBusy(true);
    try {
      if (mode === 'login') {
        const me = await login({ identifier: identifier.trim(), password });
        showToast(`Welcome back, ${me.full_name || me.username}`, 'success');
        router.replace(nextPath);
      } else if (mode === 'register') {
        await register({
          full_name: fullName.trim(),
          username: username.trim(),
          email: email.trim(),
          password,
        });
        showToast('Account created — check your email to verify it', 'success');
        setMode('login');
        setIdentifier(email.trim());
        setPassword('');
      } else {
        await authApi.forgotPassword(email.trim());
        showToast('If that address exists, a reset link is on its way', 'success');
        setMode('login');
      }
    } catch (err) {
      if (err instanceof NetworkError) {
        showToast('Cannot reach the API gateway — is it running on :8080?', 'error');
      } else if (err instanceof ApiError) {
        const message = err.message || 'Request failed';
        if (mode === 'login' && err.httpStatus === 401) {
          setErrors({ password: 'Incorrect email/username or password' });
        } else if (err.httpStatus === 409) {
          setErrors({ email: message, username: message });
        } else {
          showToast(message, 'error');
        }
      } else {
        showToast('Something went wrong', 'error');
      }
    } finally {
      setBusy(false);
    }
  };

  const tabStyle = (active: boolean): React.CSSProperties => ({
    flex: 1,
    height: 46,
    border: 'none',
    borderBottom: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
    background: 'none',
    cursor: 'pointer',
    fontSize: 13,
    fontWeight: 600,
    color: active ? 'var(--text)' : 'var(--text3)',
  });

  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        animation: 'acFadeUp .3s ease',
        position: 'relative',
      }}
    >
      <button
        type="button"
        onClick={toggle}
        aria-label={resolved === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
        className="ac-hover-surface2"
        style={{
          position: 'absolute',
          top: 18,
          right: 18,
          width: 36,
          height: 36,
          borderRadius: 8,
          border: '1px solid var(--border)',
          background: 'var(--surface)',
          color: 'var(--text2)',
          cursor: 'pointer',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {resolved === 'dark' ? <Icon.Sun /> : <Icon.Moon />}
      </button>

      <div style={{ width: '100%', maxWidth: 400 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 9,
            marginBottom: 22,
          }}
        >
          <Logo size={30} />
          <Wordmark fontSize={20} />
        </div>

        <section
          style={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 16,
            boxShadow: 'var(--shadow)',
            overflow: 'hidden',
          }}
        >
          <div
            role="tablist"
            aria-label="Sign in or register"
            style={{ display: 'flex', borderBottom: '1px solid var(--border)' }}
          >
            <button
              type="button"
              role="tab"
              aria-selected={mode !== 'register'}
              onClick={() => {
                setMode('login');
                setErrors({});
              }}
              style={tabStyle(mode !== 'register')}
            >
              Sign in
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={mode === 'register'}
              onClick={() => {
                setMode('register');
                setErrors({});
              }}
              style={tabStyle(mode === 'register')}
            >
              Create account
            </button>
          </div>

          <form
            onSubmit={onSubmit}
            style={{ padding: 22, display: 'flex', flexDirection: 'column', gap: 14 }}
          >
            <div>
              <h1 style={{ margin: '0 0 2px', fontSize: 17, fontWeight: 650, letterSpacing: '-0.01em' }}>
                {copy.title}
              </h1>
              <p style={{ margin: 0, fontSize: 12.5, color: 'var(--text3)' }}>{copy.sub}</p>
            </div>

            {mode === 'login' && (
              <label style={labelStyle()}>
                Email or username
                <input
                  value={identifier}
                  onChange={(event) => setIdentifier(event.target.value)}
                  type="text"
                  autoComplete="username"
                  placeholder="you@example.com"
                  aria-invalid={Boolean(errors.identifier)}
                  className="ac-field"
                  style={fieldStyle(Boolean(errors.identifier))}
                />
                {errors.identifier && (
                  <span role="alert" style={{ fontSize: 11.5, fontWeight: 500, color: 'var(--error)' }}>
                    {errors.identifier}
                  </span>
                )}
              </label>
            )}

            {mode === 'register' && (
              <>
                <label style={labelStyle()}>
                  Full name
                  <input
                    value={fullName}
                    onChange={(event) => setFullName(event.target.value)}
                    autoComplete="name"
                    placeholder="Alex Nguyen"
                    aria-invalid={Boolean(errors.fullName)}
                    className="ac-field"
                    style={fieldStyle(Boolean(errors.fullName))}
                  />
                  {errors.fullName && (
                    <span role="alert" style={{ fontSize: 11.5, color: 'var(--error)' }}>
                      {errors.fullName}
                    </span>
                  )}
                </label>
                <label style={labelStyle()}>
                  Username
                  <input
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    autoComplete="username"
                    placeholder="alex_codes"
                    aria-invalid={Boolean(errors.username)}
                    className="ac-field"
                    style={{ ...fieldStyle(Boolean(errors.username)), fontFamily: 'var(--font-mono)' }}
                  />
                  {errors.username && (
                    <span role="alert" style={{ fontSize: 11.5, color: 'var(--error)' }}>
                      {errors.username}
                    </span>
                  )}
                </label>
              </>
            )}

            {(mode === 'register' || mode === 'forgot') && (
              <label style={labelStyle()}>
                Email
                <input
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                  aria-invalid={Boolean(errors.email)}
                  className="ac-field"
                  style={fieldStyle(Boolean(errors.email))}
                />
                {errors.email && (
                  <span role="alert" style={{ fontSize: 11.5, color: 'var(--error)' }}>
                    {errors.email}
                  </span>
                )}
              </label>
            )}

            {mode !== 'forgot' && (
              <label style={labelStyle()}>
                Password
                <span style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                  <input
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    type={showPw ? 'text' : 'password'}
                    autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                    placeholder="••••••••"
                    aria-invalid={Boolean(errors.password)}
                    className="ac-field"
                    style={{ ...fieldStyle(Boolean(errors.password)), padding: '0 44px 0 12px' }}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPw(!showPw)}
                    aria-label={showPw ? 'Hide password' : 'Show password'}
                    className="ac-hover-text"
                    style={{
                      position: 'absolute',
                      right: 4,
                      width: 34,
                      height: 34,
                      border: 'none',
                      background: 'none',
                      color: 'var(--text3)',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <Icon.Eye />
                  </button>
                </span>
                {errors.password && (
                  <span role="alert" style={{ fontSize: 11.5, fontWeight: 500, color: 'var(--error)' }}>
                    {errors.password}
                  </span>
                )}
              </label>
            )}

            {mode === 'login' && (
              <button
                type="button"
                onClick={() => {
                  setMode('forgot');
                  setErrors({});
                }}
                style={{
                  alignSelf: 'flex-start',
                  border: 'none',
                  background: 'none',
                  padding: 0,
                  fontSize: 12,
                  fontWeight: 550,
                  color: 'var(--accent-fg)',
                  cursor: 'pointer',
                }}
              >
                Forgot password?
              </button>
            )}

            {mode === 'forgot' && (
              <button
                type="button"
                onClick={() => {
                  setMode('login');
                  setErrors({});
                }}
                style={{
                  alignSelf: 'flex-start',
                  border: 'none',
                  background: 'none',
                  padding: 0,
                  fontSize: 12,
                  fontWeight: 550,
                  color: 'var(--accent-fg)',
                  cursor: 'pointer',
                }}
              >
                ← Back to sign in
              </button>
            )}

            <button
              type="submit"
              disabled={busy}
              className="ac-hover-accent"
              style={{
                height: 42,
                border: 'none',
                borderRadius: 9,
                background: 'var(--accent)',
                color: 'var(--accent-ink)',
                fontSize: 13.5,
                fontWeight: 600,
                cursor: busy ? 'progress' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 8,
                transition: 'background .15s',
                opacity: busy ? 0.75 : 1,
              }}
            >
              {busy && <Spinner size={15} color="currentColor" />}
              {copy.cta}
            </button>
          </form>
        </section>

        <p style={{ margin: '16px 0 0', textAlign: 'center', fontSize: 11.5, color: 'var(--text3)' }}>
          Sessions are stored in HttpOnly cookies issued by the auth service.
        </p>
      </div>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <AuthCard />
    </Suspense>
  );
}
