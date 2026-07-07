import { SearchX } from "lucide-react"

interface ProblemEmptyStateProps {
  title?: string;
  description?: string;
}

export function ProblemEmptyState({ 
  title = "No problems found", 
  description = "Try adjusting your search or filters to find what you're looking for." 
}: ProblemEmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      <div className="h-12 w-12 rounded-full bg-slate-50 flex items-center justify-center mb-4 border border-slate-100">
        <SearchX className="h-6 w-6 text-slate-400" />
      </div>
      <h3 className="text-sm font-semibold text-slate-900 mb-1">{title}</h3>
      <p className="text-sm text-slate-500 max-w-sm mx-auto">
        {description}
      </p>
      <button className="mt-4 text-sm font-medium text-violet-600 hover:text-violet-700 transition-colors">
        Clear all filters
      </button>
    </div>
  )
}
