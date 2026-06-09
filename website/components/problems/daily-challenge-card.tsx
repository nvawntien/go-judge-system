import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Calendar } from "lucide-react"

export function DailyChallengeCard() {
  return (
    <Card className="rounded-xl border-slate-200 dark:border-slate-800/80 bg-white dark:bg-slate-900/50 shadow-sm mb-6 relative overflow-hidden transition-colors">
      <div className="absolute top-4 right-4 bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400 text-[10px] font-bold px-2 py-0.5 rounded uppercase transition-colors">New</div>
      <CardContent className="p-6">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4 transition-colors">Daily Challenge</h3>
        
        <div className="flex items-start gap-4 mb-4">
          <div className="w-10 h-10 rounded-lg bg-purple-50 dark:bg-violet-500/10 text-purple-600 dark:text-violet-400 flex items-center justify-center shrink-0 transition-colors">
            <Calendar className="h-5 w-5" />
          </div>
          <div>
            <div className="font-medium text-slate-900 dark:text-slate-100 text-sm leading-tight mb-1 transition-colors">Sum of Subarray Minimums</div>
            <div className="flex items-center gap-2 text-xs">
              <div className="flex items-center gap-1 text-amber-600 dark:text-amber-400 transition-colors">
                <div className="w-1.5 h-1.5 rounded-full bg-amber-500 dark:bg-amber-400 transition-colors"></div> Medium
              </div>
              <span className="text-purple-600 dark:text-violet-400 font-medium transition-colors">+150 XP</span>
            </div>
          </div>
        </div>
        
        <p className="text-sm text-slate-500 dark:text-slate-400 mb-4 leading-relaxed transition-colors">
          Solve today&apos;s challenge and keep your streak alive!
        </p>
        
        <Button className="w-full bg-violet-600 dark:bg-violet-500 hover:bg-violet-700 dark:hover:bg-violet-600 text-white dark:text-slate-950 rounded-lg shadow-sm font-semibold transition-all">
          Solve Challenge
        </Button>
      </CardContent>
    </Card>
  )
}
