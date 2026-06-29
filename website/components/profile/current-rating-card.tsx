import { RatingSnapshot } from "@/lib/mocks/profile";
import { TrendingUp, Award } from "lucide-react";

interface CurrentRatingCardProps {
  rating: RatingSnapshot;
}

export function CurrentRatingCard({ rating }: CurrentRatingCardProps) {
  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm transition-colors overflow-hidden">
      <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between transition-colors">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">Current Rating</h3>
      </div>
      
      <div className="p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-amber-50 dark:bg-amber-500/10 flex items-center justify-center shrink-0 transition-colors">
              <Award className="h-5 w-5 text-amber-500 dark:text-amber-400" />
            </div>
            <div>
              <div className="text-2xl font-bold text-slate-900 dark:text-slate-100 transition-colors">{rating.currentRating}</div>
              <div className="text-sm font-medium text-amber-600 dark:text-amber-500 transition-colors">{rating.tier}</div>
            </div>
          </div>
          
          <div className="text-right">
            <div className="text-sm text-slate-500 dark:text-slate-400 transition-colors">Top {100 - rating.percentile}%</div>
            <div className="inline-flex items-center text-xs font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-500/10 px-1.5 py-0.5 rounded mt-1 transition-colors">
              <TrendingUp className="h-3 w-3 mr-1" />
              +{rating.ratingChange}
            </div>
          </div>
        </div>
        
        {/* Simple CSS mock chart for rating history */}
        <div className="h-14 w-full flex items-end justify-between gap-1 mt-5 opacity-70">
          {[40, 45, 55, 50, 60, 65, 55, 70, 75, 80, 85, 90, 85, 95, 100].map((height, i) => (
            <div 
              key={i} 
              className="w-full bg-violet-200 dark:bg-violet-900/40 rounded-t-sm transition-colors"
              style={{ height: `${height}%` }}
            >
              {i === 14 && (
                <div className="w-full h-1 bg-violet-600 dark:bg-violet-500 rounded-t-sm"></div>
              )}
            </div>
          ))}
        </div>
        
        <div className="flex items-center justify-between mt-3 text-xs text-slate-500 dark:text-slate-400 transition-colors">
          <span>Peak: <span className="font-medium text-slate-700 dark:text-slate-300 transition-colors">{rating.peakRating}</span></span>
          <span>15 contests</span>
        </div>
      </div>
    </div>
  );
}
