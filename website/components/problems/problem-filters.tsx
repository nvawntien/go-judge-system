"use client";

import { Input } from "@/components/ui/input"
import { Search, RotateCcw, ChevronDown } from "lucide-react"

export function ProblemFilters() {
  return (
    <div className="flex flex-wrap lg:grid lg:grid-cols-[minmax(220px,1fr)_110px_96px_84px_140px_40px] items-center gap-3 lg:gap-4 mb-6 w-full">
      <div className="relative w-full lg:w-auto shrink-0 flex-1 min-w-[200px]">
        <Search className="absolute left-3 top-3 h-4 w-4 text-slate-400 dark:text-slate-500" />
        <Input 
          type="search" 
          placeholder="Search problems by title or tag..." 
          className="pl-9 h-10 w-full rounded-lg border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-900 dark:text-slate-100 text-sm shadow-sm focus-visible:ring-purple-600 dark:focus-visible:ring-violet-500 placeholder:text-slate-400 dark:placeholder:text-slate-500 transition-colors"
        />
      </div>
      
      <button className="h-10 inline-flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-3 sm:px-4 py-2 text-[13px] font-medium text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 whitespace-nowrap transition-colors">
        Difficulty <ChevronDown className="ml-1 sm:ml-2 h-4 w-4 text-slate-400 dark:text-slate-500 shrink-0" />
      </button>
      
      <button className="h-10 inline-flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-3 sm:px-4 py-2 text-[13px] font-medium text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 whitespace-nowrap transition-colors">
        Status <ChevronDown className="ml-1 sm:ml-2 h-4 w-4 text-slate-400 dark:text-slate-500 shrink-0" />
      </button>
      
      <button className="h-10 inline-flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-3 sm:px-4 py-2 text-[13px] font-medium text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 whitespace-nowrap transition-colors">
        Tags <ChevronDown className="ml-1 sm:ml-2 h-4 w-4 text-slate-400 dark:text-slate-500 shrink-0" />
      </button>

      <button className="h-10 inline-flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-3 sm:px-4 py-2 text-[13px] text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 whitespace-nowrap transition-colors">
        <span className="flex items-center"><span className="text-slate-500 dark:text-slate-400 mr-1">Sort by</span><span className="font-medium">Default</span></span> <ChevronDown className="ml-1 sm:ml-2 h-4 w-4 text-slate-400 dark:text-slate-500 shrink-0" />
      </button>

      <button 
        aria-label="Reset filters"
        title="Reset filters"
        className="h-10 w-10 flex items-center justify-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 text-slate-500 dark:text-slate-400 shadow-sm hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-200 focus:outline-none focus:ring-1 focus:ring-purple-600 dark:focus:ring-violet-500 transition-colors shrink-0"
      >
        <RotateCcw className="h-4 w-4 shrink-0" />
      </button>
    </div>
  )
}
