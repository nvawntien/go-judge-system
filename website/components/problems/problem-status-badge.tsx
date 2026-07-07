import { CheckCircle2, Circle, Clock } from "lucide-react"
import { ProblemStatus } from "@/lib/mocks/problems"

interface ProblemStatusBadgeProps {
  status: ProblemStatus;
}

export function ProblemStatusBadge({ status }: ProblemStatusBadgeProps) {
  switch (status) {
    case "Solved":
      return <div className="flex justify-center" aria-label="Solved" title="Solved"><CheckCircle2 className="h-4 w-4 text-emerald-500 dark:text-emerald-400" /></div>;
    case "Attempted":
      return <div className="flex justify-center" aria-label="Attempted" title="Attempted"><Clock className="h-4 w-4 text-amber-500 dark:text-amber-400" /></div>;
    case "Unsolved":
    default:
      return <div className="flex justify-center" aria-label="Unsolved" title="Unsolved"><Circle className="h-4 w-4 text-slate-300 dark:text-slate-700" /></div>;
  }
}
