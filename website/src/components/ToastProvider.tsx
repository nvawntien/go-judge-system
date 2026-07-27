'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

export type ToastKind = 'success' | 'error' | 'info';

interface Toast {
  id: number;
  message: string;
  kind: ToastKind;
}

interface ToastContextValue {
  showToast: (message: string, kind?: ToastKind) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

const DOT: Record<ToastKind, string> = {
  success: 'var(--success)',
  error: 'var(--error)',
  info: 'var(--accent)',
};

const BORDER: Record<ToastKind, string> = {
  success: 'var(--success)',
  error: 'var(--error)',
  info: 'var(--border2)',
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toast, setToast] = useState<Toast | null>(null);

  const showToast = useCallback((message: string, kind: ToastKind = 'info') => {
    setToast({ id: Date.now() + Math.random(), message, kind });
  }, []);

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast((current) => (current?.id === toast.id ? null : current)), 3200);
    return () => clearTimeout(id);
  }, [toast]);

  const value = useMemo(() => ({ showToast }), [showToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast && (
        <div
          role="status"
          key={toast.id}
          style={{
            position: 'fixed',
            bottom: 22,
            right: 22,
            zIndex: 80,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            background: 'var(--surface)',
            border: `1px solid ${BORDER[toast.kind]}`,
            borderRadius: 11,
            padding: '12px 16px',
            boxShadow: 'var(--shadow-lg)',
            animation: 'acToast .22s ease',
            maxWidth: 340,
          }}
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: DOT[toast.kind],
              flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 13, fontWeight: 550 }}>{toast.message}</span>
        </div>
      )}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used inside <ToastProvider>');
  return ctx;
}
