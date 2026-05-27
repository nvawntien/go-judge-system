"use client";

/**
 * Toast system — global notification (plan.md §10).
 * Z-index: var(--z-toast) = 600.
 */

import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import { X, CheckCircle, AlertCircle, Info } from "lucide-react";

type ToastType = "success" | "error" | "info";

interface Toast {
  id: string;
  type: ToastType;
  message: string;
  exiting?: boolean;
}

interface ToastContextValue {
  addToast: (type: ToastType, message: string) => void;
}

const ToastContext = createContext<ToastContextValue>({
  addToast: () => {},
});

export function useToast() {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) =>
      prev.map((t) => (t.id === id ? { ...t, exiting: true } : t)),
    );
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 200);
  }, []);

  const addToast = useCallback(
    (type: ToastType, message: string) => {
      const id = crypto.randomUUID();
      setToasts((prev) => [...prev.slice(-4), { id, type, message }]);
      setTimeout(() => removeToast(id), 5000);
    },
    [removeToast],
  );

  const icons: Record<ToastType, ReactNode> = {
    success: <CheckCircle size={18} />,
    error: <AlertCircle size={18} />,
    info: <Info size={18} />,
  };

  const colors: Record<ToastType, string> = {
    success:
      "border-[var(--oj-ac-txt)] bg-[var(--oj-ac-bg)] text-[var(--oj-ac-txt)]",
    error:
      "border-[var(--oj-wa-txt)] bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)]",
    info: "border-[var(--oj-accent)] bg-[var(--oj-accent-fill)] text-[var(--oj-accent)]",
  };

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <div
        className="fixed top-4 right-4 flex flex-col gap-2 pointer-events-none"
        style={{ zIndex: "var(--z-toast)" } as React.CSSProperties}
        aria-live="polite"
      >
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`
              pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-lg border
              shadow-lg max-w-sm ${colors[t.type]}
              ${t.exiting ? "oj-toast-exit" : "oj-toast-enter"}
            `}
            role="alert"
          >
            {icons[t.type]}
            <span className="text-sm font-medium flex-1">{t.message}</span>
            <button
              onClick={() => removeToast(t.id)}
              className="cursor-pointer opacity-70 hover:opacity-100 transition-opacity"
              aria-label="Dismiss notification"
            >
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
