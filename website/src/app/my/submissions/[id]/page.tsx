"use client";

import { use, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { submissionApi } from "@/lib/api-client";
import { useFetch } from "@/hooks/useFetch";
import { useAuth } from "@/lib/auth-context";
import { StatusBadge } from "@/components/ui/Badge";
import {
  RefreshCw,
  AlertCircle,
  ArrowLeft,
  Clock,
  Server,
  Code2,
} from "lucide-react";

export default function MySubmissionDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { state: authState } = useAuth();

  const {
    status,
    data: sub,
    error,
    refetch,
  } = useFetch(
    () => submissionApi.getMyDetail(parseInt(id, 10)),
    [id],
    { enabled: authState.status === "AUTHENTICATED" }
  );

  // Redirect unauthenticated users
  useEffect(() => {
    if (authState.status === "IDLE" || authState.status === "ERROR") {
      router.push("/auth/login");
    }
  }, [authState.status, router]);

  // Poll while judging
  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (sub && (sub.status === "PENDING" || sub.status === "JUDGING")) {
      interval = setInterval(() => {
        refetch();
      }, 2000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [sub, refetch]);

  if (
    authState.status !== "AUTHENTICATED" ||
    status === "LOADING" ||
    status === "IDLE"
  ) {
    return (
      <PageLayout className="flex items-center justify-center">
        <RefreshCw className="animate-spin text-[var(--oj-accent)]" size={32} />
      </PageLayout>
    );
  }

  if (status === "ERROR" || !sub) {
    return (
      <PageLayout className="flex flex-col items-center justify-center text-center">
        <AlertCircle size={48} className="text-[var(--oj-wa-txt)] mb-4" />
        <h2 className="text-2xl font-bold text-[var(--oj-text)] mb-2">
          Submission Not Found
        </h2>
        <p className="text-[var(--oj-muted)] mb-6">
          {error || "The requested submission does not exist."}
        </p>
        <Link
          href="/my/submissions"
          className="px-4 py-2 rounded-md bg-[var(--oj-accent)] text-white hover:bg-[var(--oj-accent-dk)] transition-colors inline-flex items-center gap-2"
        >
          <ArrowLeft size={16} />
          Back to Submissions
        </Link>
      </PageLayout>
    );
  }

  return (
    <PageLayout>
      {/* Back link */}
      <div className="mb-6">
        <Link
          href="/my/submissions"
          className="text-[var(--oj-muted)] hover:text-[var(--oj-text)] inline-flex items-center gap-2 transition-colors text-sm font-medium"
        >
          <ArrowLeft size={16} />
          Back to Submissions
        </Link>
      </div>

      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-end gap-4 mb-8">
        <div>
          <h1 className="text-3xl font-bold text-[var(--oj-text)] mb-2 flex items-center gap-3">
            Submission #{sub.id}
            {(sub.status === "PENDING" || sub.status === "JUDGING") && (
              <RefreshCw
                className="animate-spin text-[var(--oj-accent)]"
                size={20}
              />
            )}
          </h1>
          <div className="flex items-center gap-2 text-[var(--oj-muted)]">
            <span>Problem:</span>
            <span className="text-[var(--oj-accent)] font-medium">
              {sub.problem_name}
            </span>
          </div>
        </div>
        <div className="text-right text-sm text-[var(--oj-muted)]">
          Submitted on {new Date(sub.created_at).toLocaleString()}
        </div>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl p-5 flex flex-col items-center justify-center text-center col-span-1 md:col-span-2 shadow-sm">
          <div className="text-sm text-[var(--oj-muted)] mb-3 font-semibold uppercase tracking-wider">
            Status
          </div>
          <StatusBadge status={sub.status} className="text-lg px-4 py-1.5" />
        </div>

        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl p-5 flex flex-col items-center justify-center text-center shadow-sm">
          <div className="flex items-center gap-2 text-sm text-[var(--oj-muted)] mb-2 font-semibold uppercase tracking-wider">
            <Clock size={16} /> Time
          </div>
          <div className="text-2xl font-bold text-[var(--oj-text)] font-code">
            {sub.execution_time_ms ?? "-"}{" "}
            <span className="text-base font-normal text-[var(--oj-muted)]">
              ms
            </span>
          </div>
        </div>

        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl p-5 flex flex-col items-center justify-center text-center shadow-sm">
          <div className="flex items-center gap-2 text-sm text-[var(--oj-muted)] mb-2 font-semibold uppercase tracking-wider">
            <Server size={16} /> Memory
          </div>
          <div className="text-2xl font-bold text-[var(--oj-text)] font-code">
            {sub.memory_used_kb ?? "-"}{" "}
            <span className="text-base font-normal text-[var(--oj-muted)]">
              KB
            </span>
          </div>
        </div>
      </div>

      {/* Compile output */}
      {sub.compile_output && (
        <div className="mb-8">
          <h2 className="text-lg font-bold text-[var(--oj-text)] mb-3">
            Compiler Output
          </h2>
          <div className="bg-[var(--oj-code-bg)] text-[var(--oj-code-txt)] p-4 rounded-xl border border-[var(--oj-border)] shadow-sm font-code text-sm overflow-x-auto whitespace-pre">
            {sub.compile_output}
          </div>
        </div>
      )}

      {/* Failed test case */}
      {sub.failed_test && (
        <div className="mb-8">
          <h2 className="text-lg font-bold text-[var(--oj-wa-txt)] mb-3 flex items-center gap-2">
            <AlertCircle size={20} />
            Failed Test Case {sub.failed_test.test_index}
          </h2>
          <div className="border border-[var(--oj-wa-txt)]/30 rounded-xl overflow-hidden shadow-sm bg-[var(--oj-surface)]">
            <div className="p-4 space-y-4 font-code text-sm">
              <div>
                <div className="text-[var(--oj-muted)] mb-1 font-semibold font-sans">
                  Input:
                </div>
                <div className="bg-[var(--oj-bg)] p-3 rounded-lg border border-[var(--oj-border)] overflow-x-auto whitespace-pre text-[var(--oj-code-txt)]">
                  {sub.failed_test.input || "<hidden>"}
                </div>
              </div>
              <div>
                <div className="text-[var(--oj-muted)] mb-1 font-semibold font-sans">
                  Expected Output:
                </div>
                <div className="bg-[var(--oj-bg)] p-3 rounded-lg border border-[var(--oj-border)] overflow-x-auto whitespace-pre text-[var(--oj-code-txt)]">
                  {sub.failed_test.expected_output || "<hidden>"}
                </div>
              </div>
              <div>
                <div className="text-[var(--oj-wa-txt)] mb-1 font-semibold font-sans">
                  Actual Output:
                </div>
                <div className="bg-[var(--oj-wa-bg)] p-3 rounded-lg border border-[var(--oj-wa-txt)]/30 text-[var(--oj-wa-txt)] overflow-x-auto whitespace-pre">
                  {sub.failed_test.actual_output || "<hidden>"}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Source code */}
      <div>
        <h2 className="text-lg font-bold text-[var(--oj-text)] mb-3 flex items-center gap-2">
          <Code2 size={20} />
          Source Code ({sub.language})
        </h2>
        <div className="bg-[var(--oj-code-bg)] text-[var(--oj-code-txt)] p-4 rounded-xl border border-[var(--oj-border)] shadow-sm overflow-x-auto">
          <pre className="font-code text-sm leading-relaxed whitespace-pre">
            {sub.source_code}
          </pre>
        </div>
      </div>
    </PageLayout>
  );
}
