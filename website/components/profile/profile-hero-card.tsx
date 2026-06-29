import { MapPin, CalendarDays, Building2, BadgeCheck, Share2, Edit3, User, Trophy, Target, Flame, Star } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ProfileUser, ProfileStats } from "@/lib/mocks/profile"

interface ProfileHeroCardProps {
  user: ProfileUser
  stats: ProfileStats
}

export function ProfileHeroCard({ user, stats }: ProfileHeroCardProps) {
  const joinedDate = new Date(user.joinedAt).toLocaleDateString("en-US", {
    month: "long",
    year: "numeric",
  })

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm transition-colors dark:border-slate-800/80 dark:bg-slate-900">
      <div className="p-4">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
          <div className="shrink-0">
            <div className="flex h-20 w-20 items-center justify-center rounded-full border border-slate-200 bg-slate-50 text-slate-300 transition-colors dark:border-slate-700 dark:bg-slate-800 dark:text-slate-600">
              <User className="h-10 w-10" />
            </div>
          </div>

          <div className="flex-1 space-y-2.5">
            <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
              <div>
                <h1 className="flex items-center gap-2 text-2xl font-bold text-slate-900 transition-colors dark:text-slate-100">
                  {user.displayName}
                  <BadgeCheck className="h-5 w-5 text-violet-600 dark:text-violet-500" />
                </h1>
                <p className="font-medium text-slate-500 transition-colors dark:text-slate-400">{user.username}</p>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="border-slate-200 bg-white text-slate-700 transition-colors hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900/50 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  <Edit3 className="mr-2 h-4 w-4" />
                  Edit profile
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="border-slate-200 bg-white text-slate-700 transition-colors hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900/50 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  <Share2 className="h-4 w-4" />
                </Button>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2.5">
              <div className="inline-flex items-center rounded-md bg-amber-50 px-2.5 py-1 text-sm font-medium text-amber-700 transition-colors dark:bg-amber-500/10 dark:text-amber-400">
                <Trophy className="mr-1.5 h-3.5 w-3.5" />
                Global Rank: #{user.globalRank}
              </div>
            </div>

            <p className="text-sm leading-relaxed text-slate-600 transition-colors dark:text-slate-300">
              {user.bio}
            </p>

            <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-slate-500 transition-colors dark:text-slate-400">
              <div className="flex items-center gap-1.5">
                <MapPin className="h-4 w-4" />
                <span>{user.country}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Building2 className="h-4 w-4" />
                <span>{user.organization}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <CalendarDays className="h-4 w-4" />
                <span>Joined {joinedDate}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="border-t border-slate-100 bg-slate-50/50 transition-colors dark:border-slate-800/80 dark:bg-slate-900/20">
        <div className="grid grid-cols-2 divide-x divide-y divide-slate-100 dark:divide-slate-800/80 md:grid-cols-5 md:divide-y-0">
          <div className="flex flex-col items-center justify-center gap-1 px-3 py-2.5 text-center">
            <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Star className="h-3.5 w-3.5 text-violet-500" /> Rating
            </span>
            <span className="text-xl font-bold text-slate-900 dark:text-slate-100">{user.rating}</span>
          </div>
          <div className="flex flex-col items-center justify-center gap-1 px-3 py-2.5 text-center">
            <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Trophy className="h-3.5 w-3.5 text-amber-500" /> Contests Joined
            </span>
            <span className="text-xl font-bold text-slate-900 dark:text-slate-100">{stats.contestsJoined}</span>
          </div>
          <div className="flex flex-col items-center justify-center gap-1 px-3 py-2.5 text-center">
            <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Target className="h-3.5 w-3.5 text-blue-500" /> Acceptance Rate
            </span>
            <span className="text-xl font-bold text-slate-900 dark:text-slate-100">{stats.acceptanceRate}%</span>
          </div>
          <div className="flex flex-col items-center justify-center gap-1 px-3 py-2.5 text-center">
            <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Trophy className="h-3.5 w-3.5 text-slate-400" /> Global Rank
            </span>
            <span className="text-xl font-bold text-slate-900 dark:text-slate-100">#{stats.globalRank}</span>
          </div>
          <div className="flex flex-col items-center justify-center gap-1 px-3 py-2.5 text-center md:border-l-0">
            <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Flame className="h-3.5 w-3.5 text-orange-500" /> Current Streak
            </span>
            <span className="text-xl font-bold text-slate-900 dark:text-slate-100">{stats.currentStreak} days</span>
          </div>
        </div>
      </div>
    </div>
  )
}
