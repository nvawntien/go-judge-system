/**
 * Skeleton — matches final component sizes (zero layout shift).
 * plan.md §6: "Skeleton match final size".
 */

interface SkeletonProps {
  className?: string;
  width?: string;
  height?: string;
}

export function Skeleton({ className = "", width, height }: SkeletonProps) {
  return (
    <div
      className={`oj-skeleton ${className}`}
      style={{ width, height }}
      aria-hidden="true"
    />
  );
}

/** Table row skeleton — matches the problem list row layout */
export function TableRowSkeleton({ cols = 4 }: { cols?: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <Skeleton className="h-4 w-full" />
        </td>
      ))}
    </tr>
  );
}

/** Card skeleton */
export function CardSkeleton() {
  return (
    <div className="rounded-lg border border-[var(--oj-border)] p-6 space-y-4">
      <Skeleton className="h-6 w-3/4" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-5/6" />
      <div className="flex gap-2">
        <Skeleton className="h-6 w-16" />
        <Skeleton className="h-6 w-20" />
      </div>
    </div>
  );
}
