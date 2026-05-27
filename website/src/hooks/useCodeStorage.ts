"use client";

/**
 * useCodeStorage — Persist code per-problem per-language (plan.md §6).
 * Uses localStorage to preserve user's code when navigating away and back.
 */

import { useState, useEffect, useCallback } from "react";
import type { Language } from "@/types/api";

function storageKey(problemSlug: string, language: Language): string {
  return `oj-code-${problemSlug}-${language}`;
}

export function useCodeStorage(
  problemSlug: string,
  language: Language,
  defaultCode: string
) {
  const [code, setCodeInternal] = useState<string>(() => {
    if (typeof window === "undefined") return defaultCode;
    try {
      const stored = localStorage.getItem(storageKey(problemSlug, language));
      return stored ?? defaultCode;
    } catch {
      return defaultCode;
    }
  });

  // Re-load when problem or language changes
  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      const stored = localStorage.getItem(storageKey(problemSlug, language));
      setCodeInternal(stored ?? defaultCode);
    } catch {
      setCodeInternal(defaultCode);
    }
  }, [problemSlug, language, defaultCode]);

  // Persist on change (debounced via effect)
  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      localStorage.setItem(storageKey(problemSlug, language), code);
    } catch {
      // Ignore — localStorage may be full
    }
  }, [code, problemSlug, language]);

  const setCode = useCallback((value: string) => {
    setCodeInternal(value);
  }, []);

  const resetCode = useCallback(() => {
    setCodeInternal(defaultCode);
    try {
      localStorage.removeItem(storageKey(problemSlug, language));
    } catch {
      // Ignore
    }
  }, [defaultCode, problemSlug, language]);

  return { code, setCode, resetCode };
}
