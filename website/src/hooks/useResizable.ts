"use client";

/**
 * useResizable — Resizable split view with persisted layout (plan.md §6).
 * Draggable divider, min/max constraints, localStorage persistence.
 */

import { useState, useCallback, useEffect, useRef } from "react";

interface UseResizableOptions {
  /** localStorage key for persistence */
  storageKey: string;
  /** Default split ratio (0–1, left panel fraction). Default: 0.5 */
  defaultRatio?: number;
  /** Minimum ratio for left panel */
  minRatio?: number;
  /** Maximum ratio for left panel */
  maxRatio?: number;
  /** Orientation: horizontal or vertical split */
  direction?: "horizontal" | "vertical";
}

export function useResizable(options: UseResizableOptions) {
  const {
    storageKey,
    defaultRatio = 0.5,
    minRatio = 0.25,
    maxRatio = 0.75,
    direction = "horizontal",
  } = options;

  const [ratio, setRatio] = useState<number>(() => {
    if (typeof window === "undefined") return defaultRatio;
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = parseFloat(stored);
        if (!isNaN(parsed) && parsed >= minRatio && parsed <= maxRatio) {
          return parsed;
        }
      }
    } catch {
      // Ignore
    }
    return defaultRatio;
  });

  const isDragging = useRef(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Persist to localStorage on change
  useEffect(() => {
    try {
      localStorage.setItem(storageKey, String(ratio));
    } catch {
      // Ignore
    }
  }, [storageKey, ratio]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      isDragging.current = true;
      document.body.style.cursor =
        direction === "horizontal" ? "col-resize" : "row-resize";
      document.body.style.userSelect = "none";

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!isDragging.current || !containerRef.current) return;
        const rect = containerRef.current.getBoundingClientRect();

        let newRatio: number;
        if (direction === "horizontal") {
          newRatio = (moveEvent.clientX - rect.left) / rect.width;
        } else {
          newRatio = (moveEvent.clientY - rect.top) / rect.height;
        }

        newRatio = Math.min(maxRatio, Math.max(minRatio, newRatio));
        setRatio(newRatio);
      };

      const handleMouseUp = () => {
        isDragging.current = false;
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [direction, minRatio, maxRatio]
  );

  return {
    ratio,
    setRatio,
    containerRef,
    handleMouseDown,
    /** CSS for left/top panel */
    leftStyle: {
      [direction === "horizontal" ? "width" : "height"]: `${ratio * 100}%`,
    } as React.CSSProperties,
    /** CSS for right/bottom panel */
    rightStyle: {
      [direction === "horizontal" ? "width" : "height"]: `${(1 - ratio) * 100}%`,
    } as React.CSSProperties,
  };
}
