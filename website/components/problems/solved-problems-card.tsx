import { Card, CardContent } from "@/components/ui/card"

export function SolvedProblemsCard() {
  return (
    <Card className="rounded-xl border-slate-200 dark:border-slate-800/80 bg-white dark:bg-slate-900/50 shadow-sm mb-6 transition-colors">
      <CardContent className="p-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-semibold text-slate-900 dark:text-slate-100 transition-colors">Solved Problems</h3>
          <a href="#" className="text-xs font-medium text-purple-600 dark:text-violet-400 hover:underline transition-colors">View profile</a>
        </div>
        
        <div className="flex items-center gap-6">
          <div className="relative w-24 h-24 rounded-full flex items-center justify-center shrink-0" style={{ background: "conic-gradient(#8b5cf6 0% 36%, #f1f5f9 36% 100%)" }}>
            <div className="absolute inset-2 bg-white dark:bg-slate-900 rounded-full flex flex-col items-center justify-center transition-colors">
              <span className="text-2xl font-bold text-slate-900 dark:text-slate-100 transition-colors">124</span>
              <span className="text-xs text-slate-500 dark:text-slate-400 transition-colors">Solved</span>
            </div>
          </div>
          
          <div className="flex-1 space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400 transition-colors">Total Problems</span>
              <span className="font-semibold text-slate-900 dark:text-slate-100 transition-colors">342</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400 transition-colors">Attempted</span>
              <span className="font-semibold text-slate-900 dark:text-slate-100 transition-colors">184</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400 transition-colors">Acceptance Rate</span>
              <span className="font-semibold text-slate-900 dark:text-slate-100 transition-colors">67.4%</span>
            </div>
          </div>
        </div>
        
        <div className="mt-6">
          <div className="h-2 w-full bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden mb-2 transition-colors">
            <div className="h-full bg-purple-600 dark:bg-violet-500 rounded-full transition-colors" style={{ width: "36%" }}></div>
          </div>
          <div className="text-right text-xs text-slate-500 dark:text-slate-400 transition-colors">124 / 342</div>
        </div>
      </CardContent>
    </Card>
  )
}
