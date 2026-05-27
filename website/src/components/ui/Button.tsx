"use client";

/**
 * Button — plan.md §4: 5 states (Hover, Focus, Active, Disabled, Loading).
 * Prevents double submit. Shows spinner when loading.
 */

import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { Loader2 } from "lucide-react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  icon?: ReactNode;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "bg-[var(--oj-accent)] text-white hover:bg-[var(--oj-accent-dk)] focus-visible:ring-[var(--oj-accent)]",
  secondary:
    "bg-[var(--oj-surface)] text-[var(--oj-text)] border border-[var(--oj-border)] hover:bg-[var(--oj-panel)] focus-visible:ring-[var(--oj-accent)]",
  ghost:
    "bg-transparent text-[var(--oj-body)] hover:bg-[var(--oj-surface)] focus-visible:ring-[var(--oj-accent)]",
  danger:
    "bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)] hover:opacity-90 focus-visible:ring-[var(--oj-wa-txt)]",
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-xs gap-1.5",
  md: "px-4 py-2 text-sm gap-2",
  lg: "px-6 py-2.5 text-base gap-2.5",
};

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "primary",
      size = "md",
      loading = false,
      icon,
      children,
      disabled,
      className = "",
      ...props
    },
    ref,
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={`
          inline-flex items-center justify-center font-medium rounded-lg
          cursor-pointer select-none
          transition-all duration-200 ease-out
          active:scale-[0.98]
          disabled:opacity-50 disabled:cursor-not-allowed disabled:active:scale-100
          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
          ${variantStyles[variant]}
          ${sizeStyles[size]}
          ${className}
        `}
        {...props}
      >
        {loading ? (
          <Loader2 size={size === "sm" ? 14 : 16} className="animate-spin" />
        ) : icon ? (
          icon
        ) : null}
        {children}
      </button>
    );
  },
);

Button.displayName = "Button";
export { Button };
export type { ButtonProps };
