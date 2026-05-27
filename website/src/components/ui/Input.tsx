"use client";

/**
 * Input — Form input with label, error state, and focus ring (plan.md §4).
 */

import { forwardRef, type InputHTMLAttributes, useId } from "react";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, className = "", id: externalId, ...props }, ref) => {
    const internalId = useId();
    const id = externalId || internalId;

    return (
      <div className={`flex flex-col gap-1.5 ${className}`}>
        {label && (
          <label htmlFor={id} className="text-sm font-medium text-[var(--oj-body)]">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={id}
          className={`
            w-full rounded-md border bg-[var(--oj-surface)] px-3 py-2 text-sm
            text-[var(--oj-text)] placeholder:text-[var(--oj-muted)]
            transition-colors
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--oj-accent)] focus-visible:ring-offset-1 focus-visible:border-[var(--oj-accent)]
            disabled:cursor-not-allowed disabled:opacity-50
            ${error ? "border-[var(--oj-wa-txt)] focus-visible:ring-[var(--oj-wa-txt)] focus-visible:border-[var(--oj-wa-txt)]" : "border-[var(--oj-border)]"}
          `}
          {...props}
          aria-invalid={!!error}
          aria-describedby={error ? `${id}-error` : undefined}
        />
        {error && (
          <span id={`${id}-error`} className="text-xs text-[var(--oj-wa-txt)]" role="alert">
            {error}
          </span>
        )}
      </div>
    );
  }
);

Input.displayName = "Input";
export { Input };
