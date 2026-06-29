import { MOCK_PROFILE_DATA } from "@/lib/mocks/profile";
import { ProfileHeroCard } from "@/components/profile/profile-hero-card";
import { RecentSubmissionsCard } from "@/components/profile/recent-submissions-card";
import { ProblemSolvingCard } from "@/components/profile/problem-solving-card";
import { RecentActivityGrid } from "@/components/profile/recent-activity-grid";
import { AchievementsCard } from "@/components/profile/achievements-card";
import { FavoriteTopicsCard } from "@/components/profile/favorite-topics-card";
import { CurrentRatingCard } from "@/components/profile/current-rating-card";

export default function ProfilePage() {
  const data = MOCK_PROFILE_DATA;

  return (
    <div className="w-full pb-12">
      {/* 2-Column Layout */}
      <div className="grid min-w-0 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_340px] gap-6 items-start">
        {/* Main Content Column */}
        <div className="space-y-6 min-w-0">
          <ProfileHeroCard user={data.user} stats={data.stats} />
          
          <div className="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.9fr)]">
            <RecentSubmissionsCard submissions={data.recentSubmissions} />
            <ProblemSolvingCard stats={data.stats} difficultyProgress={data.solvedByDifficulty} />
          </div>

          <RecentActivityGrid activity={data.activity} />
        </div>
        
        {/* Right Sidebar */}
        <div className="space-y-6">

          <CurrentRatingCard rating={data.rating} />
          <AchievementsCard achievements={data.achievements} />
          <FavoriteTopicsCard topics={data.favoriteTopics} />
        </div>
      </div>
    </div>
  );
}
