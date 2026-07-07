import { ActivityCell } from "@/lib/mocks/profile";
import { Info } from "lucide-react";

interface RecentActivityGridProps {
  activity: ActivityCell[];
}

export function RecentActivityGrid({ activity }: RecentActivityGridProps) {
  // Sort activity by date (oldest to newest) just in case
  const sorted = [...activity].sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());
  
  // Calculate total submissions
  const totalSubmissions = 1284; // Mocked per design

  const getCellColor = (level: number) => {
    switch (level) {
      case 1: return "bg-violet-200 dark:bg-violet-900/50";
      case 2: return "bg-violet-400 dark:bg-violet-700/60";
      case 3: return "bg-violet-500 dark:bg-violet-500/80";
      case 4: return "bg-violet-600 dark:bg-violet-400";
      default: return "bg-slate-100 dark:bg-slate-800";
    }
  };

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm p-5 transition-colors overflow-hidden flex flex-col h-full">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-2">
        <h2 className="font-semibold text-slate-900 dark:text-slate-100 flex items-center gap-2 transition-colors">
          Recent Activity
          <Info className="h-4 w-4 text-slate-400 cursor-help" />
        </h2>
        <span className="text-sm font-medium text-slate-500 dark:text-slate-400 transition-colors">
          Contribution in the last 12 months
        </span>
      </div>

      {/* Grid Container */}
      <div className="w-full overflow-x-auto pb-4 hide-scrollbar">
        <div className="min-w-[700px]">
          
          {/* Months Row */}
          <div className="flex justify-between text-[11px] text-slate-500 dark:text-slate-400 mb-2 pl-[32px] pr-2">
            <span>Jun</span>
            <span>Jul</span>
            <span>Aug</span>
            <span>Sep</span>
            <span>Oct</span>
            <span>Nov</span>
            <span>Dec</span>
            <span>Jan</span>
            <span>Feb</span>
            <span>Mar</span>
            <span>Apr</span>
            <span>May</span>
          </div>
          
          {/* Grid Area with Weekdays */}
          <div className="flex gap-2">
            {/* Weekdays */}
            <div className="grid grid-rows-7 gap-1 text-[11px] text-slate-400 dark:text-slate-500 pr-1 items-center text-right w-[24px] shrink-0">
              <div className="h-2.5"></div>
              <div className="h-2.5 leading-none">Mon</div>
              <div className="h-2.5"></div>
              <div className="h-2.5 leading-none">Wed</div>
              <div className="h-2.5"></div>
              <div className="h-2.5 leading-none">Fri</div>
              <div className="h-2.5"></div>
            </div>
            
            {/* Cells */}
            <div className="grid grid-rows-7 grid-flow-col gap-1 flex-1">
              {sorted.map((cell, idx) => (
                <div
                  key={idx}
                  className={`w-2.5 h-2.5 rounded-[2px] transition-colors ${getCellColor(cell.level)}`}
                  title={`${cell.date}: ${cell.count} submissions`}
                />
              ))}
            </div>
          </div>

        </div>
      </div>

      {/* Footer */}
      <div className="mt-auto pt-4 flex flex-col sm:flex-row items-start sm:items-center justify-between text-[13px] text-slate-500 dark:text-slate-400 transition-colors gap-3">
        <div className="font-medium text-slate-700 dark:text-slate-300">
          Total Submissions: {totalSubmissions.toLocaleString()}
        </div>
        
        <div className="flex items-center gap-1.5 shrink-0">
          <span className="mr-1">Less</span>
          <div className={`w-3 h-3 rounded-[2px] ${getCellColor(0)}`} />
          <div className={`w-3 h-3 rounded-[2px] ${getCellColor(1)}`} />
          <div className={`w-3 h-3 rounded-[2px] ${getCellColor(2)}`} />
          <div className={`w-3 h-3 rounded-[2px] ${getCellColor(3)}`} />
          <div className={`w-3 h-3 rounded-[2px] ${getCellColor(4)}`} />
          <span className="ml-1">More</span>
        </div>
      </div>
      
    </div>
  );
}

