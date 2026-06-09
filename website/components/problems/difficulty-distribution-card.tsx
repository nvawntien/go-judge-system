import { Card, CardContent } from "@/components/ui/card"

export function DifficultyDistributionCard() {
  return (
    <Card className="rounded-xl border-slate-200 dark:border-slate-800/80 bg-white dark:bg-slate-900/50 shadow-sm mb-6 transition-colors">
      <CardContent className="p-6">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-6 transition-colors">Difficulty Distribution</h3>
        
        <div className="flex items-center gap-6">
          <div className="w-20 h-20 rounded-full shrink-0" style={{ background: "conic-gradient(#10b981 0% 43.3%, #f59e0b 43.3% 80.1%, #ef4444 80.1% 100%)" }}>
            <div className="w-12 h-12 bg-white dark:bg-slate-900 rounded-full mx-auto mt-4 transition-colors"></div>
          </div>
          
          <div className="flex-1 space-y-3 text-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-emerald-500 dark:bg-emerald-400 transition-colors"></div>
                <span className="text-slate-700 dark:text-slate-300 transition-colors">Easy</span>
              </div>
              <span className="text-slate-500 dark:text-slate-400 transition-colors">148 (43.3%)</span>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-amber-500 dark:bg-amber-400 transition-colors"></div>
                <span className="text-slate-700 dark:text-slate-300 transition-colors">Medium</span>
              </div>
              <span className="text-slate-500 dark:text-slate-400 transition-colors">126 (36.8%)</span>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-red-500 dark:bg-red-400 transition-colors"></div>
                <span className="text-slate-700 dark:text-slate-300 transition-colors">Hard</span>
              </div>
              <span className="text-slate-500 dark:text-slate-400 transition-colors">68 (19.9%)</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
