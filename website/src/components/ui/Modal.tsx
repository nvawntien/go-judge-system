"use client";

/**
 * Modal — native <dialog> with ARIA, Esc to close (plan.md §4).
 * Z-index: var(--z-modal) = 500.
 */

import { useEffect, useRef, type ReactNode } from "react";
import { X } from "lucide-react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function Modal({ isOpen, onClose, title, children }: ModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen && !dialog.open) {
      dialog.showModal();
      document.body.style.overflow = "hidden";
    } else if (!isOpen && dialog.open) {
      dialog.close();
      document.body.style.overflow = "";
    }

    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    const handleCancel = (e: Event) => {
      e.preventDefault();
      onClose();
    };

    dialog.addEventListener("cancel", handleCancel);
    return () => dialog.removeEventListener("cancel", handleCancel);
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <dialog
      ref={dialogRef}
      className="fixed inset-0 m-auto p-0 rounded-lg bg-[var(--oj-bg)] text-[var(--oj-text)] shadow-xl border border-[var(--oj-border)] backdrop:bg-[var(--oj-overlay)] backdrop:backdrop-blur-sm"
      style={{ zIndex: "var(--z-modal)" } as React.CSSProperties}
      onClick={(e) => {
        if (e.target === dialogRef.current) onClose();
      }}
    >
      <div
        className="flex flex-col w-full max-w-md"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-4 border-b border-[var(--oj-border)]">
          <h2 className="text-lg font-semibold">{title}</h2>
          <button
            onClick={onClose}
            className="cursor-pointer p-1 rounded-md hover:bg-[var(--oj-surface)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--oj-accent)]"
            aria-label="Close modal"
          >
            <X size={20} className="text-[var(--oj-muted)]" />
          </button>
        </div>
        <div className="p-4">{children}</div>
      </div>
    </dialog>
  );
}
