import { Achievement } from "@/lib/mocks/profile";
import { Medal, Star, Shield, Trophy } from "lucide-react";
import Link from "next/link";

interface AchievementsCardProps {
  achievements: Achievement[];
}

export function AchievementsCard({ achievements }: AchievementsCardProps) {
  // Use different icons based on rarity just for variation
  const getIcon = (rarity: string) => {
    switch (rarity) {
      case "Epic": return <Trophy className="h-4 w-4 text-amber-500" />;
      case "Rare": return <Medal className="h-4 w-4 text-violet-500" />;
      case "Common": return <Star className="h-4 w-4 text-emerald-500" />;
      default: return <Shield className="h-4 w-4 text-slate-500" />;
    }
  };

  const getIconBg = (rarity: string) => {
    switch (rarity) {
      case "Epic": return "bg-amber-50 dark:bg-amber-500/10";
      case "Rare": return "bg-violet-50 dark:bg-violet-500/10";
      case "Common": return "bg-emerald-50 dark:bg-emerald-500/10";
      default: return "bg-slate-50 dark:bg-slate-800";
    }
  };

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm transition-colors overflow-hidden">
      <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between transition-colors">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">Achievements</h3>
        <Link href="#" className="text-xs font-medium text-violet-600 dark:text-violet-400 hover:underline transition-colors">
          View all
        </Link>
      </div>
      
      <div className="p-4 grid grid-cols-2 gap-3">
        {achievements.slice(0, 4).map((achievement) => (
          <div key={achievement.id} className="flex flex-col gap-2 p-3 rounded-lg border border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50 transition-colors">
            <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 transition-colors ${getIconBg(achievement.rarity)}`}>
              {getIcon(achievement.rarity)}
            </div>
            <div>
              <h4 className="text-xs font-semibold text-slate-900 dark:text-slate-100 leading-tight transition-colors">{achievement.title}</h4>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
