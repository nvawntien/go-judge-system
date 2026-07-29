import { Suspense } from 'react';
import { VerifyEmailView } from '@/components/auth/VerifyEmailView';
import { Spinner } from '@/components/ui';

function VerifyEmailFallback() {
  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        color: 'var(--text2)',
      }}
    >
      <Spinner size={18} />
    </main>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<VerifyEmailFallback />}>
      <VerifyEmailView />
    </Suspense>
  );
}
