import { FavoriteTopic } from "@/lib/mocks/profile";
import Link from "next/link";

interface FavoriteTopicsCardProps {
  topics: FavoriteTopic[];
}

export function FavoriteTopicsCard({ topics }: FavoriteTopicsCardProps) {
  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm transition-colors overflow-hidden">
      <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between transition-colors">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">Favorite Topics</h3>
        <Link href="#" className="text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors">
          Edit
        </Link>
      </div>
      
      <div className="p-4 flex flex-wrap gap-2">
        {topics.map((topic) => (
          <div 
            key={topic.name}
            className="inline-flex items-center px-2.5 py-1 rounded-full bg-slate-50 dark:bg-slate-800 border border-slate-100 dark:border-slate-700 text-sm transition-colors"
          >
            <span className="text-slate-700 dark:text-slate-300 font-medium transition-colors">{topic.name}</span>
            <span className="ml-1.5 text-xs text-slate-400 dark:text-slate-500 transition-colors">{topic.solvedCount}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
