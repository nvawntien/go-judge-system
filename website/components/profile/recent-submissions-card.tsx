import { RecentSubmission, SubmissionResult } from "@/lib/mocks/profile";


interface RecentSubmissionsCardProps {
  submissions: RecentSubmission[];
}



function getStatusColor(result: SubmissionResult) {
  switch (result) {
    case "Accepted":
      return "text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-500/10 border-emerald-100 dark:border-emerald-500/20";
    case "Wrong Answer":
      return "text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-500/10 border-red-100 dark:border-red-500/20";
    case "Time Limit":
      return "text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10 border-amber-100 dark:border-amber-500/20";
    case "Runtime Error":
      return "text-violet-600 dark:text-violet-400 bg-violet-50 dark:bg-violet-500/10 border-violet-100 dark:border-violet-500/20";
    default:
      return "text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-800 border-slate-100 dark:border-slate-700";
  }
}

function getDifficultyDotClass(difficulty: "Easy" | "Medium" | "Hard") {
  switch (difficulty) {
    case "Easy":
      return "bg-emerald-500 dark:bg-emerald-400";
    case "Medium":
      return "bg-amber-500 dark:bg-amber-400";
    case "Hard":
      return "bg-red-500 dark:bg-red-400";
    default:
      return "bg-slate-500 dark:bg-slate-400";
  }
}

export function RecentSubmissionsCard({ submissions }: RecentSubmissionsCardProps) {
  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800/80 shadow-sm overflow-hidden transition-colors h-full flex flex-col">
      <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between transition-colors">
        <h2 className="font-semibold text-slate-900 dark:text-slate-100">Recent Submissions</h2>
        <a href="#" className="text-xs font-medium text-violet-600 dark:text-violet-400 hover:underline transition-colors">
          View all
        </a>
      </div>
      
      <div className="overflow-x-auto">
        <table className="w-full text-sm text-left">
          <thead className="bg-slate-50 dark:bg-slate-900/50 text-slate-500 dark:text-slate-400 border-b border-slate-100 dark:border-slate-800 transition-colors">
            <tr>
              <th className="px-4 py-2.5 font-medium">Problem</th>
              <th className="px-4 py-2.5 font-medium">Language</th>
              <th className="px-4 py-2.5 font-medium">Result</th>
              <th className="px-4 py-2.5 font-medium text-right">Time</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800/80 transition-colors">
            {submissions.map((sub) => (
              <tr key={sub.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
                <td className="px-4 py-2.5 align-middle">
                  <div className="flex items-center gap-2.5">
                    <span className={`h-2.5 w-2.5 rounded-full shrink-0 ${getDifficultyDotClass(sub.difficulty)}`} aria-hidden="true" />
                    <button 
                      type="button"
                      className="font-medium text-slate-900 dark:text-slate-200 hover:text-violet-600 dark:hover:text-violet-400 text-left transition-colors truncate max-w-[180px] sm:max-w-[280px]"
                      title={sub.problemTitle}
                    >
                      {sub.problemTitle}
                    </button>
                  </div>
                </td>
                <td className="px-4 py-2.5 align-middle text-slate-600 dark:text-slate-400">
                  {sub.language}
                </td>
                <td className="px-4 py-2.5 align-middle">
                  <div className={`inline-flex items-center justify-center min-w-[100px] px-2 py-0.5 rounded-md border text-xs font-medium transition-colors ${getStatusColor(sub.result)}`}>
                    {sub.result}
                  </div>
                </td>
                <td className="px-4 py-2.5 align-middle text-right text-slate-500 dark:text-slate-400">
                  {sub.submittedAt}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {submissions.length === 0 && (
        <div className="p-8 text-center text-slate-500 dark:text-slate-400">
          No recent submissions found.
        </div>
      )}
    </div>
  );
}
