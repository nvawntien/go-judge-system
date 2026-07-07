import { SolvedProblemsCard } from "./solved-problems-card"
import { DifficultyDistributionCard } from "./difficulty-distribution-card"
import { DailyChallengeCard } from "./daily-challenge-card"

export function ProblemPageSidebar() {
  return (
    <div className="w-full">
      <SolvedProblemsCard />
      <DifficultyDistributionCard />
      <DailyChallengeCard />
    </div>
  )
}
