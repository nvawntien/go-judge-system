import { CheckCircle2, Circle, Clock } from "lucide-react"

export function ProblemStatusLegend() {
  return (
    <div className="mt-4 pt-4 border-t grid grid-cols-1 md:grid-cols-3 gap-4">
      <div className="flex items-start gap-3">
        <CheckCircle2 className="h-5 w-5 text-emerald-500 mt-0.5" />
        <div>
          <div className="font-semibold text-slate-900 text-sm">Solved</div>
          <div className="text-slate-500 text-xs">You have solved this problem</div>
        </div>
      </div>
      <div className="flex items-start gap-3">
        <Clock className="h-5 w-5 text-amber-500 mt-0.5" />
        <div>
          <div className="font-semibold text-slate-900 text-sm">Attempted</div>
          <div className="text-slate-500 text-xs">You have attempted this problem</div>
        </div>
      </div>
      <div className="flex items-start gap-3">
        <Circle className="h-5 w-5 text-slate-300 mt-0.5" />
        <div>
          <div className="font-semibold text-slate-900 text-sm">Unsolved</div>
          <div className="text-slate-500 text-xs">You haven&apos;t attempted yet</div>
        </div>
      </div>
    </div>
  )
}
