import { ProblemDifficulty } from "@/lib/mocks/problems"

interface ProblemDifficultyBadgeProps {
  difficulty: ProblemDifficulty;
}

export function ProblemDifficultyBadge({ difficulty }: ProblemDifficultyBadgeProps) {
  const baseClasses = "inline-flex items-center justify-center w-[68px] py-0.5 rounded text-[11px] font-medium transition-colors";
  switch (difficulty) {
    case "Easy":
      return <span className={`${baseClasses} bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400`}>Easy</span>;
    case "Medium":
      return <span className={`${baseClasses} bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400`}>Medium</span>;
    case "Hard":
      return <span className={`${baseClasses} bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400`}>Hard</span>;
    default:
      return <span className={`${baseClasses} bg-slate-50 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400`}>{difficulty}</span>;
  }
}
