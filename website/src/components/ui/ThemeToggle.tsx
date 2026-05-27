"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "@/lib/theme-context";

export function ThemeToggle() {
  const { theme, toggle } = useTheme();

  return (
    <button
      onClick={toggle}
      className="cursor-pointer p-2 rounded-md text-[var(--oj-muted)] hover:text-[var(--oj-text)] hover:bg-[var(--oj-surface)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--oj-accent)]"
      aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
    >
      {theme === "dark" ? <Sun size={20} /> : <Moon size={20} />}
    </button>
  );
}
