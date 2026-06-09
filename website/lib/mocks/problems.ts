export type ProblemDifficulty = "Easy" | "Medium" | "Hard";
export type ProblemStatus = "Solved" | "Attempted" | "Unsolved";

export interface Problem {
  id: string;
  title: string;
  slug: string;
  difficulty: ProblemDifficulty;
  status: ProblemStatus;
  acceptanceRate: number;
  tags: string[];
  solvedCount: number;
  submissionCount: number;
}

export const MOCK_PROBLEMS: Problem[] = [
  {
    id: "1",
    title: "Two Sum",
    slug: "two-sum",
    difficulty: "Easy",
    status: "Solved",
    acceptanceRate: 52.3,
    tags: ["Array", "Hash Table"],
    solvedCount: 154320,
    submissionCount: 295066,
  },
  {
    id: "2",
    title: "Add Two Numbers",
    slug: "add-two-numbers",
    difficulty: "Medium",
    status: "Attempted",
    acceptanceRate: 41.8,
    tags: ["Linked List", "Math"],
    solvedCount: 89400,
    submissionCount: 213875,
  },
  {
    id: "3",
    title: "Longest Substring Without Repeating Characters",
    slug: "longest-substring",
    difficulty: "Medium",
    status: "Unsolved",
    acceptanceRate: 34.5,
    tags: ["Hash Table", "String", "Sliding Window"],
    solvedCount: 110200,
    submissionCount: 319420,
  },
  {
    id: "4",
    title: "Median of Two Sorted Arrays",
    slug: "median-two-sorted-arrays",
    difficulty: "Hard",
    status: "Unsolved",
    acceptanceRate: 38.1,
    tags: ["Array", "Binary Search", "Divide and Conquer"],
    solvedCount: 34500,
    submissionCount: 90551,
  },
  {
    id: "5",
    title: "Longest Palindromic Substring",
    slug: "longest-palindromic-substring",
    difficulty: "Medium",
    status: "Solved",
    acceptanceRate: 33.2,
    tags: ["String", "DP"],
    solvedCount: 67800,
    submissionCount: 204216,
  },
  {
    id: "6",
    title: "Merge Intervals",
    slug: "merge-intervals",
    difficulty: "Medium",
    status: "Unsolved",
    acceptanceRate: 47.0,
    tags: ["Array", "Sorting"],
    solvedCount: 56700,
    submissionCount: 120638,
  },
  {
    id: "7",
    title: "Trapping Rain Water",
    slug: "trapping-rain-water",
    difficulty: "Hard",
    status: "Attempted",
    acceptanceRate: 60.1,
    tags: ["Array", "Two Pointers", "DP", "Stack"],
    solvedCount: 42100,
    submissionCount: 70049,
  },
  {
    id: "8",
    title: "Best Time to Buy and Sell Stock",
    slug: "best-time-to-buy-and-sell-stock",
    difficulty: "Easy",
    status: "Solved",
    acceptanceRate: 54.8,
    tags: ["Array", "DP"],
    solvedCount: 125600,
    submissionCount: 229197,
  },
  {
    id: "9",
    title: "Word Ladder",
    slug: "word-ladder",
    difficulty: "Hard",
    status: "Unsolved",
    acceptanceRate: 37.9,
    tags: ["Hash Table", "String", "BFS"],
    solvedCount: 28900,
    submissionCount: 76253,
  },
  {
    id: "10",
    title: "Climbing Stairs",
    slug: "climbing-stairs",
    difficulty: "Easy",
    status: "Solved",
    acceptanceRate: 53.0,
    tags: ["Math", "DP", "Memoization"],
    solvedCount: 140500,
    submissionCount: 265094,
  }
];
