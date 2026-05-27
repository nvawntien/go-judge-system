"use client";

/**
 * NetworkBanner — Shows banner when offline/degraded (plan.md §24).
 */

import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { WifiOff, AlertTriangle } from "lucide-react";

export function NetworkBanner() {
  const { status } = useNetworkStatus();

  if (status === "ONLINE") return null;

  return (
    <div
      className={`
        sticky top-16 z-[var(--z-sticky)] flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium
        ${status === "OFFLINE"
          ? "bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)] border-b border-[var(--oj-wa-txt)]/20"
          : "bg-[var(--oj-tle-bg)] text-[var(--oj-tle-txt)] border-b border-[var(--oj-tle-txt)]/20"
        }
      `}
      role="alert"
    >
      {status === "OFFLINE" ? (
        <>
          <WifiOff size={16} />
          You are offline — submissions will be queued and sent when connection restores.
        </>
      ) : (
        <>
          <AlertTriangle size={16} />
          Connection degraded — real-time updates may be delayed.
        </>
      )}
    </div>
  );
}
