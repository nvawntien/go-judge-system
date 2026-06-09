"use client";

import { MOCK_PROBLEMS } from "@/lib/mocks/problems"
import { ProblemDifficultyBadge } from "./problem-difficulty-badge"
import { ProblemStatusBadge } from "./problem-status-badge"
import { ProblemTagList } from "./problem-tag-list"
import { ChevronLeft, ChevronRight } from "lucide-react"
export function ProblemTable() {
  return (
    <div className="w-full">
      <table className="w-full text-sm text-left table-fixed">
        <colgroup>
          <col className="w-14" />
          <col className="w-auto" />
          <col className="w-24" />
          <col className="w-[32%]" />
          <col className="w-24" />
        </colgroup>
        <thead className="bg-white dark:bg-slate-900/50 text-slate-500 dark:text-slate-400 border-b dark:border-slate-800 transition-colors">
          <tr>
            <th className="px-4 py-3 font-medium text-center">Status</th>
            <th className="px-4 py-3 font-medium">Problem</th>
            <th className="px-4 py-3 font-medium text-center">Difficulty</th>
            <th className="px-4 py-3 font-medium hidden lg:table-cell">Tags</th>
            <th className="px-4 py-3 font-medium hidden sm:table-cell">Acceptance</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-slate-800/80 transition-colors">
          {MOCK_PROBLEMS.map((problem) => (
            <tr key={problem.id} className="transition-colors hover:bg-violet-50/40 dark:hover:bg-slate-800/50 group bg-white dark:bg-transparent">
              <td className="px-4 py-3 align-middle text-center">
                <ProblemStatusBadge status={problem.status} />
              </td>
              <td className="px-4 py-3 align-middle truncate">
                <span 
                  className="font-medium text-slate-900 dark:text-slate-200 group-hover:text-violet-700 dark:group-hover:text-violet-400 transition-colors cursor-pointer truncate block"
                  title={problem.title}
                >
                  {problem.id}. {problem.title}
                </span>
              </td>
              <td className="px-4 py-3 align-middle text-center">
                <ProblemDifficultyBadge difficulty={problem.difficulty} />
              </td>
              <td className="px-4 py-3 align-middle hidden lg:table-cell">
                <ProblemTagList tags={problem.tags} />
              </td>
              <td className="px-4 py-3 text-slate-600 dark:text-slate-400 align-middle hidden sm:table-cell">
                <div className="flex flex-col gap-1.5 justify-center">
                  <span>{problem.acceptanceRate.toFixed(1)}%</span>
                  <div className="w-14 h-1 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-emerald-400 dark:bg-emerald-500 rounded-full" 
                      style={{ width: `${problem.acceptanceRate}%` }} 
                    />
                  </div>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      
      {/* Pagination */}
      <div className="flex items-center justify-center px-4 py-4 border-t dark:border-slate-800 transition-colors">
        <div className="flex items-center space-x-1.5">
          <button className="h-8 w-8 inline-flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-500 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <button className="h-8 w-8 inline-flex items-center justify-center rounded-lg border border-violet-600 dark:border-violet-500 bg-violet-600 dark:bg-violet-500 text-white dark:text-slate-950 font-medium shadow-sm transition-colors text-sm">
            1
          </button>
          <button className="h-8 w-8 inline-flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm text-sm">
            2
          </button>
          <button className="h-8 w-8 hidden sm:inline-flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm text-sm">
            3
          </button>
          <span className="px-1 text-slate-400 dark:text-slate-500 font-medium">...</span>
          <button className="h-8 w-8 inline-flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm text-sm">
            35
          </button>
          <button className="h-8 w-8 inline-flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-500 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
