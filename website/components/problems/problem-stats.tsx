import { StatCard } from "@/components/shared/stat-card"
import { MOCK_PROBLEMS } from "@/lib/mocks/problems"

export function ProblemStats() {
  const total = MOCK_PROBLEMS.length;
  const solved = MOCK_PROBLEMS.filter(p => p.status === "Solved").length;
  const attempted = MOCK_PROBLEMS.filter(p => p.status === "Attempted").length;
  
  // Calculate average acceptance rate of solved problems as a mock stat
  const avgAcceptance = MOCK_PROBLEMS.reduce((acc, curr) => acc + curr.acceptanceRate, 0) / (total || 1);

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-8">
      <StatCard 
        title="Total Problems" 
        value={total} 
        description="Available to practice" 
      />
      <StatCard 
        title="Solved" 
        value={solved} 
        description="Completed successfully" 
      />
      <StatCard 
        title="Attempted" 
        value={attempted} 
        description="In progress" 
      />
      <StatCard 
        title="Avg. Acceptance" 
        value={`${avgAcceptance.toFixed(1)}%`} 
        description="Global acceptance rate" 
      />
    </div>
  );
}
