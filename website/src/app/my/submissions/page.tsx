"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { submissionApi } from "@/lib/api-client";
import { useFetch } from "@/hooks/useFetch";
import { useAuth } from "@/lib/auth-context";
import { StatusBadge } from "@/components/ui/Badge";
import { TableRowSkeleton } from "@/components/ui/Skeleton";
import { RefreshCw, AlertCircle } from "lucide-react";
import type { SubmissionStatus, Language } from "@/types/api";

const STATUSES: SubmissionStatus[] = [
  "ACCEPTED", "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED",
  "RUNTIME_ERROR", "COMPILATION_ERROR", "SYSTEM_ERROR", "PENDING", "JUDGING",
];

const LANGUAGES: Language[] = ["CPP", "PYTHON", "JAVA", "GO", "JAVASCRIPT", "C"];

export default function MySubmissionsPage() {
  const router = useRouter();
  const { state: authState } = useAuth();

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [languageFilter, setLanguageFilter] = useState<string>("");

  const { status, data, error, refetch } = useFetch(
    () =>
      submissionApi.listMy({
        page,
        limit: 20,
        status: statusFilter,
        language: languageFilter,
      }),
    [page, statusFilter, languageFilter],
    { enabled: authState.status === "AUTHENTICATED" }
  );

  useEffect(() => {
    if (authState.status === "IDLE" || authState.status === "ERROR") {
      router.push("/auth/login");
    }
  }, [authState.status, router]);

  if (authState.status !== "AUTHENTICATED") {
    return (
      <PageLayout className="flex items-center justify-center">
        <RefreshCw className="animate-spin text-[var(--oj-accent)]" size={32} />
      </PageLayout>
    );
  }

  return (
    <PageLayout>
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-8 gap-4">
        <div>
          <h1 className="text-3xl font-bold text-[var(--oj-text)]">
            My Submissions
          </h1>
          <p className="text-[var(--oj-muted)] mt-1">
            History of your problem attempts.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-md text-sm px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)] text-[var(--oj-text)] min-w-[140px]"
          >
            <option value="">All Statuses</option>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s.replace(/_/g, " ")}
              </option>
            ))}
          </select>

          <select
            value={languageFilter}
            onChange={(e) => {
              setLanguageFilter(e.target.value);
              setPage(1);
            }}
            className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-md text-sm px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)] text-[var(--oj-text)] min-w-[140px]"
          >
            <option value="">All Languages</option>
            {LANGUAGES.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>

          <button
            onClick={refetch}
            className="cursor-pointer p-2 rounded-md border border-[var(--oj-border)] bg-[var(--oj-surface)] hover:bg-[var(--oj-panel)] transition-colors text-[var(--oj-muted)] hover:text-[var(--oj-text)]"
            title="Refresh"
            aria-label="Refresh submissions"
          >
            <RefreshCw
              size={18}
              className={status === "LOADING" ? "animate-spin" : ""}
            />
          </button>
        </div>
      </div>

      <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-[var(--oj-panel)] border-b border-[var(--oj-border)]">
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-24">
                  ID
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)]">
                  Problem
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-32">
                  Status
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-24">
                  Language
                </th>
                <th className="px-6 py-4 text-sm font-semibold text-[var(--oj-body)] w-40 text-right">
                  Time
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--oj-border)]">
              {status === "LOADING" || status === "IDLE" ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRowSkeleton key={i} cols={5} />
                ))
              ) : status === "ERROR" ? (
                <tr>
                  <td
                    colSpan={5}
                    className="px-6 py-8 text-center text-[var(--oj-wa-txt)]"
                  >
                    <div className="flex flex-col items-center justify-center gap-2">
                      <AlertCircle size={24} />
                      {error || "Failed to load your submissions"}
                    </div>
                  </td>
                </tr>
              ) : data?.items.length === 0 ? (
                <tr>
                  <td
                    colSpan={5}
                    className="px-6 py-12 text-center text-[var(--oj-muted)]"
                  >
                    You haven&apos;t made any submissions yet.
                  </td>
                </tr>
              ) : (
                data?.items.map((sub) => (
                  <tr
                    key={sub.id}
                    className="hover:bg-[var(--oj-bg)] transition-colors group"
                  >
                    <td className="px-6 py-4 text-sm text-[var(--oj-muted)] font-code">
                      <Link
                        href={`/my/submissions/${sub.id}`}
                        className="group-hover:text-[var(--oj-accent)] transition-colors"
                      >
                        #{sub.id}
                      </Link>
                    </td>
                    <td className="px-6 py-4">
                      <span className="font-medium text-[var(--oj-text)] line-clamp-1">
                        {sub.problem_name}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <Link href={`/my/submissions/${sub.id}`}>
                        <StatusBadge status={sub.status} />
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--oj-muted)] font-code">
                      {sub.language}
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--oj-muted)] text-right whitespace-nowrap">
                      {new Date(sub.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

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
