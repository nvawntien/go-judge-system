export type SubmissionResult =
  | "Accepted"
  | "Wrong Answer"
  | "Time Limit"
  | "Runtime Error";

export type AchievementRarity = "Common" | "Rare" | "Epic";

export interface ProfileUser {
  id: string;
  displayName: string;
  username: string;
  avatarUrl: string;
  country: string;
  location: string;
  bio: string;
  joinedAt: string;
  organization: string;
  rating: number;
  globalRank: number;
}

export interface ProfileStats {
  solvedProblems: number;
  contestsJoined: number;
  acceptanceRate: number;
  globalRank: number;
  currentStreak: number;
}

export interface DifficultyProgress {
  difficulty: "Easy" | "Medium" | "Hard";
  solved: number;
  total: number;
}

export interface RecentSubmission {
  id: string;
  problemTitle: string;
  problemSlug: string;
  difficulty: "Easy" | "Medium" | "Hard";
  language: string;
  result: SubmissionResult;
  submittedAt: string;
}

export interface Achievement {
  id: string;
  title: string;
  description: string;
  rarity: AchievementRarity;
}

export interface FavoriteTopic {
  name: string;
  solvedCount: number;
}

export interface RatingSnapshot {
  currentRating: number;
  peakRating: number;
  ratingChange: number;
  tier: string;
  percentile: number;
}

export interface ActivityCell {
  date: string;
  level: 0 | 1 | 2 | 3 | 4;
  count: number;
}

export interface ProfilePageData {
  user: ProfileUser;
  stats: ProfileStats;
  solvedByDifficulty: DifficultyProgress[];
  recentSubmissions: RecentSubmission[];
  achievements: Achievement[];
  favoriteTopics: FavoriteTopic[];
  rating: RatingSnapshot;
  activity: ActivityCell[];
}

// -----------------------------------------------------------------------------
// Mock Data
// -----------------------------------------------------------------------------

function generateActivityData(): ActivityCell[] {
  const cells: ActivityCell[] = [];
  const today = new Date();
  
  // Generate a year of activity
  for (let i = 365; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(d.getDate() - i);
    
    // Randomize activity level
    const rand = Math.random();
    let level: 0 | 1 | 2 | 3 | 4 = 0;
    let count = 0;
    
    if (rand > 0.85) {
      level = 4;
      count = Math.floor(Math.random() * 5) + 8;
    } else if (rand > 0.7) {
      level = 3;
      count = Math.floor(Math.random() * 4) + 4;
    } else if (rand > 0.5) {
      level = 2;
      count = Math.floor(Math.random() * 3) + 2;
    } else if (rand > 0.3) {
      level = 1;
      count = 1;
    }
    
    cells.push({
      date: d.toISOString().split("T")[0],
      level,
      count,
    });
  }
  return cells;
}

export const MOCK_PROFILE_DATA: ProfilePageData = {
  user: {
    id: "usr_alex123",
    displayName: "Alex",
    username: "@alex_codes",
    avatarUrl: "",
    country: "India",
    location: "India",
    bio: "Passionate about algorithms and clean code. Always learning, always building.",
    joinedAt: "2022-05-15T00:00:00Z",
    organization: "IIIT Hyderabad",
    rating: 1874,
    globalRank: 2451,
  },
  stats: {
    solvedProblems: 342,
    contestsJoined: 56,
    acceptanceRate: 67.4,
    globalRank: 2451,
    currentStreak: 30,
  },
  solvedByDifficulty: [
    { difficulty: "Easy", solved: 154, total: 200 },
    { difficulty: "Medium", solved: 130, total: 300 },
    { difficulty: "Hard", solved: 58, total: 160 },
  ],
  recentSubmissions: [
    {
      id: "sub_1",
      problemTitle: "Two Sum",
      problemSlug: "two-sum",
      difficulty: "Easy",
      language: "Go",
      result: "Accepted",
      submittedAt: "2 hours ago",
    },
    {
      id: "sub_2",
      problemTitle: "LRU Cache",
      problemSlug: "lru-cache",
      difficulty: "Medium",
      language: "Go",
      result: "Accepted",
      submittedAt: "5 hours ago",
    },
    {
      id: "sub_3",
      problemTitle: "Merge K Sorted Lists",
      problemSlug: "merge-k-sorted-lists",
      difficulty: "Hard",
      language: "Go",
      result: "Wrong Answer",
      submittedAt: "1 day ago",
    },
    {
      id: "sub_4",
      problemTitle: "Trapping Rain Water",
      problemSlug: "trapping-rain-water",
      difficulty: "Hard",
      language: "Python",
      result: "Time Limit",
      submittedAt: "2 days ago",
    },
    {
      id: "sub_5",
      problemTitle: "Valid Parentheses",
      problemSlug: "valid-parentheses",
      difficulty: "Easy",
      language: "TypeScript",
      result: "Accepted",
      submittedAt: "3 days ago",
    },
  ],
  achievements: [
    {
      id: "ach_1",
      title: "30-Day Streak",
      description: "Solved at least one problem for 30 consecutive days.",
      rarity: "Epic",
    },
    {
      id: "ach_2",
      title: "Top 10%",
      description: "Ranked in the top 10% of users.",
      rarity: "Rare",
    },
    {
      id: "ach_3",
      title: "Fast Solver",
      description: "Submitted a correct solution within 10 minutes of reading.",
      rarity: "Common",
    },
    {
      id: "ach_4",
      title: "100 Solves",
      description: "Solved 100 problems.",
      rarity: "Common",
    },
  ],
  favoriteTopics: [
    { name: "Array", solvedCount: 134 },
    { name: "Graph", solvedCount: 98 },
    { name: "DP", solvedCount: 87 },
    { name: "Binary Search", solvedCount: 76 },
    { name: "Greedy", solvedCount: 64 },
    { name: "String", solvedCount: 48 },
  ],
  rating: {
    currentRating: 1874,
    peakRating: 1923,
    ratingChange: +64,
    tier: "Expert",
    percentile: 95.0,
  },
  activity: generateActivityData(),
};
