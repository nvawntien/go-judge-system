import { ProblemFilters } from "@/components/problems/problem-filters"
import { ProblemTable } from "@/components/problems/problem-table"
import { ProblemPageSidebar } from "@/components/problems/problem-page-sidebar"

export default function ProblemsPage() {
  return (
    <div className="max-w-7xl mx-auto pb-12">
      {/* 2-Column Layout */}
      <div className="grid grid-cols-1 xl:grid-cols-[1fr_320px] gap-6 items-start">
        {/* Main Content Card */}
        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm p-5 sm:p-6 sm:pt-5 overflow-hidden transition-colors">
          <ProblemFilters />
          <ProblemTable />
        </div>
        
        {/* Right Sidebar */}
        <div className="sticky top-20">
          <ProblemPageSidebar />
        </div>
      </div>
    </div>
  )
}
