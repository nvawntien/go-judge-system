/**
 * Badge — verdict/status/difficulty badges using CSS variable colors.
 */

import type { SubmissionStatus, Difficulty } from "@/types/api";

interface BadgeProps {
  children: React.ReactNode;
  className?: string;
}

const statusStyles: Record<SubmissionStatus, string> = {
  ACCEPTED: "bg-[var(--oj-ac-bg)] text-[var(--oj-ac-txt)]",
  WRONG_ANSWER: "bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)]",
  TIME_LIMIT_EXCEEDED: "bg-[var(--oj-tle-bg)] text-[var(--oj-tle-txt)]",
  MEMORY_LIMIT_EXCEEDED: "bg-[var(--oj-tle-bg)] text-[var(--oj-tle-txt)]",
  RUNTIME_ERROR: "bg-[var(--oj-re-bg)] text-[var(--oj-re-txt)]",
  COMPILATION_ERROR: "bg-[var(--oj-ce-bg)] text-[var(--oj-ce-txt)]",
  SYSTEM_ERROR: "bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)]",
  PENDING: "bg-[var(--oj-pd-bg)] text-[var(--oj-pd-txt)]",
  JUDGING: "bg-[var(--oj-pd-bg)] text-[var(--oj-pd-txt)]",
};

const statusLabels: Record<SubmissionStatus, string> = {
  ACCEPTED: "Accepted",
  WRONG_ANSWER: "Wrong Answer",
  TIME_LIMIT_EXCEEDED: "TLE",
  MEMORY_LIMIT_EXCEEDED: "MLE",
  RUNTIME_ERROR: "Runtime Error",
  COMPILATION_ERROR: "Compile Error",
  SYSTEM_ERROR: "System Error",
  PENDING: "Pending",
  JUDGING: "Judging",
};

const difficultyStyles: Record<Difficulty, string> = {
  EASY: "bg-[var(--oj-ac-bg)] text-[var(--oj-ac-txt)]",
  MEDIUM: "bg-[var(--oj-tle-bg)] text-[var(--oj-tle-txt)]",
  HARD: "bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)]",
};

export function Badge({ children, className = "" }: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-semibold
        ${className}
      `}
    >
      {children}
    </span>
  );
}

export function StatusBadge({
  status,
  className = "",
}: {
  status: SubmissionStatus;
  className?: string;
}) {
  return (
    <Badge className={`${statusStyles[status]} ${className}`}>
      {statusLabels[status]}
    </Badge>
  );
}

export function DifficultyBadge({
  difficulty,
  className = "",
}: {
  difficulty: Difficulty;
  className?: string;
}) {
  return (
    <Badge className={`${difficultyStyles[difficulty]} ${className}`}>
      {difficulty.charAt(0) + difficulty.slice(1).toLowerCase()}
    </Badge>
  );
}
