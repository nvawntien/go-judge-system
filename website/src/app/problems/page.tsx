"use client";

import { useState } from "react";
import Link from "next/link";
import { PageLayout } from "@/components/layout/PageLayout";
import { problemApi } from "@/lib/api-client";
import { useFetch } from "@/hooks/useFetch";
import { DifficultyBadge } from "@/components/ui/Badge";
import { TableRowSkeleton } from "@/components/ui/Skeleton";
import { Search } from "lucide-react";

export default function ProblemsPage() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [difficulty, setDifficulty] = useState<string>("");

  const { status, data, error } = useFetch(
    () => problemApi.list({ page, limit: 20, search, difficulty }),
    [page, search, difficulty]
  );

  return (
    <PageLayout>
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-8 gap-4">
        <div>
          <h1 className="text-3xl font-bold text-[var(--oj-text)]">Problems</h1>
          <p className="text-[var(--oj-muted)] mt-1">
            Practice and improve your algorithmic skills.
          </p>
        </div>

        <div className="flex items-center gap-3 w-full md:w-auto">
          <div className="relative flex-1 md:w-64">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--oj-muted)]"
              size={16}
            />
            <input
              type="text"
              placeholder="Search problems..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              className="w-full pl-9 pr-4 py-2 bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-md text-sm text-[var(--oj-text)] placeholder:text-[var(--oj-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)]"
            />
          </div>
          <select
            value={difficulty}
            onChange={(e) => {
              setDifficulty(e.target.value);
              setPage(1);
            }}
            className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-md text-sm px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)] text-[var(--oj-text)]"
          >
            <option value="">All Difficulties</option>
            <option value="EASY">Easy</option>
            <option value="MEDIUM">Medium</option>
            <option value="HARD">Hard</option>
          </select>
        </div>
      </div>

      <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-[var(--oj-panel)] border-b border-[var(--oj-border)]">
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)]">
                  Title
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-32">
                  Difficulty
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-32 text-right">
                  Time Limit
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-32 text-right">
                  Memory Limit
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--oj-border)]">
              {status === "LOADING" || status === "IDLE" ? (
                <>
                  <TableRowSkeleton cols={4} />
                  <TableRowSkeleton cols={4} />
                  <TableRowSkeleton cols={4} />
                  <TableRowSkeleton cols={4} />
                  <TableRowSkeleton cols={4} />
                </>
              ) : status === "ERROR" ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-6 py-8 text-center text-[var(--oj-wa-txt)]"
                  >
                    {error || "Failed to load problems"}
                  </td>
                </tr>
              ) : data?.items.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-6 py-12 text-center text-[var(--oj-muted)]"
                  >
                    No problems found matching your criteria.
                  </td>
                </tr>
              ) : (
                data?.items.map((problem) => (
                  <tr
                    key={problem.id}
                    className="hover:bg-[var(--oj-bg)] transition-colors group"
                  >
                    <td className="px-6 py-4">
                      <Link
                        href={`/problems/${problem.slug}`}
                        className="font-medium text-[var(--oj-text)] group-hover:text-[var(--oj-accent)] transition-colors"
                      >
                        {problem.title}
                      </Link>
                    </td>
                    <td className="px-6 py-4">
                      <DifficultyBadge difficulty={problem.difficulty} />
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--oj-muted)] text-right font-code">
                      {problem.time_limit}s
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--oj-muted)] text-right font-code">
                      {problem.memory_limit}MB
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination */}
      {data && data.total > data.limit && (
        <div className="flex justify-center mt-8 gap-2">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            className="cursor-pointer px-4 py-2 rounded-md border border-[var(--oj-border)] bg-[var(--oj-surface)] text-sm font-medium disabled:opacity-50 hover:bg-[var(--oj-panel)] text-[var(--oj-text)]"
          >
            Previous
          </button>
          <span className="px-4 py-2 text-sm text-[var(--oj-muted)]">
            Page {page} of {Math.ceil(data.total / data.limit)}
          </span>
          <button
            onClick={() => setPage((p) => p + 1)}
            disabled={page >= Math.ceil(data.total / data.limit)}
            className="cursor-pointer px-4 py-2 rounded-md border border-[var(--oj-border)] bg-[var(--oj-surface)] text-sm font-medium disabled:opacity-50 hover:bg-[var(--oj-panel)] text-[var(--oj-text)]"
          >
            Next
          </button>
        </div>
      )}
    </PageLayout>
  );
}
