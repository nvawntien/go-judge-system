'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import type { CSSProperties, FormEvent } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ApiError, NetworkError, authApi } from '@/lib/api';
import { buttonStyles, Icon, Logo, Spinner, Wordmark } from '@/components/ui';
import { useTheme } from '@/components/ThemeProvider';

type VerifyEmailState =
  | { status: 'checking' }
  | { status: 'success'; message?: string }
  | { status: 'invalid_link'; message?: string }
  | { status: 'expired'; message?: string }
  | { status: 'already_verified'; message?: string }
  | { status: 'error'; message?: string };

type ResendState =
  | { status: 'idle' }
  | { status: 'sending' }
  | { status: 'sent'; message: string }
  | { status: 'error'; message: string };

interface StateCopy {
  tone: 'info' | 'success' | 'warning' | 'error';
  icon: string;
  title: string;
  description: string;
  primary?: { href: string; label: string };
  canRetry: boolean;
  canResend: boolean;
}

const COPY: Record<VerifyEmailState['status'], StateCopy> = {
  checking: {
    tone: 'info',
    icon: '•',
    title: 'Verifying your email',
    description: 'We are confirming your account. This usually takes just a moment.',
    canRetry: false,
    canResend: false,
  },
  success: {
    tone: 'success',
    icon: '✓',
    title: 'Email verified',
    description: 'Your account is ready. You can now sign in and start solving problems.',
    primary: { href: '/login', label: 'Continue to login' },
    canRetry: false,
    canResend: false,
  },
  invalid_link: {
    tone: 'warning',
    icon: '!',
    title: 'Verification link unavailable',
    description: 'This verification link is incomplete, invalid, already used, or expired.',
    canRetry: false,
    canResend: true,
  },
  expired: {
    tone: 'warning',
    icon: '!',
    title: 'Verification link expired',
    description: 'This link is no longer valid. Request a fresh verification email to continue.',
    canRetry: false,
    canResend: true,
  },
  already_verified: {
    tone: 'success',
    icon: '✓',
    title: 'Email already verified',
    description: 'This account has already been activated. You can sign in normally.',
    primary: { href: '/login', label: 'Continue to login' },
    canRetry: false,
    canResend: false,
  },
  error: {
    tone: 'error',
    icon: '!',
    title: 'We could not verify your email right now',
    description: 'The verification service is temporarily unavailable. Please try again.',
    canRetry: true,
    canResend: false,
  },
};

const TONE_STYLE: Record<StateCopy['tone'], { fg: string; bg: string; border: string }> = {
  info: { fg: 'var(--accent-fg)', bg: 'var(--accent-soft)', border: 'var(--accent-soft2)' },
  success: { fg: 'var(--success)', bg: 'var(--success-bg)', border: 'var(--success)' },
  warning: { fg: 'var(--warn)', bg: 'var(--warn-bg)', border: 'var(--warn)' },
  error: { fg: 'var(--error)', bg: 'var(--error-bg)', border: 'var(--error)' },
};

function stateMessage(state: VerifyEmailState): string | undefined {
  return 'message' in state ? state.message : undefined;
}

function mapVerifyError(err: unknown): VerifyEmailState {
  if (err instanceof DOMException && err.name === 'AbortError') {
    return { status: 'checking' };
  }
  if (err instanceof NetworkError) {
    return { status: 'error', message: 'AstraCode is temporarily unreachable. Check your connection and try again.' };
  }
  if (err instanceof ApiError) {
    if (err.code === 40102) {
      return { status: 'expired', message: 'This verification link has expired.' };
    }
    if (err.code === 40900 || err.message.toLowerCase().includes('already active')) {
      return { status: 'already_verified', message: 'This account has already been activated.' };
    }
    if (err.code === 40101 || err.httpStatus === 400 || err.httpStatus === 401 || err.httpStatus === 404) {
      return { status: 'invalid_link', message: 'This verification link is invalid or has expired.' };
    }
  }
  return { status: 'error' };
}

function fieldStyle(invalid: boolean): CSSProperties {
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

function tokenFromHash(): string {
  if (typeof window === 'undefined' || !window.location.hash) return '';

  const hashParams = new URLSearchParams(window.location.hash.slice(1));
  return hashParams.get('token')?.trim() || '';
}

export function VerifyEmailView() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { resolved, toggle } = useTheme();
  const [state, setState] = useState<VerifyEmailState>({ status: 'checking' });
  const [resendEmail, setResendEmail] = useState('');
  const [resendState, setResendState] = useState<ResendState>({ status: 'idle' });
  const [resendError, setResendError] = useState('');
  const tokenRef = useRef<string | null>(null);
  const activeRequestRef = useRef<string | null>(null);
  const requestStartedRef = useRef(false);
  const verifyAbortRef = useRef<AbortController | null>(null);
  const resendAbortRef = useRef<AbortController | null>(null);

  const startVerification = useCallback(
    (token: string) => {
      if (activeRequestRef.current === token) return;

      activeRequestRef.current = token;
      requestStartedRef.current = true;
      tokenRef.current = token;
      setState({ status: 'checking' });

      verifyAbortRef.current?.abort();
      const controller = new AbortController();
      verifyAbortRef.current = controller;

      void authApi
        .verifyEmail({ token }, controller.signal)
        .then(() => {
          setState({
            status: 'success',
            message: 'Email verified successfully, your account is now active.',
          });
        })
        .catch((err: unknown) => {
          const nextState = mapVerifyError(err);
          if (nextState.status !== 'checking') setState(nextState);
        })
        .finally(() => {
          if (activeRequestRef.current === token) activeRequestRef.current = null;
        });
    },
    [],
  );

  useEffect(() => {
    const token = searchParams.get('token')?.trim() || tokenFromHash();

    if (!token) {
      if (!tokenRef.current && !requestStartedRef.current) {
        setState({
          status: 'invalid_link',
          message: 'This verification link is incomplete or invalid.',
        });
      }
      return;
    }

    if (tokenRef.current !== token || !requestStartedRef.current) {
      startVerification(token);
      router.replace('/verify-email', { scroll: false });
    }
  }, [router, searchParams, startVerification]);

  useEffect(() => {
    return () => {
      resendAbortRef.current?.abort();
    };
  }, []);

  const copy = COPY[state.status];
  const tone = TONE_STYLE[copy.tone];
  const message = stateMessage(state) ?? copy.description;
  const canSubmitResend = resendEmail.trim().length > 0 && resendState.status !== 'sending';
  const hasTopActions = Boolean(copy.primary || copy.canRetry);

  const retryVerification = () => {
    const token = tokenRef.current;
    if (!token) {
      setState({ status: 'invalid_link', message: 'This verification link is incomplete or invalid.' });
      return;
    }
    startVerification(token);
  };

  const resendVerification = async (event: FormEvent) => {
    event.preventDefault();
    const email = resendEmail.trim();
    if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) {
      setResendError('Enter a valid email address');
      return;
    }

    setResendError('');
    setResendState({ status: 'sending' });
    resendAbortRef.current?.abort();
    const controller = new AbortController();
    resendAbortRef.current = controller;

    try {
      await authApi.resendVerification(email, controller.signal);
      setResendState({
        status: 'sent',
        message: 'If that address matches an inactive account, a new verification link is on its way.',
      });
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      if (err instanceof NetworkError) {
        setResendState({ status: 'error', message: 'AstraCode is temporarily unreachable. Check your connection and try again.' });
      } else if (err instanceof ApiError && err.httpStatus === 429) {
        setResendState({ status: 'error', message: 'Please wait a moment before requesting another link.' });
      } else {
        setResendState({ status: 'error', message: 'We could not send a new link right now.' });
      }
    }
  };

  const statusIcon = useMemo(() => {
    if (state.status === 'checking') return <Spinner size={22} color="currentColor" />;
    return <span aria-hidden="true">{copy.icon}</span>;
  }, [copy.icon, state.status]);

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
          ...buttonStyles.iconButton(36),
        }}
      >
        {resolved === 'dark' ? <Icon.Sun /> : <Icon.Moon />}
      </button>

      <div style={{ width: '100%', maxWidth: 430 }}>
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
          aria-labelledby="verify-email-title"
          style={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 16,
            boxShadow: 'var(--shadow)',
            padding: 24,
            textAlign: 'center',
          }}
        >
          <div
            aria-hidden="true"
            style={{
              width: 52,
              height: 52,
              borderRadius: 16,
              margin: '0 auto 16px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: tone.bg,
              border: `1px solid ${tone.border}`,
              color: tone.fg,
              fontSize: 24,
              fontWeight: 750,
            }}
          >
            {statusIcon}
          </div>

          <div role="status" aria-live="polite" aria-atomic="true">
            <h1
              id="verify-email-title"
              style={{ margin: '0 0 8px', fontSize: 20, lineHeight: 1.2, fontWeight: 700 }}
            >
              {copy.title}
            </h1>
            <p style={{ margin: 0, color: 'var(--text3)', fontSize: 13.5 }}>{message}</p>
          </div>

          {hasTopActions && (
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                justifyContent: 'center',
                gap: 10,
                marginTop: 22,
              }}
            >
              {copy.primary && (
                <Link href={copy.primary.href} className="ac-hover-accent" style={buttonStyles.primary(40)}>
                  {copy.primary.label}
                </Link>
              )}
              {copy.canRetry && (
                <button type="button" onClick={retryVerification} className="ac-hover-accent" style={buttonStyles.primary(40)}>
                  Try again
                </button>
              )}
              {copy.canRetry && (
                <Link href="/login" className="ac-hover-surface2" style={buttonStyles.secondary(40)}>
                  Back to login
                </Link>
              )}
            </div>
          )}

          {copy.canResend && (
            <form
              onSubmit={resendVerification}
              style={{
                marginTop: 22,
                paddingTop: 20,
                borderTop: '1px solid var(--border)',
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
                textAlign: 'left',
              }}
            >
              <label
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 5,
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--text2)',
                }}
              >
                Email address
                <input
                  value={resendEmail}
                  onChange={(event) => {
                    setResendEmail(event.target.value);
                    setResendError('');
                    if (resendState.status !== 'sending') setResendState({ status: 'idle' });
                  }}
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                  aria-invalid={Boolean(resendError)}
                  className="ac-field"
                  style={fieldStyle(Boolean(resendError))}
                />
              </label>

              {resendError && (
                <span role="alert" style={{ fontSize: 11.5, fontWeight: 500, color: 'var(--error)' }}>
                  {resendError}
                </span>
              )}
              {resendState.status === 'sent' && (
                <p role="status" style={{ margin: 0, fontSize: 12.5, color: 'var(--success)' }}>
                  {resendState.message}
                </p>
              )}
              {resendState.status === 'error' && (
                <p role="alert" style={{ margin: 0, fontSize: 12.5, color: 'var(--error)' }}>
                  {resendState.message}
                </p>
              )}

              <button
                type="submit"
                disabled={!canSubmitResend}
                className="ac-hover-accent"
                style={{
                  ...buttonStyles.primary(40),
                  opacity: canSubmitResend ? 1 : 0.65,
                  cursor: canSubmitResend ? 'pointer' : 'not-allowed',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 8,
                }}
              >
                {resendState.status === 'sending' && <Spinner size={15} color="currentColor" />}
                Send verification email
              </button>

              <Link
                href="/login"
                className="ac-hover-surface2"
                style={{ ...buttonStyles.secondary(40), alignSelf: 'center' }}
              >
                Back to login
              </Link>
            </form>
          )}
        </section>

        <p style={{ margin: '16px 0 0', textAlign: 'center', fontSize: 11.5, color: 'var(--text3)' }}>
          Verification links are never stored in this browser.
        </p>
      </div>
    </main>
  );
}
