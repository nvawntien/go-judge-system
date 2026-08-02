'use client';

import Link from 'next/link';
import { useEffect, useRef, type CSSProperties, type ReactNode } from 'react';
import { buttonStyles, CodePath, ErrorState, SkeletonBar, Spinner } from '@/components/ui';

export function AdminPageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'flex-end',
        justifyContent: 'space-between',
        gap: 14,
        marginBottom: 18,
      }}
    >
      <div style={{ minWidth: 0 }}>
        <h1 style={{ margin: 0, fontSize: 22, lineHeight: 1.2, fontWeight: 680 }}>{title}</h1>
        {description && <p style={{ margin: '5px 0 0', color: 'var(--text2)', fontSize: 13.5 }}>{description}</p>}
      </div>
      {actions && <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>{actions}</div>}
    </div>
  );
}

export function AdminLoadingState({ title = 'Loading admin data' }: { title?: string }) {
  return (
    <div role="status" aria-label={title} style={{ display: 'grid', gap: 12 }}>
      <SkeletonBar height={42} radius={8} />
      <SkeletonBar height={42} radius={8} />
      <SkeletonBar height={42} radius={8} />
      <SkeletonBar height={42} radius={8} />
    </div>
  );
}

export function AdminShellLoading() {
  return (
    <div
      role="status"
      aria-label="Loading admin console"
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--bg)',
        color: 'var(--text2)',
        gap: 12,
      }}
    >
      <Spinner />
      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12.5 }}>Checking admin session...</span>
    </div>
  );
}

export function AdminForbiddenState() {
  return (
    <section
      role="alert"
      style={{
        maxWidth: 620,
        margin: '64px auto',
        padding: 22,
        border: '1px solid var(--border)',
        borderRadius: 8,
        background: 'var(--surface)',
        boxShadow: 'var(--shadow)',
      }}
    >
      <div style={{ marginBottom: 12 }}>
        <CodePath total={4} done={1} dashed />
      </div>
      <h1 style={{ margin: '0 0 6px', fontSize: 20, fontWeight: 680 }}>403 Forbidden</h1>
      <p style={{ margin: '0 0 18px', color: 'var(--text2)', fontSize: 13.5 }}>
        Your account is signed in, but it does not have enough permission to use the admin console.
      </p>
      <Link href="/" className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
        Back to AstraCode
      </Link>
    </section>
  );
}

export function AdminFeatureUnavailable({
  title = 'Feature not available yet',
  description = 'The backend API for this section has not been implemented.',
  backHref = '/admin',
  backLabel = 'Back to Overview',
}: {
  title?: string;
  description?: string;
  backHref?: string;
  backLabel?: string;
}) {
  return (
    <div
      style={{
        padding: '58px 20px',
        textAlign: 'center',
        border: '1px solid var(--border)',
        borderRadius: 8,
        background: 'var(--surface)',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'center', opacity: 0.72, marginBottom: 14 }}>
        <CodePath total={5} done={2} dashed />
      </div>
      <h2 style={{ margin: '0 0 5px', fontSize: 17, fontWeight: 680 }}>{title}</h2>
      <p style={{ margin: '0 auto 18px', maxWidth: 460, fontSize: 13.5, color: 'var(--text2)' }}>{description}</p>
      <Link href={backHref} className="ac-hover-accent" style={buttonStyles.primary(38)}>
        {backLabel}
      </Link>
    </div>
  );
}

export function AdminApiError({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: string;
  onRetry: () => void;
}) {
  return <ErrorState title={title} detail={error} onRetry={onRetry} />;
}

export function AdminDialog({
  title,
  children,
  onClose,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const focusable = dialogRef.current?.querySelector<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    );
    focusable?.focus();

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
      if (event.key !== 'Tab') return;

      const items = Array.from(
        dialogRef.current?.querySelectorAll<HTMLElement>(
          'button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])',
        ) ?? [],
      );
      if (!items.length) return;
      const first = items[0];
      const last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [onClose]);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="admin-dialog-title"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 120,
        background: 'rgba(17, 13, 35, .42)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 18,
      }}
    >
      <div ref={dialogRef} style={{ ...adminCard, width: 'min(480px, 100%)', padding: 18 }}>
        <h2 id="admin-dialog-title" style={{ margin: '0 0 12px', fontSize: 17 }}>
          {title}
        </h2>
        {children}
      </div>
    </div>
  );
}

export const adminCard: CSSProperties = {
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--surface)',
  boxShadow: 'var(--shadow)',
};

export const adminField: CSSProperties = {
  height: 38,
  border: '1px solid var(--border)',
  borderRadius: 8,
  background: 'var(--surface)',
  color: 'var(--text)',
  padding: '0 10px',
  fontSize: 13,
  boxSizing: 'border-box',
};
